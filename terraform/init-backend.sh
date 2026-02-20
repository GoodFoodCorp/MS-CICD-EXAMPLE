#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Script d'initialisation du backend Terraform S3
# ═══════════════════════════════════════════════════════════

set -e

REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="auth-service"
ENVIRONMENT="prod"
BUCKET_NAME="${PROJECT_NAME}-terraform-state-${ENVIRONMENT}"
TABLE_NAME="${PROJECT_NAME}-terraform-lock-${ENVIRONMENT}"

echo "🚀 Initialisation du backend Terraform pour ${PROJECT_NAME}..."
echo "   Région: ${REGION}"
echo "   Bucket: ${BUCKET_NAME}"
echo "   Table DynamoDB: ${TABLE_NAME}"
echo ""

# Vérifier si le bucket existe déjà
if aws s3 ls "s3://${BUCKET_NAME}" 2>/dev/null; then
    echo "✅ Le bucket ${BUCKET_NAME} existe déjà"
else
    echo "📦 Création du bucket S3 ${BUCKET_NAME}..."
    aws s3api create-bucket --bucket "${BUCKET_NAME}" --region "${REGION}"
    
    echo "🔒 Activation du versioning..."
    aws s3api put-bucket-versioning \
        --bucket "${BUCKET_NAME}" \
        --versioning-configuration Status=Enabled
    
    echo "🔐 Activation du chiffrement..."
    aws s3api put-bucket-encryption \
        --bucket "${BUCKET_NAME}" \
        --server-side-encryption-configuration '{
            "Rules": [{
                "ApplyServerSideEncryptionByDefault": {
                    "SSEAlgorithm": "AES256"
                }
            }]
        }'
    
    echo "🚫 Blocage de l'accès public..."
    aws s3api put-public-access-block \
        --bucket "${BUCKET_NAME}" \
        --public-access-block-configuration \
        BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
fi

# Vérifier si la table DynamoDB existe déjà
if aws dynamodb describe-table --table-name "${TABLE_NAME}" --region "${REGION}" 2>/dev/null; then
    echo "✅ La table DynamoDB ${TABLE_NAME} existe déjà"
else
    echo "📊 Création de la table DynamoDB ${TABLE_NAME}..."
    aws dynamodb create-table \
        --table-name "${TABLE_NAME}" \
        --attribute-definitions AttributeName=LockID,AttributeType=S \
        --key-schema AttributeName=LockID,KeyType=HASH \
        --billing-mode PAY_PER_REQUEST \
        --region "${REGION}"
    
    echo "⏳ Attente de la création de la table..."
    aws dynamodb wait table-exists --table-name "${TABLE_NAME}" --region "${REGION}"
fi

echo ""
echo "✅ Backend initialisé avec succès!"
echo ""
echo "📝 Configuration à ajouter dans provider.tf:"
echo ""
echo "terraform {"
echo "  backend \"s3\" {"
echo "    bucket         = \"${BUCKET_NAME}\""
echo "    key            = \"terraform.tfstate\""
echo "    region         = \"${REGION}\""
echo "    encrypt        = true"
echo "    dynamodb_table = \"${TABLE_NAME}\""
echo "  }"
echo "}"
echo ""
echo "⚠️  Après avoir activé le backend, exécutez:"
echo "   terraform init -migrate-state"
