.PHONY: build-extproc-budget build-extproc-ratelimit build-ui-server test lint clean docker-build-extproc-budget docker-build-extproc-ratelimit docker-build-ui docker-build-all push-extproc-budget push-extproc-ratelimit push-ui push-all deploy-all deploy-extproc-budget deploy-extproc-ratelimit deploy-ui undeploy-all migrate port-forward-all port-forward-extproc port-forward-ui help ui-dev

# Variables
IMAGE_REPO?=
IMAGE_PREFIX := $(if $(IMAGE_REPO),$(IMAGE_REPO)/,)
VERSION?=latest
NAMESPACE?=agentgateway-system
PLATFORMS?=linux/amd64,linux/arm64

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# UI development server (with proxy to backend)
ui-dev:
	cd ui && bun dev

# Test
test:
	$(GOTEST) -v -race ./...

test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...

fmt:
	$(GOFMT) -w -s .

# Clean
clean:
	rm -rf bin/
	rm -rf ui/dist/
	rm -f coverage.out coverage.html

# Dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy
	cd ui && bun install

# Build ext-proc binaries
build-extproc-budget:
	$(GOBUILD) -o bin/quota-budget-extproc -v ./cmd/extproc-budget

build-extproc-ratelimit:
	$(GOBUILD) -o bin/quota-ratelimit-extproc -v ./cmd/extproc-ratelimit

# Build UI server binary
build-ui-server:
	$(GOBUILD) -o bin/quota-ui -v ./cmd/ui

# Docker builds
docker-build-extproc-budget:
	docker buildx build --platform $(PLATFORMS) --load -f Dockerfile.extproc-budget -t $(IMAGE_PREFIX)quota-budget-extproc:$(VERSION) .

docker-build-extproc-ratelimit:
	docker buildx build --platform $(PLATFORMS) --load -f Dockerfile.extproc-ratelimit -t $(IMAGE_PREFIX)quota-ratelimit-extproc:$(VERSION) .

docker-build-ui:
	docker buildx build --platform $(PLATFORMS) --load -f Dockerfile.ui -t $(IMAGE_PREFIX)quota-management-ui:$(VERSION) .

docker-build-all: docker-build-extproc-budget docker-build-extproc-ratelimit docker-build-ui

push-extproc-budget: ## Build and push multi-arch quota-budget-extproc image
	docker buildx build --platform $(PLATFORMS) \
		--push \
		-t $(IMAGE_PREFIX)quota-budget-extproc:$(VERSION) \
		-f Dockerfile.extproc-budget .

push-extproc-ratelimit: ## Build and push multi-arch quota-ratelimit-extproc image
	docker buildx build --platform $(PLATFORMS) \
		--push \
		-t $(IMAGE_PREFIX)quota-ratelimit-extproc:$(VERSION) \
		-f Dockerfile.extproc-ratelimit .

push-ui: ## Build and push multi-arch quota-management-ui image
	docker buildx build --platform $(PLATFORMS) \
		--push \
		-t $(IMAGE_PREFIX)quota-management-ui:$(VERSION) \
		-f Dockerfile.ui .

push-all: push-extproc-budget push-extproc-ratelimit push-ui ## Build and push all multi-arch images

# Deploy - split container mode
deploy-all:
	kubectl apply -f deploy/postgres.yaml
	@echo "Waiting for PostgreSQL to be ready..."
	kubectl wait --for=condition=ready pod -l app=quota-management-postgres -n $(NAMESPACE) --timeout=120s
	$(MAKE) deploy-extproc-budget
	$(MAKE) deploy-extproc-ratelimit
	sed 's|image: quota-management-ui:latest|image: $(IMAGE_PREFIX)quota-management-ui:$(VERSION)|' \
		deploy/ui-deployment.yaml | kubectl apply -f -
	kubectl apply -f deploy/ui-service.yaml

deploy-extproc-budget:
	sed 's|image: quota-budget-extproc:latest|image: $(IMAGE_PREFIX)quota-budget-extproc:$(VERSION)|' \
		deploy/extproc-deployment.yaml | kubectl apply -f -
	kubectl apply -f deploy/extproc-service.yaml

deploy-extproc-ratelimit:
	sed 's|image: quota-ratelimit-extproc:latest|image: $(IMAGE_PREFIX)quota-ratelimit-extproc:$(VERSION)|' \
		deploy/ratelimit-deployment.yaml | kubectl apply -f -
	kubectl apply -f deploy/ratelimit-service.yaml

deploy-ui:
	sed 's|image: quota-management-ui:latest|image: $(IMAGE_PREFIX)quota-management-ui:$(VERSION)|' \
		deploy/ui-deployment.yaml | kubectl apply -f -
	kubectl apply -f deploy/ui-service.yaml

undeploy-all:
	kubectl delete -f deploy/extproc-deployment.yaml --ignore-not-found
	kubectl delete -f deploy/extproc-service.yaml --ignore-not-found
	kubectl delete -f deploy/ui-deployment.yaml --ignore-not-found
	kubectl delete -f deploy/ui-service.yaml --ignore-not-found
	kubectl delete -f deploy/postgres.yaml --ignore-not-found

# Database migrations
migrate:
	@echo "Running migrations..."
	PGPASSWORD=budget psql -h localhost -U budget -d budget_management -f migrations/001_initial.sql
	PGPASSWORD=budget psql -h localhost -U budget -d budget_management -f migrations/002_match_expression_index.sql

# Port forward - split container mode
port-forward-all:
	@echo "Forwarding ext-proc (4444) and UI (8080)..."
	kubectl port-forward -n $(NAMESPACE) svc/quota-management-extproc 4444:4444 9090:9090 &
	kubectl port-forward -n $(NAMESPACE) svc/quota-management-ui 8080:8080 9091:9091

port-forward-extproc:
	kubectl port-forward -n $(NAMESPACE) svc/quota-management-extproc 4444:4444 9090:9090

port-forward-ui:
	kubectl port-forward -n $(NAMESPACE) svc/quota-management-ui 8080:8080 9091:9091

# Help
help:
	@echo "Quota Management"
	@echo ""
	@echo "Usage:"
	@echo "  make build-extproc-budget    Build budget ext-proc binary"
	@echo "  make build-extproc-ratelimit Build ratelimit ext-proc binary"
	@echo "  make build-ui-server         Build UI server binary"
	@echo "  make ui-dev                  Start UI development server"
	@echo "  make test                    Run tests"
	@echo "  make lint                    Run linter"
	@echo "  make clean                   Clean build artifacts"
	@echo "  make deps                    Download all dependencies (Go + UI)"
	@echo "  make push-extproc-budget     Push quota-budget-extproc image to registry"
	@echo "  make push-extproc-ratelimit  Push quota-ratelimit-extproc image to registry"
	@echo "  make push-ui                 Push quota-management-ui image to registry"
	@echo "  make push-all                Push all 3 images to registry"
	@echo "  make deploy-all              Deploy all components to Kubernetes"
	@echo "  make deploy-extproc-budget   Deploy budget ext-proc"
	@echo "  make deploy-extproc-ratelimit Deploy ratelimit ext-proc"
	@echo "  make deploy-ui               Deploy UI server"
	@echo "  make undeploy-all            Remove all from Kubernetes"
	@echo "  make migrate                 Run database migrations"
	@echo "  make port-forward-all        Port forward all services"
	@echo "  make help                    Show this help"
