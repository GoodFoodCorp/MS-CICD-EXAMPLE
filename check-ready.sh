#!/bin/bash
# Verification finale avant deploiement

set -euo pipefail

echo "[CHECK] Verification de l'environnement..."
echo ""

ERRORS=0

# Verifier AWS CLI
if ! command -v aws &>/dev/null; then
    echo "[ERROR] AWS CLI non installe"
    ((ERRORS++))
else
    echo "[OK] AWS CLI installe"
fi

# Verifier les credentials
if aws sts get-caller-identity &>/dev/null; then
    ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
    echo "[OK] Credentials AWS valides (Compte: ${ACCOUNT})"
else
    echo "[ERROR] Credentials AWS invalides"
    ((ERRORS++))
fi

# Verifier Terraform
if ! command -v terraform &>/dev/null; then
    echo "[ERROR] Terraform non installe"
    ((ERRORS++))
else
    echo "[OK] Terraform installe"
fi

# Verifier les VPCs
REGION="${AWS_REGION:-us-east-1}"
VPC_COUNT=$(aws ec2 describe-vpcs --region "${REGION}" --query 'Vpcs | length(@)' --output text 2>/dev/null || echo "?")
echo ""
echo "[INFO] VPCs dans ${REGION}: ${VPC_COUNT}/5"

if [ "$VPC_COUNT" != "?" ] && [ "$VPC_COUNT" -ge 5 ]; then
    echo "[WARNING] ATTENTION : Limite de VPCs atteinte!"
    echo "   Supprimez au moins 1 VPC avant de continuer"
    ((ERRORS++))
fi

# Verifier les fichiers
echo ""
echo "[FILES] Verification des fichiers..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/terraform" || { echo "[ERROR] Dossier terraform introuvable"; exit 1; }

required_files=("ci-deploy.sh" "pre-deploy-cleanup.sh" "main.tf" "provider.tf")
for file in "${required_files[@]}"; do
    if [ -f "${file}" ]; then
        echo "   [OK] terraform/${file}"
    else
        echo "   [ERROR] terraform/${file} manquant"
        ((ERRORS++))
    fi
done

cd "${SCRIPT_DIR}"

echo ""
echo "================================================================"
echo "[INFO] Resume: Erreurs=${ERRORS}"
echo "================================================================"

if [ $ERRORS -eq 0 ]; then
    echo "[OK] Environnement pret"
    exit 0
else
    echo "[ERROR] Corrigez les erreurs avant de continuer"
    exit 1
fi
