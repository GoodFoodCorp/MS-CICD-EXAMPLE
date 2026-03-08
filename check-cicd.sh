#!/bin/bash
# Script de verification avant push CI/CD

set -euo pipefail

echo "[CHECK] Verification de la configuration CI/CD..."
echo ""

ERRORS=0
WARNINGS=0

# 1. Verifier les credentials AWS
echo "[1/7] Credentials AWS..."

if aws sts get-caller-identity &>/dev/null; then
    ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
    USER=$(aws sts get-caller-identity --query Arn --output text | cut -d'/' -f2)
    echo "   [OK] Credentials valides - Compte: ${ACCOUNT} / User: ${USER}"
else
    echo "   [ERROR] Credentials AWS invalides"
    ((ERRORS++))
fi

# 2. Verifier les fichiers Terraform
echo ""
echo "[2/7] Fichiers Terraform..."

REQUIRED_FILES=(
    "terraform/main.tf"
    "terraform/provider.tf"
    "terraform/variables.tf"
    "terraform/ci-deploy.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "   [OK] $file"
    else
        echo "   [ERROR] $file manquant"
        ((ERRORS++))
    fi
done

# 3. Verifier la configuration du backend S3
echo ""
echo "[3/7] Backend S3..."

if grep -q 'backend "s3"' terraform/provider.tf; then
    echo "   [OK] Backend S3 configure"
    BUCKET=$(grep 'bucket' terraform/provider.tf | grep -v '#' | awk -F'"' '{print $2}' | head -1)
    [ -n "$BUCKET" ] && echo "      Bucket: ${BUCKET}"
else
    echo "   [WARNING] Backend S3 non configure (sera cree par la CI)"
    ((WARNINGS++))
fi

# 4. Verifier le workflow GitHub Actions
echo ""
echo "[4/7] Workflow GitHub Actions..."

if [ -f ".github/workflows/ci-cd.yml" ]; then
    echo "   [OK] Workflow CI/CD present"
    if grep -q "ci-deploy.sh" .github/workflows/ci-cd.yml; then
        echo "   [OK] Script ci-deploy.sh reference dans le workflow"
    else
        echo "   [WARNING] Script ci-deploy.sh non reference"
        ((WARNINGS++))
    fi
else
    echo "   [ERROR] Workflow CI/CD manquant"
    ((ERRORS++))
fi

# 5. Secrets GitHub requis
echo ""
echo "[5/7] Secrets GitHub requis (a verifier dans Settings > Secrets):"
for secret in AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_REGION DOCKER_USERNAME DOCKER_PASSWORD; do
    echo "   - ${secret}"
done

# 6. Limites AWS
echo ""
echo "[6/7] Limites AWS..."

if aws sts get-caller-identity &>/dev/null; then
    REGION="${AWS_REGION:-us-east-1}"
    VPC_COUNT=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text 2>/dev/null || echo "?")
    echo "   VPCs utilises: ${VPC_COUNT}/5"
    if [ "$VPC_COUNT" != "?" ] && [ "$VPC_COUNT" -ge 5 ]; then
        echo "   [WARNING] Limite de VPCs atteinte"
        ((WARNINGS++))
    fi
    RDS_COUNT=$(aws rds describe-db-instances --region "${REGION}" --query 'DBInstances | length(@)' --output text 2>/dev/null || echo "?")
    echo "   Instances RDS: ${RDS_COUNT}"
fi

# 7. Ressources existantes
echo ""
echo "[7/7] Ressources existantes..."

if aws sts get-caller-identity &>/dev/null; then
    REGION="${AWS_REGION:-us-east-1}"
    PROJECT_NAME="auth-service"

    if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${PROJECT_NAME}-key" &>/dev/null; then
        echo "   [INFO] Key pair ${PROJECT_NAME}-key existe (import auto)"
    fi
    if aws rds describe-db-subnet-groups --region "${REGION}" --db-subnet-group-name "${PROJECT_NAME}-db-subnet-group" &>/dev/null; then
        echo "   [INFO] DB Subnet Group existe (import auto)"
    fi
    VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
        --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
        --query 'Vpcs[*].VpcId' --output text 2>/dev/null)
    if [ -n "$VPC_IDS" ]; then
        echo "   [WARNING] VPCs du projet trouves: ${VPC_IDS}"
    fi
fi

# Resume
echo ""
echo "================================================================"
echo "[INFO] Resume: Erreurs=${ERRORS}  Avertissements=${WARNINGS}"
echo "================================================================"

if [ $ERRORS -eq 0 ]; then
    echo "[OK] Pret pour le deploiement"
    echo ""
    echo "Prochaines etapes:"
    echo "   1. git add ."
    echo "   2. git commit -m 'feat: ...'"
    echo "   3. git push origin main"
    exit 0
else
    echo "[ERROR] Corrigez les erreurs avant de continuer"
    exit 1
fi
