package main

import (
	"fmt"
	"github.com/gorilla/mux"
	annotateapi "github.com/logzio/ezkonnect-server/api/annotate"
	stateapi "github.com/logzio/ezkonnect-server/api/state"
	"log"
	"net/http"
)

// main starts the server. Endpoints:
// 1. /api/v1/state - returns a list of all custom resources of type InstrumentedApplication
// 2. /api/v1/annotate - handles the POST request for annotating a supported resource kind
func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/v1/state", stateapi.GetCustomResourcesHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotate", annotateapi.UpdateResourceAnnotations).Methods(http.MethodPost)
	fmt.Println("Starting server on :5050")
	log.Fatal(http.ListenAndServe(":5050", router))
}
