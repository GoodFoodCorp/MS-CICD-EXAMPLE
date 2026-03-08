# Makefile - Auth Service CI/CD

.PHONY: help setup build test test-coverage run docker-build docker-push docker-run docker-stop docker-logs
.PHONY: terraform-init terraform-plan terraform-apply terraform-destroy terraform-output
.PHONY: k8s-apply k8s-status k8s-logs k8s-describe k8s-shell
.PHONY: deploy rollback monitor notify-test security-scan clean deps fmt lint ssh

# Variables
APP_NAME       = auth-service
DOCKER_USERNAME ?= $(shell echo $${DOCKER_USERNAME:-youruser})
VERSION        ?= latest
NAMESPACE      = production

# ANSI colors
BLUE   = \033[0;34m
GREEN  = \033[0;32m
YELLOW = \033[1;33m
RED    = \033[0;31m
NC     = \033[0m

# Help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# -------------------------------------------------------------------
# Setup & Configuration
# -------------------------------------------------------------------

setup: ## Initial setup (dependencies, AWS, SSH, terraform.tfvars)
	@chmod +x scripts/*.sh
	@./scripts/setup.sh

# -------------------------------------------------------------------
# Development
# -------------------------------------------------------------------

build: ## Build the Go binary
	@echo "[BUILD] Building $(APP_NAME)..."
	@go build -o $(APP_NAME) ./cmd/main.go
	@echo "[OK] Build successful: $(APP_NAME)"

test: ## Run all tests
	@echo "[TEST] Running tests..."
	@./run_tests.sh

test-coverage: ## Run tests with coverage report
	@echo "[TEST] Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out -o coverage.html
	@echo "[OK] Report generated: coverage.html"

run: ## Run the application locally
	@echo "[RUN] Starting application..."
	@go run ./cmd/main.go

fmt: ## Format Go code
	@echo "[FMT] Formatting code..."
	@go fmt ./...
	@echo "[OK] Code formatted"

lint: ## Lint Go code
	@echo "[LINT] Running golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run ./...

# -------------------------------------------------------------------
# Docker
# -------------------------------------------------------------------

docker-build: ## Build Docker image
	@echo "[DOCKER] Building image $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION)..."
	@docker build -t $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION) .
	@docker tag $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION) $(DOCKER_USERNAME)/$(APP_NAME):latest
	@echo "[OK] Image built"

docker-push: ## Push Docker image to registry
	@echo "[DOCKER] Pushing image..."
	@docker push $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION)
	@docker push $(DOCKER_USERNAME)/$(APP_NAME):latest
	@echo "[OK] Image pushed"

docker-run: ## Start containers locally via docker-compose
	@docker-compose up -d
	@echo "[OK] Containers started"

docker-stop: ## Stop containers
	@docker-compose down

docker-logs: ## Follow container logs
	@docker-compose logs -f

# -------------------------------------------------------------------
# Terraform
# -------------------------------------------------------------------

terraform-init: ## Initialize Terraform
	@cd terraform && terraform init

terraform-plan: ## Plan Terraform changes
	@cd terraform && terraform plan

terraform-apply: ## Apply Terraform infrastructure
	@cd terraform && terraform apply

terraform-destroy: ## Destroy all Terraform resources
	@cd terraform && terraform destroy

terraform-output: ## Show Terraform outputs
	@cd terraform && terraform output

# -------------------------------------------------------------------
# Kubernetes
# -------------------------------------------------------------------

k8s-apply: ## Apply all Kubernetes manifests
	@echo "[K8S] Applying manifests..."
	@kubectl apply -f k8s/namespace.yaml
	@kubectl apply -f k8s/deployment.yaml
	@echo "[OK] Manifests applied"

k8s-status: ## Show pod status
	@kubectl get pods -n $(NAMESPACE) -l app=$(APP_NAME) -o wide

k8s-logs: ## Follow pod logs
	@kubectl logs -f -n $(NAMESPACE) -l app=$(APP_NAME) --tail=100

k8s-describe: ## Describe the deployment
	@kubectl describe deployment $(APP_NAME) -n $(NAMESPACE)

k8s-shell: ## Open a shell in a running pod
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pods -n $(NAMESPACE) -l app=$(APP_NAME) -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

# -------------------------------------------------------------------
# Deployment
# -------------------------------------------------------------------

deploy: ## Deploy application (make deploy VERSION=x.y.z)
	@./scripts/deploy.sh $(VERSION)

rollback: ## Rollback the deployment
	@./scripts/rollback.sh

monitor: ## Open monitoring dashboard
	@./scripts/monitor.sh

notify-test: ## Test notification webhook
	@./scripts/notify.sh

# -------------------------------------------------------------------
# Security
# -------------------------------------------------------------------

security-scan: ## Run govulncheck and gosec
	@echo "[SECURITY] Running govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...
	@echo "[SECURITY] Running gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec ./...
	@echo "[OK] Security scan complete"

# -------------------------------------------------------------------
# Utilities
# -------------------------------------------------------------------

clean: ## Remove generated files (binary, coverage, test cache)
	@echo "[CLEAN] Cleaning..."
	@rm -f $(APP_NAME) coverage.out coverage.html
	@go clean -testcache
	@echo "[OK] Done"

deps: ## Update Go dependencies
	@echo "[DEPS] Updating dependencies..."
	@go mod tidy
	@go mod download
	@echo "[OK] Dependencies updated"

ssh: ## SSH to the K3s server
	@cd terraform && ssh -i ~/.ssh/id_rsa ec2-user@$$(terraform output -raw k3s_public_ip)
