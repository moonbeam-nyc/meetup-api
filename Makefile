.PHONY: help build tag push docker-release docker-public k8s-apply k8s-delete k8s-restart k8s-logs k8s-status dev dev-build dev-logs dev-down test-server go-run go-build go-fmt

# Container image configuration
REGISTRY := ghcr.io/moonbeam-nyc
IMAGE_NAME := meetup-api
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
IMAGE := $(REGISTRY)/$(IMAGE_NAME)

# Kubernetes namespace
NAMESPACE := meetup-api

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Docker targets
build: ## Build the Docker image
	docker build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

tag: ## Tag the image with version and latest
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest

push: ## Push the image to GitHub Container Registry
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

docker-public: ## Instructions to make the image public (API doesn't work for org packages)
	@echo "To make $(IMAGE_NAME) public on ghcr.io:"
	@echo "1. Go to https://github.com/orgs/moonbeam-nyc/packages/container/$(IMAGE_NAME)"
	@echo "2. Click 'Package settings' (on the right)"
	@echo "3. Scroll to 'Danger Zone' → Change visibility → Public"
	@echo ""
	@echo "Note: GitHub's API doesn't support changing org package visibility"

docker-release: build push ## Build and push Docker image with latest tag

# Kubernetes targets
k8s-apply: ## Apply all Kubernetes manifests
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/ingress.yaml

k8s-delete: ## Delete all Kubernetes resources
	kubectl delete -f k8s/ingress.yaml --ignore-not-found
	kubectl delete -f k8s/service.yaml --ignore-not-found
	kubectl delete -f k8s/deployment.yaml --ignore-not-found
	kubectl delete -f k8s/custom-headers.yaml --ignore-not-found
	kubectl delete -f k8s/namespace.yaml --ignore-not-found

k8s-restart: ## Restart the deployment
	kubectl rollout restart deployment/$(IMAGE_NAME) -n $(NAMESPACE)

k8s-logs: ## View logs from the deployment
	kubectl logs -f deployment/$(IMAGE_NAME) -n $(NAMESPACE)

k8s-status: ## Check deployment status
	kubectl get all -n $(NAMESPACE)

# Development targets
dev: ## Run development server in Docker
	docker compose up

dev-build: ## Build and run development server in Docker
	docker compose up --build

dev-logs: ## View docker compose logs
	docker compose logs -f

dev-down: ## Stop docker compose services
	docker compose down

test-server: ## Test server endpoints in Docker
	@./test-server.sh

# Go development targets (if you have Go installed locally)
go-run: ## Run Go server locally
	cd cmd/server && go run .

go-build: ## Build Go binary locally
	go build -o meetup-api ./cmd/server

go-fmt: ## Format Go code
	go fmt ./...
