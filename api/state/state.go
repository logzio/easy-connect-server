package state

import (
	"context"
	"encoding/json"
	"github.com/logzio/easy-connect-server/api"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"net/http"
	"strings"
)

// InstrumentdApplicationData is the data structure for the custom resource
// the response will contain a list of these fields
// name: the name of the custom resource
// namespace: the namespace of the custom resource
// controller_kind: the kind of the controller that created the custom resource
// container_name: the name of the container
// traces_instrumented: whether the container is instrumented or not
// traces_instrumentable: whether we can instrument the container or not
// application: the name of the application that the container belongs to
// language: the language of the application that the container belongs to
// detection_status: the status of the detection process
// log_type: the log type of the application that the container belongs to
type InstrumentdApplicationData struct {
	Name                       string  `json:"name"`
	Namespace                  string  `json:"namespace"`
	ControllerKind             string  `json:"controller_kind"`
	ContainerName              *string `json:"container_name"`
	TracesInstrumented         bool    `json:"traces_instrumented"`
	ServiceName                *string `json:"service_name"`
	TracesInstrumentable       bool    `json:"traces_instrumentable"`
	Application                *string `json:"application"`
	Language                   *string `json:"language"`
	DetectionStatus            string  `json:"detection_status"`
	OpentelemetryPreconfigured *bool   `json:"opentelemetry_preconfigured"`
	LogType                    *string `json:"log_type"`
}

// GetCustomResourcesHandler lists all custom resources of type InstrumentedApplication
func GetCustomResourcesHandler(w http.ResponseWriter, r *http.Request) {
	logger := api.InitLogger()
	defer logger.Sync()
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	config, err := api.GetConfig()
	if err != nil {
		logger.Error(api.ErrorKubeConfig, zap.Error(err))
		http.Error(w, api.ErrorKubeConfig+err.Error(), http.StatusInternalServerError)
		return
	}
	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Error(api.ErrorDynamic, zap.Error(err))
		http.Error(w, api.ErrorDynamic+err.Error(), http.StatusInternalServerError)
		return
	}
	gvr := schema.GroupVersionResource{
		Group:    api.ResourceGroup,
		Version:  api.ResourceVersion,
		Resource: api.ResourceInstrumentedApplication,
	}
	// List all custom resources
	instrumentedApplicationsList, err := dynamicClient.Resource(gvr).Namespace("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		logger.Error(api.ErrorList, zap.Error(err))
		http.Error(w, api.ErrorList+err.Error(), http.StatusInternalServerError)
		return
	}
	// Build a list of InstrumentdApplicationData from the custom resources
	var data []InstrumentdApplicationData
	for _, item := range instrumentedApplicationsList.Items {
		name := item.GetName()
		// Skip internal resources
		if api.IsInternalResource(name) {
			continue
		}
		namespace := item.GetNamespace()
		controllerKind := strings.ToLower(item.GetOwnerReferences()[0].Kind)
		status := item.Object["status"].(map[string]interface{})
		spec := item.Object["spec"].(map[string]interface{})
		logType := spec["logType"].(string)
		// Check if the languages field is present in the spec
		languages, langOk := spec["languages"].([]interface{})
		if langOk {
			// Handle the languages field
			for _, language := range languages {
				langStr := language.(map[string]interface{})["language"].(string)
				containerNameStr := language.(map[string]interface{})["containerName"].(string)
				// Handle the serviceName field, since this app can be instrumented
				serviceName := calculateServiceName(language)
				otelDetectedBool := language.(map[string]interface{})["opentelemetryPreconfigured"].(bool)
				entry := InstrumentdApplicationData{
					Name:                       name,
					Namespace:                  namespace,
					ControllerKind:             controllerKind,
					TracesInstrumented:         status["tracesInstrumented"].(bool),
					TracesInstrumentable:       true,
					ServiceName:                &serviceName,
					ContainerName:              &containerNameStr,
					Language:                   &langStr,
					DetectionStatus:            status["instrumentationDetection"].(map[string]interface{})["phase"].(string),
					LogType:                    &logType,
					OpentelemetryPreconfigured: &otelDetectedBool,
				}
				data = append(data, entry)
			}
		}
		// Check if the applications field is present in the spec
		applications, appOk := spec["applications"].([]interface{})
		// Handle the applications field
		if appOk {
			for _, application := range applications {
				applicationStr := application.(map[string]interface{})["application"].(string)
				containerNameStr := application.(map[string]interface{})["containerName"].(string)
				otelDetectedBool := false
				entry := InstrumentdApplicationData{
					Name:                       name,
					Namespace:                  namespace,
					ControllerKind:             controllerKind,
					TracesInstrumented:         status["tracesInstrumented"].(bool),
					TracesInstrumentable:       false,
					ContainerName:              &containerNameStr,
					Application:                &applicationStr,
					DetectionStatus:            status["instrumentationDetection"].(map[string]interface{})["phase"].(string),
					LogType:                    &logType,
					OpentelemetryPreconfigured: &otelDetectedBool,
				}
				data = append(data, entry)
			}
		}
		// Handle the case where the languages and applications fields are not present in the spec
		if !langOk && !appOk {
			otelDetectedBool := false
			entry := InstrumentdApplicationData{
				Name:                       name,
				Namespace:                  namespace,
				ControllerKind:             controllerKind,
				TracesInstrumented:         status["tracesInstrumented"].(bool),
				TracesInstrumentable:       false,
				DetectionStatus:            status["instrumentationDetection"].(map[string]interface{})["phase"].(string),
				LogType:                    &logType,
				OpentelemetryPreconfigured: &otelDetectedBool,
			}
			data = append(data, entry)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func calculateServiceName(service interface{}) string {
	if service.(map[string]interface{})["activeServiceName"] == nil {
		return ""
	}
	return service.(map[string]interface{})["activeServiceName"].(string)
}
