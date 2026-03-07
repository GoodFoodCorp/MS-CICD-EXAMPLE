#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Script de nettoyage rapide avant déploiement CI/CD
# ═══════════════════════════════════════════════════════════

set -e

REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="auth-service"

echo "🧹 Nettoyage pré-déploiement pour ${PROJECT_NAME}"
echo "   Région: ${REGION}"
echo ""

# ═══════════════════════════════════════════════════════════
# 1. Supprimer la key pair si elle existe
# ═══════════════════════════════════════════════════════════
echo " Nettoyage Key Pair..."
KEY_PAIR="${PROJECT_NAME}-key"

if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${KEY_PAIR}" &>/dev/null; then
    echo "   ➜ Suppression de ${KEY_PAIR}..."
    aws ec2 delete-key-pair --region "${REGION}" --key-name "${KEY_PAIR}"
    echo "   [OK] Key pair supprimée"
else
    echo "   [INFO]  Key pair n'existe pas"
fi

# ═══════════════════════════════════════════════════════════
# 2. Supprimer le DB Subnet Group si possible
# ═══════════════════════════════════════════════════════════
echo ""
echo "[INFO] Nettoyage DB Subnet Group..."
SUBNET_GROUP="${PROJECT_NAME}-db-subnet-group"

# Vérifier si une instance RDS utilise ce subnet group
DB_IDENTIFIER="${PROJECT_NAME}-postgres"
if aws rds describe-db-instances --region "${REGION}" --db-instance-identifier "${DB_IDENTIFIER}" &>/dev/null; then
    echo "   [WARNING]  Instance RDS ${DB_IDENTIFIER} existe encore"
    echo "   [INFO]  Le subnet group ne peut pas être supprimé"
else
    if aws rds describe-db-subnet-groups --region "${REGION}" --db-subnet-group-name "${SUBNET_GROUP}" &>/dev/null; then
        echo "   ➜ Suppression du subnet group ${SUBNET_GROUP}..."
        aws rds delete-db-subnet-group --region "${REGION}" --db-subnet-group-name "${SUBNET_GROUP}"
        echo "   [OK] Subnet group supprimé"
    else
        echo "   [INFO]  Subnet group n'existe pas"
    fi
fi

# ═══════════════════════════════════════════════════════════
# 3. Afficher les VPCs et leur utilisation
# ═══════════════════════════════════════════════════════════
echo ""
echo "🌐 Analyse des VPCs..."

TOTAL_VPCS=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text)
echo "   Total VPCs dans la région: ${TOTAL_VPCS}/5"

if [ "$TOTAL_VPCS" -ge 5 ]; then
    echo ""
    echo "   [WARNING]  LIMITE ATTEINTE ! Détails des VPCs:"
    echo ""
    
    aws ec2 describe-vpcs --region "${REGION}" \
        --query 'Vpcs[*].[VpcId,CidrBlock,Tags[?Key==`Name`].Value|[0],Tags[?Key==`Project`].Value|[0]]' \
        --output text | while read VPC_ID CIDR NAME PROJECT; do
        
        # Compter les ressources dans ce VPC
        INSTANCES=$(aws ec2 describe-instances --region "${REGION}" \
            --filters "Name=vpc-id,Values=${VPC_ID}" "Name=instance-state-name,Values=running,stopped" \
            --query 'Reservations[*].Instances[*].InstanceId' \
            --output text | wc -w | tr -d ' ')
        
        RDS=$(aws rds describe-db-instances --region "${REGION}" \
            --query "DBInstances[?DBSubnetGroup.VpcId=='${VPC_ID}'].DBInstanceIdentifier" \
            --output text | wc -w | tr -d ' ')
        
        echo "    VPC: ${VPC_ID}"
        echo "      CIDR: ${CIDR}"
        echo "      Name: ${NAME:-N/A}"
        echo "      Project: ${PROJECT:-N/A}"
        echo "      Instances EC2: ${INSTANCES}"
        echo "      Instances RDS: ${RDS}"
        
        if [ "${PROJECT}" = "${PROJECT_NAME}" ] && [ "${INSTANCES}" = "0" ] && [ "${RDS}" = "0" ]; then
            echo "      [OK] Ce VPC du projet est vide (peut être supprimé)"
        fi
        echo ""
    done
    
    echo ""
    echo "💡 Pour supprimer les VPCs du projet vides:"
    echo "   Utilisez la console AWS ou l'outil de nettoyage complet:"
    echo "   ./auto-cleanup.sh"
    echo ""
    echo "   OU supprimez manuellement avec:"
    VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
        --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
        --query 'Vpcs[*].VpcId' \
        --output text)
    
    for VPC_ID in $VPC_IDS; do
        echo "   aws ec2 delete-vpc --vpc-id ${VPC_ID} --region ${REGION}"
        echo "   (Supprimez d'abord tous les sous-réseaux, IGW, etc.)"
    done
fi

# ═══════════════════════════════════════════════════════════
# 4. Nettoyer l'état Terraform local si présent
# ═══════════════════════════════════════════════════════════
echo ""
echo "🔧 Nettoyage des fichiers Terraform locaux..."

if [ -f "terraform.tfstate" ]; then
    echo "   ➜ Suppression de terraform.tfstate local..."
    rm -f terraform.tfstate terraform.tfstate.backup
    echo "   [OK] État local supprimé"
fi

if [ -f "backend-override.tf" ]; then
    echo "   ➜ Suppression de backend-override.tf..."
    rm -f backend-override.tf
    echo "   [OK] Backend override supprimé"
fi

if [ -d ".terraform" ]; then
    echo "   ➜ Nettoyage du dossier .terraform..."
    rm -rf .terraform
    echo "   [OK] Dossier .terraform nettoyé"
fi

# ═══════════════════════════════════════════════════════════
# Résumé
# ═══════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════"
echo "[OK] Nettoyage terminé"
echo "═══════════════════════════════════════════════════════════"

if [ "$TOTAL_VPCS" -lt 5 ]; then
    echo ""
    echo "[DEPLOY] Prêt pour le déploiement !"
    echo "   Vous pouvez maintenant exécuter:"
    echo "   ./ci-deploy.sh"
else
    echo ""
    echo "[WARNING]  Action requise: Supprimez des VPCs pour continuer"
    echo "   La limite de VPCs (5) est atteinte."
    exit 1
fi
