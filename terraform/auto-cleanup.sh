#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Script de nettoyage AUTOMATIQUE des ressources orphelines
# ═══════════════════════════════════════════════════════════
# ⚠️  ATTENTION : Ce script supprime automatiquement des ressources !
# Utilisez-le uniquement si vous êtes sûr de ce que vous faites.
# ═══════════════════════════════════════════════════════════

set -e

REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="auth-service"
DRY_RUN="${DRY_RUN:-false}"

echo "🧹 Nettoyage automatique des ressources orphelines pour ${PROJECT_NAME}"
echo "   Région: ${REGION}"
echo "   Mode: $([ "$DRY_RUN" = "true" ] && echo 'DRY RUN (simulation)' || echo 'RÉEL')"
echo ""

# Compteurs
DELETED=0
ERRORS=0

# ═══════════════════════════════════════════════════════════
# 1. Nettoyer la Key Pair
# ═══════════════════════════════════════════════════════════
echo "🔑 Vérification de la Key Pair..."
KEY_PAIR="${PROJECT_NAME}-key"

if aws ec2 describe-key-pairs --region "${REGION}" --key-names "${KEY_PAIR}" &>/dev/null; then
    echo "   ➜ Key pair trouvée: ${KEY_PAIR}"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "   [DRY RUN] Suppression simulée de ${KEY_PAIR}"
    else
        if aws ec2 delete-key-pair --region "${REGION}" --key-name "${KEY_PAIR}"; then
            echo "   ✅ Key pair ${KEY_PAIR} supprimée"
            ((DELETED++))
        else
            echo "   ❌ Erreur lors de la suppression de ${KEY_PAIR}"
            ((ERRORS++))
        fi
    fi
else
    echo "   ℹ️  Aucune key pair ${KEY_PAIR} trouvée"
fi

# ═══════════════════════════════════════════════════════════
# 2. Nettoyer les instances RDS (arrêt seulement, pas suppression)
# ═══════════════════════════════════════════════════════════
echo ""
echo "🗄️  Vérification des instances RDS..."
DB_IDENTIFIER="${PROJECT_NAME}-postgres"

if aws rds describe-db-instances --region "${REGION}" --db-instance-identifier "${DB_IDENTIFIER}" &>/dev/null; then
    DB_STATUS=$(aws rds describe-db-instances --region "${REGION}" \
        --db-instance-identifier "${DB_IDENTIFIER}" \
        --query 'DBInstances[0].DBInstanceStatus' \
        --output text)
    
    echo "   ➜ Instance RDS trouvée: ${DB_IDENTIFIER} (statut: ${DB_STATUS})"
    echo "   ⚠️  Les instances RDS ne sont PAS supprimées automatiquement"
    echo "   💡 Pour supprimer manuellement:"
    echo "      aws rds delete-db-instance --db-instance-identifier ${DB_IDENTIFIER} --skip-final-snapshot --region ${REGION}"
else
    echo "   ℹ️  Aucune instance RDS ${DB_IDENTIFIER} trouvée"
fi

# ═══════════════════════════════════════════════════════════
# 3. Nettoyer le DB Subnet Group (seulement si RDS n'existe pas)
# ═══════════════════════════════════════════════════════════
echo ""
echo "📊 Vérification du DB Subnet Group..."
SUBNET_GROUP="${PROJECT_NAME}-db-subnet-group"

if aws rds describe-db-subnet-groups --region "${REGION}" --db-subnet-group-name "${SUBNET_GROUP}" &>/dev/null; then
    echo "   ➜ DB Subnet Group trouvé: ${SUBNET_GROUP}"
    
    # Vérifier si utilisé par une instance RDS
    if aws rds describe-db-instances --region "${REGION}" --db-instance-identifier "${DB_IDENTIFIER}" &>/dev/null; then
        echo "   ⚠️  Le subnet group est utilisé par l'instance RDS ${DB_IDENTIFIER}"
        echo "   ℹ️  Supprimez d'abord l'instance RDS"
    else
        if [ "$DRY_RUN" = "true" ]; then
            echo "   [DRY RUN] Suppression simulée de ${SUBNET_GROUP}"
        else
            if aws rds delete-db-subnet-group --region "${REGION}" --db-subnet-group-name "${SUBNET_GROUP}"; then
                echo "   ✅ DB Subnet Group ${SUBNET_GROUP} supprimé"
                ((DELETED++))
            else
                echo "   ❌ Erreur lors de la suppression de ${SUBNET_GROUP}"
                ((ERRORS++))
            fi
        fi
    fi
else
    echo "   ℹ️  Aucun DB Subnet Group ${SUBNET_GROUP} trouvé"
