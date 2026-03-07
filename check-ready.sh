#!/bin/bash
# ═══════════════════════════════════════════════════════════
# Vérification finale avant déploiement
# ═══════════════════════════════════════════════════════════

set -e

echo "[CHECK] Vérification de l'environnement..."
echo ""

ERRORS=0

# Vérifier AWS CLI
if ! command -v aws &>/dev/null; then
    echo "[ERROR] AWS CLI non installé"
    ((ERRORS++))
else
    echo "[OK] AWS CLI installé"
fi

# Vérifier les credentials
if aws sts get-caller-identity &>/dev/null; then
    ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
    echo "[OK] Credentials AWS valides (Compte: ${ACCOUNT})"
else
    echo "[ERROR] Credentials AWS invalides"
    ((ERRORS++))
fi

# Vérifier Terraform
if ! command -v terraform &>/dev/null; then
    echo "[ERROR] Terraform non installé"
    ((ERRORS++))
else
    echo "[OK] Terraform installé"
fi

# Vérifier les VPCs
REGION="${AWS_REGION:-us-east-1}"
VPC_COUNT=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text 2>/dev/null || echo "?")
echo ""
echo "[INFO] VPCs dans ${REGION}: ${VPC_COUNT}/5"

if [ "$VPC_COUNT" != "?" ] && [ "$VPC_COUNT" -ge 5 ]; then
    echo "[WARNING]  ATTENTION : Limite de VPCs atteinte!"
    echo "   Supprimez au moins 1 VPC avant de continuer"
    echo ""
    echo "   Console AWS : https://console.aws.amazon.com/vpc/"
    ((ERRORS++))
fi

# Vérifier les fichiers
echo ""
echo "[FILES] Vérification des fichiers..."
cd terraform 2>/dev/null || { echo "[ERROR] Dossier terraform introuvable"; exit 1; }

required_files=("ci-deploy.sh" "pre-deploy-cleanup.sh" "main.tf" "provider.tf")
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "   [OK] $file"
    else
        echo "   [ERROR] $file manquant"
        ((ERRORS++))
    fi
done

# Résumé
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ $ERRORS -eq 0 ]; then
    echo "[OK] Tout est prêt pour le déploiement!"
    echo ""
    echo "[DEPLOY] Prochaines étapes:"
    echo "   1. cd terraform && ./pre-deploy-cleanup.sh"
    echo "   2. ./ci-deploy.sh"
    echo ""
    echo "   Ou directement:"
    echo "   git add . && git commit -m 'deploy' && git push"
    exit 0
else
    echo "[ERROR] $ERRORS erreur(s) détectée(s)"
    echo "   Corrigez les erreurs ci-dessus avant de continuer"
    exit 1
fi
