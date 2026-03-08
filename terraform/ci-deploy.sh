#!/bin/bash
# Script CI/CD - Gestion Terraform
# Gere: creation du backend S3, import des ressources existantes,
#       detection des changements, application selective.

set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="auth-service"
ENVIRONMENT="prod"

AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
BUCKET_NAME="${PROJECT_NAME}-tfstate-${ENVIRONMENT}-${AWS_ACCOUNT_ID}"

echo "[DEPLOY] Initialisation du deploiement Terraform CI/CD"
echo "   Region:      ${REGION}"
echo "   Projet:      ${PROJECT_NAME}"
echo "   Compte AWS:  ${AWS_ACCOUNT_ID}"
echo ""

# --- Etape 1 : Verifier/Creer le backend S3 ---
echo "[1/7] Verification du backend S3..."

if ! aws s3 ls "s3://${BUCKET_NAME}" 2>/dev/null; then
    echo "   -> Creation du bucket S3 ${BUCKET_NAME}..."
    aws s3api create-bucket --bucket "${BUCKET_NAME}" --region "${REGION}" 2>/dev/null || true
    echo "   -> Configuration du bucket..."
    aws s3api put-bucket-versioning \
        --bucket "${BUCKET_NAME}" \
        --versioning-configuration Status=Enabled 2>/dev/null || true

    aws s3api put-bucket-encryption \
        --bucket "${BUCKET_NAME}" \
        --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}' 2>/dev/null || true

    aws s3api put-public-access-block \
        --bucket "${BUCKET_NAME}" \
        --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true 2>/dev/null || true
    echo "   [OK] Bucket S3 cree et configure"
else
    echo "   [OK] Bucket S3 existe deja"
fi

# --- Etape 2 : Configurer le backend Terraform ---
echo ""
echo "[2/7] Configuration du backend Terraform..."

cat > backend-override.tf << EOF
terraform {
  backend "s3" {
    bucket  = "${BUCKET_NAME}"
    key     = "terraform.tfstate"
    region  = "${REGION}"
    encrypt = true
  }
}
EOF

echo "   [OK] Backend configure (Bucket: ${BUCKET_NAME})"

# --- Etape 3 : Initialiser Terraform ---
echo ""
echo "[3/7] Initialisation de Terraform..."
terraform init -reconfigure

# --- Etape 4 : Importer les ressources existantes si necessaire ---
echo ""
echo "[4/7] Verification et import des ressources existantes..."

import_if_exists() {
    local resource_type=$1
    local resource_name=$2
    local aws_id=$3

    if terraform state show "${resource_type}.${resource_name}" &>/dev/null; then
        echo "   [OK] ${resource_type}.${resource_name} deja dans l'etat"
        return 0
    fi

    echo "   -> Import de ${resource_type}.${resource_name} (${aws_id})..."
    if terraform import -input=false "${resource_type}.${resource_name}" "${aws_id}" 2>&1 | tee /tmp/tf-import.log; then
        echo "   [OK] Import reussi"
        return 0
    else
        if grep -q "Cannot import non-existent\|does not exist" /tmp/tf-import.log; then
            echo "   [INFO] Ressource n'existe pas dans AWS (sera creee)"
        else
            echo "   [WARNING] Import echoue (voir logs)"
        fi
        return 1
    fi
}

echo ""
echo "   Key Pair..."
if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${PROJECT_NAME}-key" &>/dev/null; then
    import_if_exists "aws_key_pair" "deployer" "${PROJECT_NAME}-key" || true
else
    echo "   [INFO] Key pair n'existe pas (sera creee)"
fi

echo ""
echo "   DB Subnet Group..."
if aws rds describe-db-subnet-groups --region "${REGION}" --db-subnet-group-name "${PROJECT_NAME}-db-subnet-group" &>/dev/null; then
    import_if_exists "aws_db_subnet_group" "main" "${PROJECT_NAME}-db-subnet-group" || true
else
    echo "   [INFO] DB Subnet Group n'existe pas (sera cree)"
fi

echo ""
echo "   VPCs du projet..."
VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
    --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
    --query 'Vpcs[*].VpcId' --output text 2>/dev/null)

if [ -n "$VPC_IDS" ]; then
    VPC_COUNT=$(echo "$VPC_IDS" | wc -w | tr -d ' ')
    echo "   [WARNING] ${VPC_COUNT} VPC(s) du projet trouves: ${VPC_IDS}"
    TOTAL_VPCS=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text)
    if [ "$TOTAL_VPCS" -ge 5 ]; then
        echo ""
        echo "   [ERROR] Limite de VPCs atteinte (${TOTAL_VPCS}/5)"
        echo "   Supprimez les VPCs non utilises avant de continuer."
        echo ""
    fi
else
    echo "   [INFO] Aucun VPC du projet trouve"
fi

# --- Etape 5 : Planifier les changements ---
echo ""
echo "[5/7] Planification des changements..."

PLAN_EXIT_CODE=0
terraform plan -detailed-exitcode -out=tfplan || PLAN_EXIT_CODE=$?

if [ $PLAN_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "[OK] Infrastructure a jour - Aucun changement necessaire"
    echo "TERRAFORM_CHANGES=false" >> "${GITHUB_OUTPUT:-/dev/null}"
    exit 0
elif [ $PLAN_EXIT_CODE -eq 2 ]; then
    echo ""
    echo "[INFO] Changements detectes - Application necessaire"
    echo "TERRAFORM_CHANGES=true" >> "${GITHUB_OUTPUT:-/dev/null}"
elif [ $PLAN_EXIT_CODE -eq 1 ]; then
    echo ""
    echo "[ERROR] Erreur lors de la planification"
    exit 1
fi

# --- Etape 6 : Appliquer les changements ---
echo ""
echo "[6/7] Application des changements..."

if terraform apply -auto-approve tfplan; then
    echo ""
    echo "[OK] Infrastructure deployee avec succes"
else
    echo ""
    echo "[ERROR] Erreur lors de l'application"
    exit 1
fi

# --- Etape 7 : Recuperer les outputs ---
echo ""
echo "[7/7] Recuperation des outputs Terraform..."

if terraform output k3s_public_ip &>/dev/null; then
    K3S_IP=$(terraform output -raw k3s_public_ip)
    echo "K3S_IP=${K3S_IP}" >> "${GITHUB_OUTPUT:-/dev/null}"
    echo "   [OK] IP K3s: ${K3S_IP}"
    terraform output -raw ssh_private_key > ../ssh_key.pem
    chmod 600 ../ssh_key.pem
    echo "   [OK] Cle SSH sauvegardee"
else
    echo "   [WARNING] Pas d'output k3s_public_ip (normal si infra deja existante)"
fi

echo ""
echo "[OK] Deploiement Terraform termine avec succes!"