fi

# ═══════════════════════════════════════════════════════════
# 4. Lister les VPCs (pas de suppression automatique)
# ═══════════════════════════════════════════════════════════
echo ""
echo "🌐 Vérification des VPCs..."

VPC_IDS=$(aws ec2 describe-vpcs --region "${REGION}" \
    --filters "Name=tag:Project,Values=${PROJECT_NAME}" \
    --query 'Vpcs[*].VpcId' \
    --output text)

if [ -n "$VPC_IDS" ]; then
    echo "   ⚠️  VPCs du projet ${PROJECT_NAME} trouvés:"
    for VPC_ID in $VPC_IDS; do
        VPC_NAME=$(aws ec2 describe-vpcs --region "${REGION}" \
            --vpc-ids "${VPC_ID}" \
            --query 'Vpcs[0].Tags[?Key==`Name`].Value' \
            --output text)
        
        INSTANCE_COUNT=$(aws ec2 describe-instances --region "${REGION}" \
            --filters "Name=vpc-id,Values=${VPC_ID}" "Name=instance-state-name,Values=running,stopped" \
            --query 'Reservations[*].Instances[*].InstanceId' \
            --output text | wc -w | tr -d ' ')
        
        echo "      • ${VPC_ID} (${VPC_NAME}) - ${INSTANCE_COUNT} instance(s)"
    done
    
    echo ""
    echo "   ⚠️  Les VPCs ne sont PAS supprimés automatiquement (trop risqué)"
    echo "   💡 Pour supprimer les VPCs, utilisez la console AWS ou le script cleanup-orphaned.sh interactif"
else
    echo "   ℹ️  Aucun VPC avec le tag Project=${PROJECT_NAME} trouvé"
fi

# ═══════════════════════════════════════════════════════════
# 5. Nettoyer les Security Groups orphelins
# ═══════════════════════════════════════════════════════════
echo ""
echo "🔒 Vérification des Security Groups..."

if [ -n "$VPC_IDS" ]; then
    for VPC_ID in $VPC_IDS; do
        SG_IDS=$(aws ec2 describe-security-groups --region "${REGION}" \
            --filters "Name=vpc-id,Values=${VPC_ID}" "Name=tag:Project,Values=${PROJECT_NAME}" \
            --query 'SecurityGroups[?GroupName!=`default`].GroupId' \
            --output text)
        
        for SG_ID in $SG_IDS; do
            SG_NAME=$(aws ec2 describe-security-groups --region "${REGION}" \
                --group-ids "${SG_ID}" \
                --query 'SecurityGroups[0].GroupName' \
                --output text)
            
            # Vérifier si utilisé
            ATTACHED=$(aws ec2 describe-network-interfaces --region "${REGION}" \
                --filters "Name=group-id,Values=${SG_ID}" \
                --query 'NetworkInterfaces[*].NetworkInterfaceId' \
                --output text)
            
            if [ -z "$ATTACHED" ]; then
                echo "   ➜ Security Group orphelin: ${SG_ID} (${SG_NAME})"
                
                if [ "$DRY_RUN" = "true" ]; then
                    echo "      [DRY RUN] Suppression simulée"
                else
                    if aws ec2 delete-security-group --region "${REGION}" --group-id "${SG_ID}" 2>/dev/null; then
                        echo "      ✅ Security Group supprimé"
                        ((DELETED++))
                    else
                        echo "      ⚠️  Impossible de supprimer (dépendances?)"
                    fi
                fi
            fi
        done
    done
fi

# ═══════════════════════════════════════════════════════════
# Résumé
# ═══════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 Résumé du nettoyage"
echo "═══════════════════════════════════════════════════════════"
echo "   Mode: $([ "$DRY_RUN" = "true" ] && echo 'DRY RUN' || echo 'RÉEL')"
echo "   Ressources supprimées: ${DELETED}"
echo "   Erreurs: ${ERRORS}"
echo ""

if [ "$DRY_RUN" = "true" ]; then
    echo "💡 Pour exécuter en mode réel:"
    echo "   DRY_RUN=false ./auto-cleanup.sh"
    echo ""
fi

echo "✅ Nettoyage terminé!"
echo ""
echo "📝 Prochaines étapes recommandées:"
echo "   1. Vérifiez les ressources restantes dans la console AWS"
echo "   2. Si des VPCs existent, supprimez-les manuellement ou via Terraform"
echo "   3. Exécutez 'terraform init -migrate-state' pour migrer l'état vers S3"
echo "   4. Exécutez 'terraform plan' pour voir ce qui sera créé"

exit 0
