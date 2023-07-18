# Ezkonnect-server
Ezkonnect server is a web server written in go that exposes an API for Ezkonnect. It is responsible for managing the state of instrumented applications.
This server is designed to run in a kubernetes environment, However it can run locally as well if a `kubeconfig` file connected to a kubernetes cluster is present on your machine.
### getting started
1. Clone the repo
2. Run `make server-local` to start the server
3. The server will be running on `localhost:5050`

### API
**Full API docs can be found [Here](./api.md)**
- Get the state Instrumented Applications `[GET] /api/v1/state`

This endpoint retrieves information about instrumented applications in the form of custom resources of type InstrumentedApplication.

- Update traces resource annotations `[POST] /api/v1/annotate`

This endpoint allows you to update annotations for Kubernetes deployments and statefulsets. The annotations can be used to enable or disable telemetry features such as traces auto instrumentation and log type.


### development
- run `make server-local` to start the server
- run `make docker-build` to build the docker image
- run `make docker-push` to push the docker image to the registry
- run `deploy-kubectl` to deploy the server to your cluster
- run `clean-kubectl` to delete the server from your cluster


## changelog
- v1.0.5
  - Unify logs and traces endpoint
  - Validate instrumentation status change and log type before returning a 200 response
  - Remove action from the request model
  - Add `REQUESTTIMEOUT_SECONDS` env var
  - Add `instrumetable` flag
- v1.0.4
  - Add support for opentelemetry detection
  - update api docs
- v1.0.3
  - Delete `logz.io/application_type` annotation while rolling back
- v1.0.2
  - Ignore internal resources
- v1.0.1
  - Add support for adding service name
- v1.0.0 - Initial release
  - A web server written in go that exposes an API for Ezkonnect. It is responsible for managing the state of instrumented applications.
