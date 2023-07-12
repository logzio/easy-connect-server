package annotate

import (
	"context"
	"encoding/json"
	"github.com/logzio/ezkonnect-server/api"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strings"
	"time"
)

const (
	LogTypeAnnotation         = "logz.io/application_type"
	InstrumentationAnnotation = "logz.io/traces_instrument"
	ServiceNameAnnotation     = "logz.io/service-name"
)

// ResourceAnnotateRequest is the JSON body of the POST request
// It contains the name, controller_kind, namespace, and log type of the resource
// name: name of the resource
// controller_kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// log_type: desired log type
// service_name: the desired service name for the application, should delete instrumentation if this filed is empty
type ResourceAnnotateRequest struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	ControllerKind string `json:"controller_kind"`
	LogType        string `json:"log_type,omitempty"`
	ServiceName    string `json:"service_name,omitempty"`
}

// ResourceAnnotateResponse is the data structure for the custom resource
// the response will contain a list of these fields
// name: the name of the custom resource
// namespace: the namespace of the custom resource
// controller_kind: the kind of the controller that created the custom resource
// log_type: the log type of the application that the container belongs to
// service_name: the updated service name

type ResourceAnnotateResponse struct {
	Name           string  `json:"name"`
	Namespace      string  `json:"namespace"`
	ControllerKind string  `json:"controller_kind"`
	ServiceName    *string `json:"service_name"`
	LogType        *string `json:"log_type"`
}

