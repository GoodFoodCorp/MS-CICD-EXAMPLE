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

echo "[DEPLOY] Initialisation du déploiement Terraform CI/CD"
echo "   Région: ${REGION}"
echo "   Projet: ${PROJECT_NAME}"
echo "   Compte AWS: ${AWS_ACCOUNT_ID}"
echo ""

# ═══════════════════════════════════════════════════════════
# Étape 1 : Vérifier/Créer le backend S3 (sans DynamoDB)
# ═══════════════════════════════════════════════════════════
echo " Vérification du backend S3..."

if ! aws s3 ls "s3://${BUCKET_NAME}" 2>/dev/null; then
    echo "   ➜ Création du bucket S3 ${BUCKET_NAME}..."
    aws s3api create-bucket --bucket "${BUCKET_NAME}" --region "${REGION}" 2>/dev/null
    
    echo "   ➜ Configuration du bucket..."
    aws s3api put-bucket-versioning \
        --bucket "${BUCKET_NAME}" \
        --versioning-configuration Status=Enabled 2>/dev/null
    
    aws s3api put-bucket-encryption \
        --bucket "${BUCKET_NAME}" \
        --server-side-encryption-configuration '{
            "Rules": [{
                "ApplyServerSideEncryptionByDefault": {
                    "SSEAlgorithm": "AES256"
                }
            }]
        }' 2>/dev/null
    
    aws s3api put-public-access-block \
        --bucket "${BUCKET_NAME}" \
        --public-access-block-configuration \
        BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true 2>/dev/null
    
    echo "   [OK] Bucket S3 créé et configuré"
else
    echo "   [OK] Bucket S3 existe déjà"
fi

# ═══════════════════════════════════════════════════════════
# Étape 2 : Configurer le backend Terraform (S3 uniquement)
# ═══════════════════════════════════════════════════════════
echo ""
echo "⚙️  Configuration du backend Terraform..."

cat > backend-override.tf <<EOF
terraform {
  backend "s3" {
    bucket  = "${BUCKET_NAME}"
    key     = "terraform.tfstate"
    region  = "${REGION}"
    encrypt = true
  }
}
EOF

echo "   [OK] Backend configuré:"
echo "      Bucket: ${BUCKET_NAME}"
echo "      Region: ${REGION}"

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
echo "[CHECK] Vérification et import des ressources existantes..."

import_if_exists() {
    local resource_type=$1
    local resource_name=$2
    local aws_id=$3
    
    # Vérifier si la ressource est déjà dans l'état Terraform
    if terraform state show "${resource_type}.${resource_name}" &>/dev/null; then
        echo "   [OK] ${resource_type}.${resource_name} déjà dans l'état"
        return 0
    fi
    
    # Tenter l'import
    echo "   ➜ Import de ${resource_type}.${resource_name} (${aws_id})..."
    if terraform import -input=false "${resource_type}.${resource_name}" "${aws_id}" 2>&1 | tee /tmp/tf-import.log; then
        echo "   [OK] Import réussi"
        return 0
    else
        if grep -q "Cannot import non-existent" /tmp/tf-import.log || grep -q "does not exist" /tmp/tf-import.log; then
            echo "   [INFO]  Ressource n'existe pas dans AWS (sera créée)"
        else
            echo "   [WARNING]  Import échoué (voir logs)"
        fi
        return 1
    fi
}

# Importer les ressources clés
echo ""
echo "    Vérification Key Pair..."
if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${PROJECT_NAME}-key" &>/dev/null; then
    echo "      ➜ Key pair existe dans AWS, import..."
    import_if_exists "aws_key_pair" "deployer" "${PROJECT_NAME}-key" || true
else
    echo "      [INFO]  Key pair n'existe pas (sera créée)"
fi

echo ""
echo "   [INFO] Vérification DB Subnet Group..."
if aws rds describe-db-subnet-groups --region "${REGION}" --db-subnet-group-name "${PROJECT_NAME}-db-subnet-group" &>/dev/null; then
    echo "      ➜ DB Subnet Group existe dans AWS, import..."
    import_if_exists "aws_db_subnet_group" "main" "${PROJECT_NAME}-db-subnet-group" || true
else
    echo "      [INFO]  DB Subnet Group n'existe pas (sera créé)"
fi

# Vérifier les VPCs existants du projet
echo ""
echo "   🌐 Vérification VPCs..."
VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
    --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
    --query 'Vpcs[*].VpcId' \
    --output text 2>/dev/null)

if [ -n "$VPC_IDS" ]; then
    VPC_COUNT=$(echo "$VPC_IDS" | wc -w | tr -d ' ')
    echo "      [WARNING]  ${VPC_COUNT} VPC(s) trouvé(s) avec le tag Project=${PROJECT_NAME}"
    echo "      IDs: ${VPC_IDS}"
    
    # Vérifier la limite totale
    TOTAL_VPCS=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text)
    if [ "$TOTAL_VPCS" -ge 5 ]; then
        echo ""
        echo "      [ERROR] ATTENTION : Limite de VPCs atteinte (${TOTAL_VPCS}/5)"
        echo "      Terraform ne pourra pas créer de nouveau VPC."
        echo "      Supprimez les VPCs non utilisés ou importez un VPC existant."
        echo ""
    fi
else
    echo "      [INFO]  Aucun VPC du projet trouvé"
fi

# ═══════════════════════════════════════════════════════════
# Étape 5 : Planifier les changements
# ═══════════════════════════════════════════════════════════
echo ""
echo "[LOG] Planification des changements..."

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
    echo "[OK] Infrastructure à jour - Aucun changement nécessaire"
    echo "TERRAFORM_CHANGES=false" >> "$GITHUB_OUTPUT"
    exit 0
elif [ $PLAN_EXIT_CODE -eq 2 ]; then
    echo ""
    echo "📝 Changements détectés - Application nécessaire"
    echo "TERRAFORM_CHANGES=true" >> "$GITHUB_OUTPUT"
elif [ $PLAN_EXIT_CODE -eq 1 ]; then
    echo ""
    echo "[ERROR] Erreur lors de la planification"
    exit 1
fi

# ═══════════════════════════════════════════════════════════
# Étape 6 : Appliquer les changements
# ═══════════════════════════════════════════════════════════
echo ""
echo "[DEPLOY] Application des changements..."

if terraform apply -auto-approve tfplan; then
    echo ""
    echo "[OK] Infrastructure déployée avec succès"
else
    echo ""
    echo "[ERROR] Erreur lors de l'application"
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
    echo "   [OK] IP K3s: ${K3S_IP}"
    
    # Sauvegarder la clé SSH
    terraform output -raw ssh_private_key > ../ssh_key.pem
    chmod 600 ../ssh_key.pem
    echo "   [OK] Clé SSH sauvegardée"
else
    echo "   [WARNING]  Pas d'output k3s_public_ip (normal si infra déjà existante)"
fi

echo ""
echo "[OK] Déploiement Terraform terminé avec succès!"
