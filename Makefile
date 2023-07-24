IMAGE_NAME := ezkonnect-server
IMAGE_TAG := v1.0.6
DOCKER_REPO := logzio/$(IMAGE_NAME):$(IMAGE_TAG)
K8S_NAMESPACE := ezkonnect

.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_REPO) .

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_REPO)

.PHONY: deploy-kubectl
deploy-kubectl:
	kubectl apply -f deploy/k8s-manifest.yaml -n $(K8S_NAMESPACE)

.PHONY: clean-kubectl
clean-kubectl:
	kubectl delete -f deploy/k8s-manifest.yaml -n $(K8S_NAMESPACE)

.PHONY: local-server
local-server:
	go run main.go

.PHONY: test-api-clean
test-api-clean:
	kubectl delete -f test/demoServices.yaml

.PHONY: test-api-deploy
test-api-deploy:
	kubectl apply -f test/demoServices.yaml

.PHONY: test-api
test-api:
	go test ./test

