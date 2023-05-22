package state

import (
	"context"
	"encoding/json"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// InstrumentdApplicationData is the data structure for the custom resource
// the response will contain a list of these fields
// name: the name of the custom resource
// namespace: the namespace of the custom resource
// controller_kind: the kind of the controller that created the custom resource
// container_name: the name of the container
// instrumented: whether the container is instrumented or not
// application: the name of the application that the container belongs to
// language: the language of the application that the container belongs to

type InstrumentdApplicationData struct {
	Name                string  `json:"name"`
	Namespace           string  `json:"namespace"`
	ControllerKind      string  `json:"controller_kind"`
	ContainerName       *string `json:"container_name"`
	TracesInstrumented  bool    `json:"traces_instrumented"`
	MetricsInstrumented bool    `json:"metrics_instrumented"`
	Application         *string `json:"application"`
	Language            *string `json:"language"`
}

// GetCustomResourcesHandler lists all custom resources of type InstrumentedApplication
func GetCustomResourcesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	config, err := getConfig()
	if err != nil {
		log.Fatalf("Error getting Kubernetes config: %v", err)
	}
	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating dynamic client: %v", err)
	}
	gvr := schema.GroupVersionResource{
		Group:    "logz.io",
		Version:  "v1alpha1",
		Resource: "instrumentedapplications",
	}
	// List all custom resources
	instrumentedApplicationsList, err := dynamicClient.Resource(gvr).Namespace("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing custom resources: %v", err)
	}
	// Build a list of InstrumentdApplicationData from the custom resources
	var data []InstrumentdApplicationData
	for _, item := range instrumentedApplicationsList.Items {
		name := item.GetName()
		namespace := item.GetNamespace()
		ControllerKind := strings.ToLower(item.GetOwnerReferences()[0].Kind)
		status := item.Object["status"]
		spec := item.Object["spec"]
		// Check if the languages field is present in the spec
		languages, langOk := spec.(map[string]interface{})["languages"].([]interface{})
		if langOk {
			// Handle the languages field
			for _, language := range languages {
				langStr := language.(map[string]interface{})["language"].(string)
				containerNameStr := language.(map[string]interface{})["containerName"].(string)
				entry := InstrumentdApplicationData{
					Name:                name,
					Namespace:           namespace,
					ControllerKind:      ControllerKind,
					TracesInstrumented:  status.(map[string]interface{})["tracesInstrumented"].(bool),
					MetricsInstrumented: status.(map[string]interface{})["metricsInstrumented"].(bool),
					ContainerName:       &containerNameStr,
					Language:            &langStr,
				}
				data = append(data, entry)
			}
		}
		// Check if the applications field is present in the spec
		applications, appOk := spec.(map[string]interface{})["applications"].([]interface{})
		// Handle the applications field
		if appOk {
			for _, application := range applications {
				applicationStr := application.(map[string]interface{})["application"].(string)
				containerNameStr := application.(map[string]interface{})["containerName"].(string)
				entry := InstrumentdApplicationData{
					Name:                name,
					Namespace:           namespace,
					ControllerKind:      ControllerKind,
					TracesInstrumented:  status.(map[string]interface{})["tracesInstrumented"].(bool),
					MetricsInstrumented: status.(map[string]interface{})["metricsInstrumented"].(bool),
					ContainerName:       &containerNameStr,
					Application:         &applicationStr,
				}
				data = append(data, entry)
			}
		}
		// Handle the case where the languages and applications fields are not present in the spec
		if !langOk && !appOk {
			entry := InstrumentdApplicationData{
				Name:                name,
				Namespace:           namespace,
				ControllerKind:      ControllerKind,
				TracesInstrumented:  status.(map[string]interface{})["tracesInstrumented"].(bool),
				MetricsInstrumented: status.(map[string]interface{})["metricsInstrumented"].(bool),
			}
			data = append(data, entry)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func getConfig() (*rest.Config, error) {
	var config *rest.Config

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}