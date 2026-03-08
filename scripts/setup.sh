#!/bin/bash
# Script de setup initial
# Configure l'environnement pour le deploiement

set -euo pipefail

echo "--------------------------------------------"
echo "   Setup - Auth Service CI/CD"
echo "--------------------------------------------"
echo ""

# 1. Verifier les dependances
echo "[1/6] Dependances..."
echo ""

MISSING_DEPS=()

if command -v terraform &>/dev/null; then
    echo "  [OK] Terraform: $(terraform version -json | python3 -c 'import sys,json; print(json.load(sys.stdin).get("terraform_version","?"))')"
else
    echo "  [ERROR] Terraform non installe"
    MISSING_DEPS+=("terraform")
fi

if command -v aws &>/dev/null; then
    echo "  [OK] AWS CLI: $(aws --version 2>&1 | cut -d' ' -f1)"
else
    echo "  [ERROR] AWS CLI non installe"
    MISSING_DEPS+=("aws-cli")
fi

if command -v docker &>/dev/null; then
    echo "  [OK] Docker: $(docker --version | cut -d' ' -f3 | tr -d ',')"
else
    echo "  [ERROR] Docker non installe"
    MISSING_DEPS+=("docker")
fi

if command -v kubectl &>/dev/null; then
    echo "  [OK] kubectl: $(kubectl version --client -o json 2>/dev/null | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("clientVersion",{}).get("gitVersion","?"))')"
else
    echo "  [ERROR] kubectl non installe"
    MISSING_DEPS+=("kubectl")
fi

if command -v jq &>/dev/null; then
    echo "  [OK] jq: $(jq --version)"
else
    echo "  [ERROR] jq non installe"
    MISSING_DEPS+=("jq")
fi

echo ""

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
    echo "[ERROR] Dependances manquantes: ${MISSING_DEPS[*]}"
    exit 1
fi

# 2. Configuration AWS
echo "[2/6] Configuration AWS..."
echo ""

if aws sts get-caller-identity &>/dev/null; then
    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    REGION=$(aws configure get region 2>/dev/null || echo "us-east-1")
    echo "  [OK] AWS configure"
    echo "     Account ID: $ACCOUNT_ID"
    echo "     Region:     $REGION"
else
    echo "  [ERROR] AWS non configure"
    echo ""
    echo "Configurez AWS avec: aws configure"
    exit 1
fi

echo ""

# 3. Cle SSH
echo "[3/6] Cle SSH..."
echo ""

if [ -f ~/.ssh/id_rsa.pub ]; then
    echo "  [OK] Cle SSH trouvee: ~/.ssh/id_rsa.pub"
else
    echo "  [WARNING] Cle SSH non trouvee"
    echo ""
    echo "Voulez-vous generer une nouvelle cle SSH ? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
        echo "  [OK] Cle SSH generee"
    else
        echo "  [ERROR] Cle SSH requise pour le deploiement"
        exit 1
    fi
fi

echo ""

# 4. Configuration Terraform
echo "[4/6] Configuration Terraform..."
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/../terraform"

if [ ! -f terraform.tfvars ]; then
    echo "  [WARNING] terraform.tfvars non trouve"
    echo ""
    echo "Creation de terraform.tfvars..."

    read -rp "Docker Hub username: " DOCKER_USERNAME
    read -rsp "Database password: " DB_PASSWORD
    echo ""
    read -rp "Webhook URL (ou Entree pour laisser vide): " USER_WEBHOOK
    WEBHOOK_URL="${USER_WEBHOOK:-}"

    cat > terraform.tfvars << EOF
aws_region         = "${REGION}"
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
db_password          = "${DB_PASSWORD}"

docker_username = "${DOCKER_USERNAME}"
app_port        = 8081
webhook_url     = "${WEBHOOK_URL}"
EOF

    echo "  [OK] terraform.tfvars cree"
else
    echo "  [OK] terraform.tfvars existe"
fi

cd "${SCRIPT_DIR}/.."
echo ""

# 5. Secrets GitHub requis
echo "[5/6] Secrets GitHub a configurer (Settings > Secrets > Actions):"
echo ""
echo "  AWS_ACCESS_KEY_ID         - Cle d'acces AWS"
echo "  AWS_SECRET_ACCESS_KEY     - Secret AWS"
echo "  AWS_REGION                - Region: ${REGION}"
echo "  DOCKER_USERNAME           - Username Docker Hub"
echo "  DOCKER_PASSWORD           - Token Docker Hub"
echo "  KUBE_CONFIG               - Config kubectl (base64)"
echo "  WEBHOOK_URL               - URL du webhook (optionnel)"
echo ""

# 6. Permissions
echo "[6/6] Permissions des scripts..."
echo ""

chmod +x "${SCRIPT_DIR}"/*.sh
echo "  [OK] Scripts rendus executables"
echo ""

echo "--------------------------------------------"
echo "[OK] Setup termine!"
echo "--------------------------------------------"
echo ""
echo "Prochaines etapes:"
echo ""
echo "  1. Deployer l'infrastructure:"
echo "     cd terraform"
echo "     terraform init"
echo "     terraform plan"
echo "     terraform apply"
echo ""
echo "  2. Lancer la CI/CD:"
echo "     git push origin main"
