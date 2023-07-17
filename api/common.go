package api

import (
	"fmt"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	KindDeployment             = "deployment"
	KindStatefulSet            = "statefulSet"
	ActionAdd                  = "add"
	ActionDelete               = "delete"
	ErrorDecodeJSON            = "Error decoding JSON body "
	ErrorKubeConfig            = "Error getting Kubernetes config "
	ErrorKubeClient            = "Error creating Kubernetes clientset "
	ErrorInvalidInput          = "Invalid input "
	ErrorDynamic               = "Error getting dynamic client "
	ErrorUpdate                = "Error updating resource "
	ErrorUpdateLogType         = "Error updating resource log type "
	ErrorUpdateInstrumentation = "Error updating resource instrumentation "
	ErrorGet                   = "Error getting resource "
	ErrorList                  = "Error listing resources "
	ErrorTimeout               = "Timeout while updating the instrumentation status: "

	ResourceGroup                   = "logz.io"
	ResourceVersion                 = "v1alpha1"
	ResourceInstrumentedApplication = "instrumentedapplications"

	LogzioServiceAnnotationName = "logz.io/service-name"
)

var (
	ValidKinds   = []string{KindDeployment, KindStatefulSet}
	ValidActions = []string{ActionAdd, ActionDelete}
)

// InitLogger initializes the logger
func InitLogger() zap.SugaredLogger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"} // write to stdout
	config.InitialFields = map[string]interface{}{}
	logger, configErr := config.Build()
	if configErr != nil {
		fmt.Printf("Error while initializing the logger: %v", configErr)
		panic(configErr)
	}
	return *logger.Sugar()
}

// GetTimeout returns the timeout for the request in time.Duration
func GetTimeout() (time.Duration, error) {
	var timeoutSeconds int
	var err error
	timeoutStr := os.Getenv("REQUEST_TIMEOUT_SECONDS")
	if timeoutStr == "" {
		// Default timeout is 90 seconds
		timeoutSeconds = 90
	} else {
		timeoutSeconds, err = strconv.Atoi(timeoutStr)
		if err != nil {
			return time.Duration(0), err
		}
	}
	return time.Duration(timeoutSeconds) * time.Second, nil
}

// DeepEqualMap compares two maps
func DeepEqualMap(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if vb, ok := b[k]; !ok || !reflect.DeepEqual(v, vb) {
			return false
		}
	}
	return true
}

// GetConfig returns a Kubernetes config
func GetConfig() (*rest.Config, error) {
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

func IsInternalResource(name string) bool {
	return strings.Contains(name, "ezkonnect") || (name == "kubernetes-instrumentor")
}
