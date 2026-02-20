#!/bin/bash

# ═══════════════════════════════════════════════════════════
# Script de setup initial
# Configure l'environnement pour le déploiement
# ═══════════════════════════════════════════════════════════

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   🔧 Setup - Auth Service CI/CD${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# ═══════════════════════════════════════════════════════════
# 1. Vérifier les dépendances
# ═══════════════════════════════════════════════════════════

echo -e "${YELLOW}1️⃣  Vérification des dépendances...${NC}"
echo ""

MISSING_DEPS=()

# Terraform
if command -v terraform &> /dev/null; then
  echo -e "  ${GREEN}✅ Terraform: $(terraform version -json | jq -r '.terraform_version')${NC}"
else
  echo -e "  ${RED}❌ Terraform non installé${NC}"
  MISSING_DEPS+=("terraform")
fi

# AWS CLI
if command -v aws &> /dev/null; then
  echo -e "  ${GREEN}✅ AWS CLI: $(aws --version | cut -d' ' -f1)${NC}"
else
  echo -e "  ${RED}❌ AWS CLI non installé${NC}"
  MISSING_DEPS+=("aws-cli")
fi

# Docker
if command -v docker &> /dev/null; then
  echo -e "  ${GREEN}✅ Docker: $(docker --version | cut -d' ' -f3 | tr -d ',')${NC}"
else
  echo -e "  ${RED}❌ Docker non installé${NC}"
  MISSING_DEPS+=("docker")
fi

# kubectl
if command -v kubectl &> /dev/null; then
  echo -e "  ${GREEN}✅ kubectl: $(kubectl version --client --short 2>/dev/null | cut -d' ' -f3)${NC}"
else
  echo -e "  ${RED}❌ kubectl non installé${NC}"
  MISSING_DEPS+=("kubectl")
fi

# jq
if command -v jq &> /dev/null; then
  echo -e "  ${GREEN}✅ jq: $(jq --version)${NC}"
else
  echo -e "  ${RED}❌ jq non installé${NC}"
  MISSING_DEPS+=("jq")
fi

echo ""

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
  echo -e "${RED}❌ Dépendances manquantes: ${MISSING_DEPS[*]}${NC}"
  echo -e "${YELLOW}Veuillez installer les dépendances manquantes avant de continuer.${NC}"
  exit 1
fi

# ═══════════════════════════════════════════════════════════
# 2. Configuration AWS
# ═══════════════════════════════════════════════════════════

echo -e "${YELLOW}2️⃣  Configuration AWS...${NC}"
echo ""

if aws sts get-caller-identity &> /dev/null; then
  ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
  REGION=$(aws configure get region || echo "eu-west-3")
  echo -e "  ${GREEN}✅ AWS configuré${NC}"
  echo -e "     Account ID: $ACCOUNT_ID"
  echo -e "     Région:     $REGION"
else
  echo -e "  ${RED}❌ AWS non configuré${NC}"
  echo ""
  echo -e "${YELLOW}Configurez AWS avec:${NC}"
  echo -e "  ${CYAN}aws configure${NC}"
  exit 1
fi

echo ""

# ═══════════════════════════════════════════════════════════
# 3. Clé SSH
# ═══════════════════════════════════════════════════════════

echo -e "${YELLOW}3️⃣  Vérification de la clé SSH...${NC}"
echo ""

if [ -f ~/.ssh/id_rsa.pub ]; then
  echo -e "  ${GREEN}✅ Clé SSH trouvée: ~/.ssh/id_rsa.pub${NC}"
else
  echo -e "  ${YELLOW}⚠️  Clé SSH non trouvée${NC}"
  echo ""
  echo -e "${YELLOW}Voulez-vous générer une nouvelle clé SSH ? (y/N)${NC}"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
    echo -e "  ${GREEN}✅ Clé SSH générée${NC}"
  else
    echo -e "  ${RED}❌ Clé SSH requise pour le déploiement${NC}"
    exit 1
  fi
fi

echo ""

# ═══════════════════════════════════════════════════════════
# 4. Configuration Terraform
# ═══════════════════════════════════════════════════════════

echo -e "${YELLOW}4️⃣  Configuration Terraform...${NC}"
echo ""

cd terraform

if [ ! -f terraform.tfvars ]; then
  echo -e "  ${YELLOW}⚠️  terraform.tfvars non trouvé${NC}"
  echo ""
  echo -e "${YELLOW}Création de terraform.tfvars...${NC}"
  
  # Demander les informations
  read -p "Docker Hub username: " DOCKER_USERNAME
  read -p "Database password: " -s DB_PASSWORD
  echo ""
  read -p "Webhook URL (ou Entrée pour valeur par défaut): " WEBHOOK_URL
  WEBHOOK_URL=${WEBHOOK_URL:-"https://webhook.site/unique-id"}
  
  # Créer le fichier
  cat > terraform.tfvars <<EOF
# Configuration générée automatiquement
aws_region         = "$REGION"
environment        = "prod"
project_name       = "auth-service"

vpc_cidr           = "10.0.0.0/16"
availability_zones = ["${REGION}a", "${REGION}b"]
public_subnets     = ["10.0.1.0/24"]
private_subnets    = ["10.0.10.0/24", "10.0.11.0/24"]

instance_type      = "t3.medium"
key_name           = "auth-service-key"

db_allocated_storage = 20
db_instance_class    = "db.t3.micro"
db_name              = "authdb"
db_username          = "authuser"
db_password          = "$DB_PASSWORD"

docker_username = "$DOCKER_USERNAME"
app_port        = 8081
webhook_url     = "$WEBHOOK_URL"
EOF
  
  echo -e "  ${GREEN}✅ terraform.tfvars créé${NC}"
else
  echo -e "  ${GREEN}✅ terraform.tfvars existe${NC}"
fi

cd ..
echo ""

# ═══════════════════════════════════════════════════════════
# 5. GitHub Secrets recommandés
# ═══════════════════════════════════════════════════════════

echo -e "${YELLOW}5️⃣  GitHub Secrets à configurer:${NC}"
echo ""
echo -e "  ${CYAN}AWS_ACCESS_KEY_ID${NC}         - Clé d'accès AWS"
echo -e "  ${CYAN}AWS_SECRET_ACCESS_KEY${NC}     - Secret AWS"
echo -e "  ${CYAN}AWS_REGION${NC}                - Région: $REGION"
echo -e "  ${CYAN}DOCKER_USERNAME${NC}           - Username Docker Hub"
echo -e "  ${CYAN}DOCKER_PASSWORD${NC}           - Token Docker Hub"
echo -e "  ${CYAN}KUBE_CONFIG${NC}               - Config kubectl (base64)"
echo -e "  ${CYAN}WEBHOOK_URL${NC}               - URL du webhook"
echo ""

# ═══════════════════════════════════════════════════════════
# 6. Rendre les scripts exécutables
# ═══════════════════════════════════════════════════════════

echo -e "${YELLOW}6️⃣  Configuration des permissions...${NC}"
echo ""

chmod +x scripts/*.sh
echo -e "  ${GREEN}✅ Scripts rendus exécutables${NC}"
echo ""

# ═══════════════════════════════════════════════════════════
# Résumé
# ═══════════════════════════════════════════════════════════

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Setup terminé!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📝 Prochaines étapes:${NC}"
echo ""
echo -e "  1️⃣  Déployer l'infrastructure:"
echo -e "     ${CYAN}cd terraform${NC}"
echo -e "     ${CYAN}terraform init${NC}"
echo -e "     ${CYAN}terraform plan${NC}"
echo -e "     ${CYAN}terraform apply${NC}"
echo ""
echo -e "  2️⃣  Configurer GitHub Secrets (voir liste ci-dessus)"
echo ""
echo -e "  3️⃣  Pusher le code sur GitHub:"
echo -e "     ${CYAN}git add .${NC}"
echo -e "     ${CYAN}git commit -m 'feat: CI/CD setup'${NC}"
echo -e "     ${CYAN}git push origin main${NC}"
echo ""
echo -e "  4️⃣  Le pipeline CI/CD se déclenchera automatiquement!"
echo ""
