## API Documentation
- ### `[GET] /api/v1/state` Get the state Instrumented Applications 
This endpoint retrieves information about instrumented applications in the form of custom resources of type InstrumentedApplication.

### Request
- Method: `GET`
- Path: `/api/v1/state`

### Response
### Success
- Status code: `200 OK`
- Content-Type: `application/json`

The response body will be a JSON array of objects, where each object contains the following fields:
- `name` (string): The name of the custom resource.
- `namespace` (string): The namespace of the custom resource.
- `controller_kind` (string): The kind of the controller (lowercased owner reference kind).
- `container_name` (string, optional): The container name associated with the instrumented application. Will be empty if both language and application fields are empty.
- `traces_instrumented` (bool): Whether the application is instrumented or not.
- `traces_instrumentable` (bool): Whether the application can be instrumented or not.
- `application` (string, optional): The application name if available in the spec.
- `language` (string, optional): The programming language if available in the spec.
- `log_type` (string, optional): The log type if available in the spec.
- `opentelemetry_preconfigured` bool: Whether the application has opentelemetry libraries or not.
- `detection_status` (string): The status of the detection process. Can be one of the following:
    - `pending`: The detection process has not started yet.
    - `Completed`: The detection process has completed successfully.
    - `Running`: The detection process is still running.
    - `error`: The detection process has failed.


Each instrumented application can have a `language` and/or an `application` field, or none of them. If neither `language` nor `application` is present, the application cannot be instrumented. If at least one of `language` or `application` fields is non-empty, there will also be a `container_name` field. However, if both language and application fields are empty, the `container_name` will be empty as well.


#### Example Success Response
```json
[
    {
        "name": "my-instrumented-app",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "app-container",
        "traces_instrumented": true,
        "traces_instrumentable": true,
        "application": null,
        "language": "python",
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "nginx"
    },
    {
        "name": "uninstrumented-app",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "",
        "traces_instrumented": false,
        "traces_instrumentable": false,
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "log"
    },
    {
        "name": "statefulset-with-app-detection",
        "namespace": "default",
        "controller_kind": "statefulset",
        "container_name": "app-container",
        "traces_instrumented": false,
        "traces_instrumentable": false,
        "application": "my-app",
        "language": null,
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "log2"
    },
    {
        "name": "deployment-with-language-detection",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "app-container",
        "traces_instrumented": false,
        "traces_instrumentable": true,
        "application": null,
        "language": "java",
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "nginx"
    }
]
```
### Errors
- Status code: `405 Method Not Allowed`

The request method is not GET.

Example error response:

```json
{
"error": "Invalid request method"
}
```
- Status code: `500 Internal Server Error`

There was an error processing the request, such as failing to interact with the Kubernetes cluster.

Example error response:

```json
{
"error": "Error message"
}
```


- ### POST /api/v1/annotate
This endpoint updates the annotations on a Kubernetes resource based on the given input.

## Request:
- path: `/api/v1/annotate`
- Method: `POST`

**Request JSON Object:**

*   `name` : \[string\] The name of the resource to be annotated.
*   `namespace` : \[string\] The namespace of the resource.
*   `controller_kind` : \[string\] The kind of controller that created the resource (either 'Deployment' or 'StatefulSet').
*   `log_type` : \[string, optional\] The log type of the application that the container belongs to.
*   `container_name` : \[string\] The name of the container associated with the request.
*   `service_name` : \[string, optional\] The desired service name for the application. If this field is empty, the instrumentation will be deleted.

### Success Response
**Condition:** If the annotations on the resource are successfully updated and the custom resource is updated.

**Code:** `200 OK`
**Content example:**

```
{     
  "name": "resource-name", 
  "namespace": "resource-namespace",     
  "controller_kind": "Deployment",     
  "service_name": "service-name",     
  "container_name": "container-name",     
  "log_type": "log-type" 
}
```

### Error Response

**Condition:** If the request is invalid.

**Code:** `400 Bad Request`

**Content example:**

```
{     
"error": "Invalid input" 
}
```

**Condition:** If there is an error when getting the Kubernetes configuration.

**Code:** `500 Internal Server Error`

**Content example:**


```
{     
"error": "Error in getting Kubernetes configuration" 
}
```

**Condition:** If there is an error when updating the resource.

**Code:** `500 Internal Server Error`

**Content example:**


```
{
"error": "Error in updating resource"
}
```

**Condition:** If there is a timeout error.

**Code:** `500 Internal Server Error`

**Content example:**


```
{
"error": "Timeout in updating resource"
}
```

Notes
-----

*   This endpoint requires a JSON request body with details about the resource to be annotated.
*   The `controller_kind` must be either 'Deployment' or 'StatefulSet'.
*   The server will respond with an HTTP 400 status if the `controller_kind` is invalid.
*   The `log_type` field is optional and is used to set the desired log type. If it is not provided, any existing log type annotation on the resource will be removed.
*   The `service_name` field is also optional. If it is provided, the server will set the service name and ensure that instrumentation is enabled. If the `service_name` is not provided, any existing service name annotation and instrumentation will be removed.
*   The server will respond with an HTTP 500 status if it encounters any errors while updating the resource.
*   The server will also respond with an HTTP 500 status if the operation times out.