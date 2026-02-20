# ═══════════════════════════════════════════════════════════
# Makefile - Auth Service CI/CD
# ═══════════════════════════════════════════════════════════

.PHONY: help setup build test docker terraform deploy clean monitor rollback

# Variables
APP_NAME = auth-service
DOCKER_USERNAME ?= votreusername
VERSION ?= latest
NAMESPACE = production

# Colors
BLUE = \033[0;34m
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m # No Color

# ═══════════════════════════════════════════════════════════
# Help
# ═══════════════════════════════════════════════════════════

help: ## Afficher l'aide
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BLUE)  Auth Service - Makefile$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make $(YELLOW)<target>$(NC)\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

# ═══════════════════════════════════════════════════════════
# Setup & Configuration
# ═══════════════════════════════════════════════════════════

setup: ## Configuration initiale complète
	@echo "$(YELLOW)🔧 Setup initial...$(NC)"
	@chmod +x scripts/*.sh
	@./scripts/setup.sh

# ═══════════════════════════════════════════════════════════
# Development
# ═══════════════════════════════════════════════════════════

build: ## Build l'application Go
	@echo "$(YELLOW)🔨 Build de l'application...$(NC)"
	@go build -o $(APP_NAME) ./cmd/main.go
	@echo "$(GREEN)✅ Build réussi: $(APP_NAME)$(NC)"

test: ## Lancer les tests
	@echo "$(YELLOW)🧪 Lancement des tests...$(NC)"
	@./run_tests.sh

test-coverage: ## Tests avec couverture détaillée
	@echo "$(YELLOW)📊 Tests avec couverture...$(NC)"
	@go test ./... -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Rapport généré: coverage.html$(NC)"

run: ## Lancer l'application localement
	@echo "$(YELLOW)🚀 Démarrage de l'application...$(NC)"
	@go run ./cmd/main.go

# ═══════════════════════════════════════════════════════════
# Docker
# ═══════════════════════════════════════════════════════════

docker-build: ## Build l'image Docker
	@echo "$(YELLOW)🐳 Build Docker...$(NC)"
	@docker build -t $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION) .
	@docker tag $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION) $(DOCKER_USERNAME)/$(APP_NAME):latest
	@echo "$(GREEN)✅ Image créée: $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION)$(NC)"

docker-push: ## Push l'image Docker
	@echo "$(YELLOW)📤 Push Docker...$(NC)"
	@docker push $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION)
	@docker push $(DOCKER_USERNAME)/$(APP_NAME):latest
	@echo "$(GREEN)✅ Image pushée$(NC)"

docker-run: ## Lancer le container localement
	@echo "$(YELLOW)🐳 Démarrage du container...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)✅ Container démarré$(NC)"

docker-stop: ## Arrêter le container
	@docker-compose down

docker-logs: ## Voir les logs Docker
	@docker-compose logs -f

# ═══════════════════════════════════════════════════════════
# Terraform
# ═══════════════════════════════════════════════════════════

terraform-init: ## Initialiser Terraform
	@echo "$(YELLOW)🏗️  Terraform init...$(NC)"
	@cd terraform && terraform init

terraform-plan: ## Plan Terraform
	@echo "$(YELLOW)📋 Terraform plan...$(NC)"
	@cd terraform && terraform plan

terraform-apply: ## Appliquer l'infrastructure
	@echo "$(YELLOW)🚀 Terraform apply...$(NC)"
	@cd terraform && terraform apply

terraform-destroy: ## Détruire l'infrastructure
	@echo "$(YELLOW)💣 Terraform destroy...$(NC)"
	@cd terraform && terraform destroy

terraform-output: ## Afficher les outputs Terraform
	@cd terraform && terraform output

# ═══════════════════════════════════════════════════════════
# Kubernetes
# ═══════════════════════════════════════════════════════════

k8s-apply: ## Appliquer les manifestes K8s
	@echo "$(YELLOW)☸️  Application des manifestes K8s...$(NC)"
	@kubectl apply -f k8s/namespace.yaml
	@kubectl apply -f k8s/deployment.yaml
	@echo "$(GREEN)✅ Manifestes appliqués$(NC)"

k8s-status: ## Voir le statut des pods
	@kubectl get pods -n $(NAMESPACE) -l app=$(APP_NAME)

k8s-logs: ## Voir les logs des pods
	@kubectl logs -f -n $(NAMESPACE) -l app=$(APP_NAME) --tail=100

k8s-describe: ## Décrire le deployment
	@kubectl describe deployment $(APP_NAME) -n $(NAMESPACE)

k8s-shell: ## Shell dans un pod
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pods -n $(NAMESPACE) -l app=$(APP_NAME) -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

# ═══════════════════════════════════════════════════════════
# Deployment
# ═══════════════════════════════════════════════════════════

deploy: ## Déployer l'application
	@./scripts/deploy.sh $(VERSION)

rollback: ## Rollback du déploiement
	@./scripts/rollback.sh

monitor: ## Dashboard de monitoring
	@./scripts/monitor.sh

notify-test: ## Tester les notifications
	@./scripts/notify.sh test

# ═══════════════════════════════════════════════════════════
# Security
# ═══════════════════════════════════════════════════════════

security-scan: ## Scanner les vulnérabilités
	@echo "$(YELLOW)🔐 Security scan...$(NC)"
	@echo "  → govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...
	@echo "  → gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec ./...
	@echo "$(GREEN)✅ Scan terminé$(NC)"

# ═══════════════════════════════════════════════════════════
# Utilities
# ═══════════════════════════════════════════════════════════

clean: ## Nettoyer les fichiers générés
	@echo "$(YELLOW)🧹 Nettoyage...$(NC)"
	@rm -f $(APP_NAME)
	@rm -f coverage.out coverage.html
	@rm -f *.log
	@go clean -testcache
	@echo "$(GREEN)✅ Nettoyage terminé$(NC)"

deps: ## Mettre à jour les dépendances
	@echo "$(YELLOW)📦 Mise à jour des dépendances...$(NC)"
	@go mod tidy
	@go mod download
	@echo "$(GREEN)✅ Dépendances à jour$(NC)"

fmt: ## Formater le code
	@echo "$(YELLOW)💅 Formatage du code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formaté$(NC)"

lint: ## Linter le code
	@echo "$(YELLOW)🔍 Linting...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run ./...
	@echo "$(GREEN)✅ Linting terminé$(NC)"

# ═══════════════════════════════════════════════════════════
# SSH & Infra
# ═══════════════════════════════════════════════════════════

ssh: ## SSH vers le serveur K3s
	@echo "$(YELLOW)🔐 Connexion SSH...$(NC)"
	@cd terraform && ssh -i ~/.ssh/id_rsa ec2-user@$$(terraform output -raw k3s_public_ip)

get-kubeconfig: ## Récupérer le kubeconfig
	@echo "$(YELLOW)📥 Récupération du kubeconfig...$(NC)"
	@cd terraform && scp -i ~/.ssh/id_rsa ec2-user@$$(terraform output -raw k3s_public_ip):/home/ec2-user/.kube/config ~/.kube/config-auth-service
	@echo "$(GREEN)✅ Kubeconfig sauvegardé: ~/.kube/config-auth-service$(NC)"
	@echo "$(YELLOW)Pour l'utiliser: export KUBECONFIG=~/.kube/config-auth-service$(NC)"

# ═══════════════════════════════════════════════════════════
# Complete workflows
# ═══════════════════════════════════════════════════════════

all: deps fmt lint test build ## Pipeline complet local

ci: deps test security-scan ## Simulation du pipeline CI

full-deploy: terraform-apply get-kubeconfig k8s-apply ## Déploiement complet from scratch

.DEFAULT_GOAL := help
