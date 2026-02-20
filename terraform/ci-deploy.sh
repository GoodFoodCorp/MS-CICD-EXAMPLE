#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Script CI/CD - Gestion intelligente de Terraform
# ═══════════════════════════════════════════════════════════
# Gère automatiquement :
# - Création du backend S3
# - Import des ressources existantes
# - Détection des changements
# - Application sélective
# ═══════════════════════════════════════════════════════════

set -e

REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="auth-service"
ENVIRONMENT="prod"

# Récupérer l'ID du compte AWS pour rendre le bucket unique
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

BUCKET_NAME="${PROJECT_NAME}-tfstate-${ENVIRONMENT}-${AWS_ACCOUNT_ID}"
TABLE_NAME="${PROJECT_NAME}-terraform-lock-${ENVIRONMENT}"

echo "🚀 Initialisation du déploiement Terraform CI/CD"
echo "   Région: ${REGION}"
echo "   Projet: ${PROJECT_NAME}"
echo "   Compte AWS: ${AWS_ACCOUNT_ID}"
echo ""

# ═══════════════════════════════════════════════════════════
# Étape 1 : Vérifier/Créer le backend S3
# ═══════════════════════════════════════════════════════════
echo "📦 Vérification du backend S3..."

if ! aws s3 ls "s3://${BUCKET_NAME}" 2>/dev/null; then
    echo "   ➜ Création du bucket S3 ${BUCKET_NAME}..."
    aws s3api create-bucket --bucket "${BUCKET_NAME}" --region "${REGION}"
    
    aws s3api put-bucket-versioning \
        --bucket "${BUCKET_NAME}" \
        --versioning-configuration Status=Enabled
    
    aws s3api put-bucket-encryption \
        --bucket "${BUCKET_NAME}" \
        --server-side-encryption-configuration '{
            "Rules": [{
                "ApplyServerSideEncryptionByDefault": {
                    "SSEAlgorithm": "AES256"
                }
            }]
        }'
    
    aws s3api put-public-access-block \
        --bucket "${BUCKET_NAME}" \
        --public-access-block-configuration \
        BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
    
    echo "   ✅ Bucket S3 créé"
else
    echo "   ✅ Bucket S3 existe déjà"
fi

if ! aws dynamodb describe-table --table-name "${TABLE_NAME}" --region "${REGION}" 2>/dev/null; then
    echo "   ➜ Création de la table DynamoDB ${TABLE_NAME}..."
    aws dynamodb create-table \
        --table-name "${TABLE_NAME}" \
        --attribute-definitions AttributeName=LockID,AttributeType=S \
        --key-schema AttributeName=LockID,KeyType=HASH \
        --billing-mode PAY_PER_REQUEST \
        --region "${REGION}"
    
    aws dynamodb wait table-exists --table-name "${TABLE_NAME}" --region "${REGION}"
    echo "   ✅ Table DynamoDB créée"
else
    echo "   ✅ Table DynamoDB existe déjà"
fi

# ═══════════════════════════════════════════════════════════
# Étape 2 : Configurer le backend Terraform dynamiquement
# ═══════════════════════════════════════════════════════════
echo ""
echo "⚙️  Configuration du backend Terraform..."

cat > backend-override.tf <<EOF
terraform {
  backend "s3" {
    bucket         = "${BUCKET_NAME}"
    key            = "terraform.tfstate"
    region         = "${REGION}"
    encrypt        = true
    dynamodb_table = "${TABLE_NAME}"
  }
}
EOF

echo "   ✅ Backend configuré avec:"
echo "      Bucket: ${BUCKET_NAME}"
echo "      Region: ${REGION}"
echo "      Table: ${TABLE_NAME}"

# ═══════════════════════════════════════════════════════════
# Étape 3 : Initialiser Terraform
# ═══════════════════════════════════════════════════════════
echo ""
echo "🔧 Initialisation de Terraform..."
terraform init -reconfigure

# ═══════════════════════════════════════════════════════════
# Étape 4 : Importer les ressources existantes si nécessaire
# ═══════════════════════════════════════════════════════════
echo ""
echo "🔍 Vérification des ressources existantes..."

import_if_exists() {
    local resource_type=$1
    local resource_name=$2
    local aws_id=$3
    
    # Vérifier si la ressource est déjà dans l'état Terraform
    if terraform state show "${resource_type}.${resource_name}" &>/dev/null; then
        echo "   ✅ ${resource_type}.${resource_name} déjà dans l'état"
        return 0
    fi
    
    # Tenter l'import
    echo "   ➜ Import de ${resource_type}.${resource_name}..."
    if terraform import "${resource_type}.${resource_name}" "${aws_id}" 2>/dev/null; then
        echo "   ✅ Import réussi"
        return 0
    else
        echo "   ℹ️  Ressource n'existe pas dans AWS (sera créée)"
        return 1
    fi
}

# Importer les ressources clés
import_if_exists "aws_key_pair" "deployer" "${PROJECT_NAME}-key" || true
import_if_exists "aws_db_subnet_group" "main" "${PROJECT_NAME}-db-subnet-group" || true

# ═══════════════════════════════════════════════════════════
# Étape 5 : Planifier les changements
# ═══════════════════════════════════════════════════════════
echo ""
echo "📋 Planification des changements..."

# Créer un plan et capturer le statut
if terraform plan -detailed-exitcode -out=tfplan; then
    PLAN_EXIT_CODE=$?
else
    PLAN_EXIT_CODE=$?
fi

# Exit codes Terraform plan:
# 0 = Pas de changements
# 1 = Erreur
# 2 = Changements détectés

if [ $PLAN_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ Infrastructure à jour - Aucun changement nécessaire"
    echo "TERRAFORM_CHANGES=false" >> "$GITHUB_OUTPUT"
    exit 0
elif [ $PLAN_EXIT_CODE -eq 2 ]; then
    echo ""
    echo "📝 Changements détectés - Application nécessaire"
    echo "TERRAFORM_CHANGES=true" >> "$GITHUB_OUTPUT"
elif [ $PLAN_EXIT_CODE -eq 1 ]; then
    echo ""
    echo "❌ Erreur lors de la planification"
    exit 1
fi

# ═══════════════════════════════════════════════════════════
# Étape 6 : Appliquer les changements
# ═══════════════════════════════════════════════════════════
echo ""
echo "🚀 Application des changements..."

if terraform apply -auto-approve tfplan; then
    echo ""
    echo "✅ Infrastructure déployée avec succès"
else
    echo ""
    echo "❌ Erreur lors de l'application"
    exit 1
fi

# ═══════════════════════════════════════════════════════════
# Étape 7 : Récupérer les outputs
# ═══════════════════════════════════════════════════════════
echo ""
echo "📤 Récupération des outputs Terraform..."

if terraform output k3s_public_ip &>/dev/null; then
    K3S_IP=$(terraform output -raw k3s_public_ip)
    echo "K3S_IP=${K3S_IP}" >> "$GITHUB_OUTPUT"
    echo "   ✅ IP K3s: ${K3S_IP}"
    
    # Sauvegarder la clé SSH
    terraform output -raw ssh_private_key > ../ssh_key.pem
    chmod 600 ../ssh_key.pem
    echo "   ✅ Clé SSH sauvegardée"
else
    echo "   ⚠️  Pas d'output k3s_public_ip (normal si infra déjà existante)"
fi

echo ""
echo "✅ Déploiement Terraform terminé avec succès!"
