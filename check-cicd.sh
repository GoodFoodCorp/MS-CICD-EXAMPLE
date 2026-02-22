#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Script de vérification avant push
# ═══════════════════════════════════════════════════════════

set -e

echo "🔍 Vérification de la configuration CI/CD..."
echo ""

ERRORS=0
WARNINGS=0

# ═══════════════════════════════════════════════════════════
# 1. Vérifier les credentials AWS
# ═══════════════════════════════════════════════════════════
echo "1️⃣  Vérification des credentials AWS..."

if aws sts get-caller-identity &>/dev/null; then
    ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
    USER=$(aws sts get-caller-identity --query Arn --output text | cut -d'/' -f2)
    echo "   ✅ Credentials AWS valides"
    echo "      Compte: ${ACCOUNT}"
    echo "      User: ${USER}"
else
    echo "   ❌ Credentials AWS invalides"
    echo "      Assurez-vous que vos credentials sont configurés"
    ((ERRORS++))
fi

# ═══════════════════════════════════════════════════════════
# 2. Vérifier les fichiers Terraform
# ═══════════════════════════════════════════════════════════
echo ""
echo "2️⃣  Vérification des fichiers Terraform..."

REQUIRED_FILES=(
    "terraform/main.tf"
    "terraform/provider.tf"
    "terraform/variables.tf"
    "terraform/ci-deploy.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "   ✅ $file"
    else
        echo "   ❌ $file manquant"
        ((ERRORS++))
    fi
done

# ═══════════════════════════════════════════════════════════
# 3. Vérifier la configuration du backend S3
# ═══════════════════════════════════════════════════════════
echo ""
echo "3️⃣  Vérification du backend S3 dans provider.tf..."

if grep -q 'backend "s3"' terraform/provider.tf; then
    echo "   ✅ Backend S3 configuré"
    
    # Vérifier le bucket dans la config
    BUCKET=$(grep 'bucket' terraform/provider.tf | grep -v '#' | awk -F'"' '{print $2}' | head -1)
    if [ -n "$BUCKET" ]; then
        echo "      Bucket: ${BUCKET}"
    fi
else
    echo "   ⚠️  Backend S3 non configuré (sera créé par la CI)"
    ((WARNINGS++))
fi

# ═══════════════════════════════════════════════════════════
# 4. Vérifier le workflow GitHub Actions
# ═══════════════════════════════════════════════════════════
echo ""
echo "4️⃣  Vérification du workflow GitHub Actions..."

if [ -f ".github/workflows/ci-cd.yml" ]; then
    echo "   ✅ Workflow CI/CD présent"
    
    # Vérifier que le job terraform utilise le script ci-deploy.sh
    if grep -q "ci-deploy.sh" .github/workflows/ci-cd.yml; then
        echo "   ✅ Script ci-deploy.sh utilisé dans le workflow"
    else
        echo "   ⚠️  Script ci-deploy.sh non utilisé dans le workflow"
        ((WARNINGS++))
    fi
else
    echo "   ❌ Workflow CI/CD manquant"
    ((ERRORS++))
fi

# ═══════════════════════════════════════════════════════════
# 5. Vérifier les secrets GitHub (localement impossible, guide)
# ═══════════════════════════════════════════════════════════
echo ""
echo "5️⃣  Secrets GitHub requis (à vérifier dans Settings → Secrets):"
echo "   - AWS_ACCESS_KEY_ID"
echo "   - AWS_SECRET_ACCESS_KEY"
echo "   - AWS_SESSION_TOKEN"
echo "   - AWS_REGION"
echo "   - DOCKER_USERNAME"
echo "   - DOCKER_PASSWORD"

# ═══════════════════════════════════════════════════════════
# 6. Vérifier les limites AWS
# ═══════════════════════════════════════════════════════════
echo ""
echo "6️⃣  Vérification des limites AWS..."

if aws sts get-caller-identity &>/dev/null; then
    REGION="${AWS_REGION:-us-east-1}"
    
    # Compter les VPCs
    VPC_COUNT=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text 2>/dev/null || echo "?")
    echo "   VPCs utilisés: ${VPC_COUNT}/5"
    
    if [ "$VPC_COUNT" != "?" ] && [ "$VPC_COUNT" -ge 5 ]; then
        echo "   ⚠️  Limite de VPCs atteinte! Supprimez des VPCs non utilisés"
        ((WARNINGS++))
    fi
    
    # Compter les instances RDS
    RDS_COUNT=$(aws rds describe-db-instances --region "${REGION}" --query 'DBInstances | length(@)' --output text 2>/dev/null || echo "?")
    echo "   Instances RDS: ${RDS_COUNT}"
fi

# ═══════════════════════════════════════════════════════════
# 7. Vérifier les ressources existantes
# ═══════════════════════════════════════════════════════════
echo ""
echo "7️⃣  Vérification des ressources existantes..."

if aws sts get-caller-identity &>/dev/null; then
    REGION="${AWS_REGION:-us-east-1}"
    PROJECT_NAME="auth-service"
    
    # Key pair
    if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${PROJECT_NAME}-key" &>/dev/null; then
        echo "   ⚠️  Key pair ${PROJECT_NAME}-key existe (sera importée auto)"
    fi
    
    # DB Subnet Group
    if aws rds describe-db-subnet-groups --region "${REGION}" --db-subnet-group-name "${PROJECT_NAME}-db-subnet-group" &>/dev/null; then
        echo "   ⚠️  DB Subnet Group existe (sera importé auto)"
    fi
    
    # VPCs du projet
    VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
        --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
        --query 'Vpcs[*].VpcId' \
        --output text 2>/dev/null)
    
    if [ -n "$VPC_IDS" ]; then
        echo "   ⚠️  VPCs du projet trouvés: ${VPC_IDS}"
        echo "      (Si problème, utilisez ./terraform/auto-cleanup.sh)"
    fi
fi

# ═══════════════════════════════════════════════════════════
# Résumé
# ═══════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 Résumé de la vérification"
echo "═══════════════════════════════════════════════════════════"
echo "   Erreurs: ${ERRORS}"
echo "   Avertissements: ${WARNINGS}"
echo ""

if [ $ERRORS -eq 0 ]; then
    echo "✅ Configuration prête pour le déploiement!"
    echo ""
    echo "📝 Prochaines étapes:"
    echo "   1. git add ."
    echo "   2. git commit -m 'feat: configuration CI/CD automatique'"
    echo "   3. git push origin main"
    echo ""
    echo "🚀 La CI/CD gérera automatiquement:"
    echo "   • Création du backend S3"
    echo "   • Import des ressources existantes"
    echo "   • Déploiement de l'infrastructure (si nécessaire)"
    echo "   • Déploiement de l'application sur Kubernetes"
    exit 0
else
    echo "❌ Corrigez les erreurs avant de continuer"
    exit 1
fi
