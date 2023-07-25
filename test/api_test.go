package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

type State struct {
	Name                       string `json:"name"`
	Namespace                  string `json:"namespace"`
	ControllerKind             string `json:"controller_kind"`
	ContainerName              string `json:"container_name"`
	TracesInstrumented         bool   `json:"traces_instrumented"`
	TracesInstrumentable       bool   `json:"traces_instrumentable"`
	Application                string `json:"application"`
	ServiceName                string `json:"service_name"`
	Language                   string `json:"language"`
	LogType                    string `json:"log_type"`
	OpenTelemetryPreconfigured bool   `json:"opentelemetry_preconfigured"`
	DetectionStatus            string `json:"detection_status"`
}

type AnnotateRequest struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	ControllerKind string `json:"controller_kind"`
	LogType        string `json:"log_type"`
	ContainerName  string `json:"container_name"`
	ServiceName    string `json:"service_name"`
}
type AnnotateResponse struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	ControllerKind string `json:"controller_kind"`
	ServiceName    string `json:"service_name"`
	ContainerName  string `json:"container_name"`
	LogType        string `json:"log_type"`
}

type TestCase struct {
	Description string
	Modify      func(s *State) *AnnotateRequest
}

func TestApi(t *testing.T) {
	// Define test cases
	testCases := []TestCase{
		{
			Description: "Add service name and logType",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        s.Name,
					ContainerName:  s.ContainerName,
					ServiceName:    s.Name,
				}
			},
		},
		{
			Description: "Update service name",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        s.Name,
					ContainerName:  s.ContainerName,
					ServiceName:    s.Name + s.Namespace,
				}
			},
		},
		{
			Description: "Update log type",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        s.Name + s.Namespace,
					ContainerName:  s.ContainerName,
					ServiceName:    s.Name + s.Namespace,
				}
			},
		},
		{
			Description: "remove service name and log type",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        "",
					ContainerName:  s.ContainerName,
					ServiceName:    "",
				}
			},
		},
		{
			Description: "add log type",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        s.Name,
					ContainerName:  s.ContainerName,
					ServiceName:    "",
				}
			},
		},
		{
			Description: "add service name",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        s.Name,
					ContainerName:  s.ContainerName,
					ServiceName:    s.Name + s.ContainerName,
				}
			},
		},
		{
			Description: "remove log type",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        "",
					ContainerName:  s.ContainerName,
					ServiceName:    s.Name + s.ContainerName,
				}
			},
		},
		{
			Description: "remove service name",
			Modify: func(s *State) *AnnotateRequest {
				return &AnnotateRequest{
					Name:           s.Name,
					Namespace:      s.Namespace,
					ControllerKind: s.ControllerKind,
					LogType:        "",
					ContainerName:  s.ContainerName,
					ServiceName:    "",
				}
			},
		},
	}

	host := "localhost"
	port := "5050"
	client := &http.Client{}
	stateUrl := fmt.Sprintf("http://%s:%s/api/v1/state", host, port)
	annotateUrl := fmt.Sprintf("http://%s:%s/api/v1/annotate", host, port)

	// Perform GET request
	resp, err := client.Get(stateUrl)
	if err != nil {
		t.Fatalf("Failed to perform GET request: %v", err)
	}
	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Unmarshal the response into a slice of state objects
	var states []State
	err = json.Unmarshal(body, &states)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Loop over test cases
	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			log.Println("Case:", tc.Description)

			// Loop over each state object, perform annotate request and validate the response
			for _, state := range states {
				if !state.TracesInstrumentable {
					continue
				}

				annotate := tc.Modify(&state)

				annotateReqBody, err := json.Marshal(annotate)
				if err != nil {
					t.Fatalf("Failed to marshal annotate request body: %v", err)
				}

				annotateReq, err := http.NewRequest("POST", annotateUrl, bytes.NewBuffer(annotateReqBody))
				if err != nil {
					t.Fatalf("Failed to create annotate request: %v", err)
				}
				annotateReq.Header.Set("Content-Type", "application/json")

				annotateResp, err := client.Do(annotateReq)
				if err != nil {
					t.Fatalf("Failed to perform annotate request: %v", err)
				}
				annotateRespBody, err := ioutil.ReadAll(annotateResp.Body)
				if err != nil {
					t.Fatalf("Failed to read annotate response body: %v", err)
				}

				var annotateRespBodyStruct AnnotateResponse
				err = json.Unmarshal(annotateRespBody, &annotateRespBodyStruct)
				if err != nil {
					t.Fatalf("Failed to unmarshal annotate response: %v", err)
				}

				// Now you can access the fields in the response struct
				assert.Equal(t, annotateRespBodyStruct.Name, annotate.Name)
				assert.Equal(t, annotateRespBodyStruct.ServiceName, annotate.ServiceName)
				assert.Equal(t, annotateRespBodyStruct.LogType, annotate.LogType)
				assert.Equal(t, annotateRespBodyStruct.ControllerKind, annotate.ControllerKind)
				assert.Equal(t, annotateRespBodyStruct.Namespace, annotate.Namespace)
				assert.Equal(t, annotateRespBodyStruct.ContainerName, annotate.ContainerName)

				// Validate that the state has changed after annotation
				respAfterAnnotate, err := client.Get(stateUrl)
				if err != nil {
					t.Fatalf("Failed to perform GET request after annotation: %v", err)
				}
				bodyAfterAnnotate, err := ioutil.ReadAll(respAfterAnnotate.Body)
				if err != nil {
					t.Fatalf("Failed to read response body after annotation: %v", err)
				}

				var statesAfterAnnotate []State
				err = json.Unmarshal(bodyAfterAnnotate, &statesAfterAnnotate)
				if err != nil {
					t.Fatalf("Failed to unmarshal response after annotation: %v", err)
				}
				// compare the response to the actual new state
				for _, newState := range statesAfterAnnotate {
					if newState.Name == annotate.Name {
						log.Println("comparing: " + annotate.Name)
						log.Printf("Service name: actual=%s requested=%s", annotate.ServiceName, newState.ServiceName)
						log.Printf("Log type: actual=%s requested=%s", annotate.LogType, newState.LogType)
						// Check that the state has changed accordingly
						assert.Equal(t, newState.ServiceName, annotate.ServiceName)
						assert.Equal(t, newState.LogType, annotate.LogType)
						assert.Equal(t, newState.Namespace, annotate.Namespace)
						assert.Equal(t, newState.ControllerKind, annotate.ControllerKind)
						assert.Equal(t, newState.ContainerName, annotate.ContainerName)

					}
				}
			}
		})
	}
}