func UpdateResourceAnnotations(w http.ResponseWriter, r *http.Request) {
	logger := api.InitLogger()
	// Decode JSON body
	var resource ResourceAnnotateRequest
	err := json.NewDecoder(r.Body).Decode(&resource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the Kubernetes config
	config, err := api.GetConfig()
	if err != nil {
		logger.Error(api.ErrorKubeConfig, err)
		http.Error(w, api.ErrorKubeConfig, http.StatusInternalServerError)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Error(api.ErrorDynamic, zap.Error(err))
		http.Error(w, api.ErrorDynamic+err.Error(), http.StatusInternalServerError)
		return
	}
	// instrumented application crd scheme
	gvr := schema.GroupVersionResource{
		Group:    api.ResourceGroup,
		Version:  api.ResourceVersion,
		Resource: api.ResourceInstrumentedApplication,
	}
	// Validate input before updating resources to avoid changing resources and retuning an error
	validRequest := isValidResourceAnnotateRequest(resource)
	// if one of the requests is invalid, return an error
	if !validRequest {
		logger.Error(api.ErrorInvalidInput)
		http.Error(w, api.ErrorInvalidInput, http.StatusBadRequest)
		return
	}

	// Define timeout for the context
	ctxDuration, err := api.GetTimeout()
	if err != nil {
		logger.Error(api.ErrorInvalidInput, err)
		http.Error(w, api.ErrorInvalidInput+err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), ctxDuration)
	defer cancel()
	// Create a channel to signal about workload and crd updates
	updateCh := make(chan struct{})
	// Create a dynamic factory that watches for changes in the InstrumentedApplication CRD corresponding to the resource
	dynamicFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, 1*time.Second, resource.Namespace, func(options *v1.ListOptions) {
		options.FieldSelector = "metadata.name=" + resource.Name
	})
	crdInformer := dynamicFactory.ForResource(gvr)
	// watch for crd spec changes to indicate about log type changes
	crdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			newSpec := newObj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
			oldSpec := oldObj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
			if !api.DeepEqualMap(oldSpec, newSpec) {
				updateCh <- struct{}{} // Signal that the update occurred
			}
		},
	})
	// watch for crd status changes to indicate about instrumentation status change (instrument, rollback)
	crdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			newStatus := newObj.(*unstructured.Unstructured).Object["status"].(map[string]interface{})
			oldStatus := oldObj.(*unstructured.Unstructured).Object["status"].(map[string]interface{})
			if !api.DeepEqualMap(oldStatus, newStatus) {
				updateCh <- struct{}{} // Signal that the update occurred
			}
		},
	})
	// start watching for changes in crd
	dynamicFactory.Start(ctx.Done())

	// Create a shared informer factory for Deployments and statefulsets
	sharedFactory := informers.NewSharedInformerFactory(clientset, 1*time.Second)
	var sharedInformer cache.SharedIndexInformer
	switch resource.ControllerKind {
	case api.KindDeployment:
		// Create an informer for Deployments
		sharedInformer = sharedFactory.Apps().V1().Deployments().Informer()
	case api.KindStatefulSet:
		// Create an informer for StatefulSets
		sharedInformer = sharedFactory.Apps().V1().StatefulSets().Informer()
	}

	// handlers for update events on Deployments/StatefulSets
	sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Cast the updated object to a metav1.Object to access its annotations
			newMeta := newObj.(metav1.Object)
			oldMeta := oldObj.(metav1.Object)

			// Check if the ServiceNameAnnotation has changed
			if newMeta.GetAnnotations()[ServiceNameAnnotation] != oldMeta.GetAnnotations()[ServiceNameAnnotation] {
				logger.Info("ServiceNameAnnotation changed: " + resource.Name)
				updateCh <- struct{}{}
			}
		},
	})
	// start watching for changes in workload
	sharedFactory.Start(ctx.Done())

	// Create the response
	response := ResourceAnnotateResponse{
		Name:           resource.Name,
		Namespace:      resource.Namespace,
		ControllerKind: resource.ControllerKind,
		LogType:        &resource.LogType,
		ServiceName:    &resource.ServiceName,
	}
	// choose the instrumentation annotation value and value according to the service name
	actionValue := "true"
	if resource.ServiceName == "" {
		actionValue = "rollback"
	}
	instrumentationAnnotations := map[string]string{}
	instrumentationAnnotations[InstrumentationAnnotation] = actionValue
	// add service name annotation if exists
	if resource.ServiceName != "" {
		instrumentationAnnotations[ServiceNameAnnotation] = resource.ServiceName
	}

	// Update workload and custom resources
	switch resource.ControllerKind {
	case api.KindDeployment:
		logger.Info("Updating deployment: ", resource.Name)
		deployment, getErr := clientset.AppsV1().Deployments(resource.Namespace).Get(ctx, resource.Name, v1.GetOptions{})
		if getErr != nil {
			logger.Error(api.ErrorGet, getErr)
			http.Error(w, api.ErrorGet+getErr.Error(), http.StatusInternalServerError)
			return
		}
		// handle log type
		if deployment.Spec.Template.ObjectMeta.Annotations == nil {
			deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		if len(resource.LogType) != 0 {
			deployment.Spec.Template.ObjectMeta.Annotations[LogTypeAnnotation] = resource.LogType
		} else {
			delete(deployment.Spec.Template.ObjectMeta.Annotations, LogTypeAnnotation)
		}

		// handle traces instrumentation annotations
		for k, v := range instrumentationAnnotations {
			deployment.Spec.Template.ObjectMeta.Annotations[k] = v
		}

		_, err = clientset.AppsV1().Deployments(resource.Namespace).Update(ctx, deployment, v1.UpdateOptions{})
		if err != nil {
			logger.Error(api.ErrorUpdate, err)
			http.Error(w, api.ErrorUpdate+err.Error(), http.StatusInternalServerError)
			return
		}

	case api.KindStatefulSet:
		logger.Info("Updating statefulset: ", resource.Name)
		statefulSet, getErr := clientset.AppsV1().StatefulSets(resource.Namespace).Get(ctx, resource.Name, v1.GetOptions{})
		if getErr != nil {
			logger.Error(api.ErrorGet, getErr)
			http.Error(w, api.ErrorGet+getErr.Error(), http.StatusInternalServerError)
			return
		}

		if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
			statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		// handle logs
		if len(resource.LogType) != 0 {
			statefulSet.Spec.Template.ObjectMeta.Annotations[LogTypeAnnotation] = resource.LogType
		} else {
			delete(statefulSet.Spec.Template.ObjectMeta.Annotations, LogTypeAnnotation)
		}
		// handle traces instrumentation annotations
		for k, v := range instrumentationAnnotations {
			statefulSet.Spec.Template.ObjectMeta.Annotations[k] = v
		}

		_, err = clientset.AppsV1().StatefulSets(resource.Namespace).Update(ctx, statefulSet, v1.UpdateOptions{})
		if err != nil {
			logger.Error(api.ErrorUpdate, err)
			http.Error(w, api.ErrorUpdate+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Wait for the update to occur or timeout
	select {
	case <-updateCh:
		logger.Info("crd changed: ", resource.Name)

	case <-ctx.Done():
		logger.Error(api.ErrorTimeout + resource.Name)
		http.Error(w, api.ErrorTimeout+resource.Name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func isValidResourceAnnotateRequest(req ResourceAnnotateRequest) bool {
	for _, validKind := range api.ValidKinds {
		if req.ControllerKind == strings.ToLower(validKind) {
			return true
		}
	}
	return false
}
