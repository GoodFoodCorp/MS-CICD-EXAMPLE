#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Script de nettoyage des ressources AWS orphelines
# ═══════════════════════════════════════════════════════════

set -e

REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="auth-service"

echo "🧹 Nettoyage des ressources orphelines pour ${PROJECT_NAME}..."
echo "   Région: ${REGION}"
echo ""

# Fonction pour confirmer l'action
confirm() {
    read -p "$1 (y/N): " response
    case "$response" in
        [yY][eE][sS]|[yY]) 
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Lister les VPCs
echo "📋 Liste des VPCs dans la région ${REGION}:"
aws ec2 describe-vpcs --region "${REGION}" \
    --query 'Vpcs[*].[VpcId,Tags[?Key==`Name`].Value|[0],CidrBlock]' \
    --output table

echo ""
echo "⚠️  VPCs du projet ${PROJECT_NAME}:"
VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
    --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
    --query 'Vpcs[*].VpcId' \
    --output text)

if [ -z "$VPC_IDS" ]; then
    echo "   Aucun VPC trouvé avec le tag Project=${PROJECT_NAME}"
else
    for VPC_ID in $VPC_IDS; do
        echo "   - ${VPC_ID}"
        
        # Vérifier si le VPC a des instances en cours d'exécution
        INSTANCES=$(aws ec2 describe-instances --region "${REGION}" \
            --filters "Name=vpc-id,Values=${VPC_ID}" "Name=instance-state-name,Values=running" \
            --query 'Reservations[*].Instances[*].InstanceId' \
            --output text)
        
        if [ -n "$INSTANCES" ]; then
            echo "     ⚠️  Contient des instances en cours: ${INSTANCES}"
        fi
    done
fi

echo ""
echo "📋 Liste des Key Pairs:"
aws ec2 describe-key-pairs --region "${REGION}" \
    --query 'KeyPairs[*].[KeyName,KeyFingerprint]' \
    --output table

KEY_PAIR="${PROJECT_NAME}-key"
if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${KEY_PAIR}" 2>/dev/null; then
    echo ""
    if confirm "🗑️  Voulez-vous supprimer la key pair ${KEY_PAIR}?"; then
        aws ec2 delete-key-pair --region "${REGION}" --key-name "${KEY_PAIR}"
        echo "✅ Key pair ${KEY_PAIR} supprimée"
    fi
fi

echo ""
echo "📋 Liste des DB Subnet Groups:"
aws rds describe-db-subnet-groups --region "${REGION}" \
    --query 'DBSubnetGroups[*].[DBSubnetGroupName,VpcId]' \
    --output table

SUBNET_GROUP="${PROJECT_NAME}-db-subnet-group"
if aws rds describe-db-subnet-groups --region "${REGION}" \
    --db-subnet-group-name "${SUBNET_GROUP}" 2>/dev/null; then
    echo ""
    echo "⚠️  DB Subnet Group trouvé: ${SUBNET_GROUP}"
    echo "   Note: Ne peut être supprimé que s'il n'est pas utilisé par une instance RDS"
fi

echo ""
echo "📋 Liste des instances RDS:"
aws rds describe-db-instances --region "${REGION}" \
    --query 'DBInstances[*].[DBInstanceIdentifier,DBInstanceStatus,Engine,EngineVersion]' \
    --output table

echo ""
echo "💡 Actions recommandées:"
echo "   1. Vérifiez quelles ressources sont encore nécessaires"
echo "   2. Pour importer une ressource existante dans Terraform:"
echo "      terraform import <resource_type>.<resource_name> <resource_id>"
echo "      Exemple: terraform import aws_key_pair.deployer ${KEY_PAIR}"
echo ""
echo "   3. Pour supprimer un VPC, utilisez la console AWS ou:"
echo "      aws ec2 delete-vpc --vpc-id <vpc-id> --region ${REGION}"
echo "      (après avoir supprimé toutes les ressources associées)"
echo ""
echo "✅ Analyse terminée"
