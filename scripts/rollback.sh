#!/bin/bash
# Script de rollback
# Usage: ./rollback.sh [revision]

set -euo pipefail

# Configuration
NAMESPACE="production"
DEPLOYMENT="auth-service"
REVISION="${1:-}"

echo "--------------------------------------------"
echo "   Rollback de Auth Service"
echo "--------------------------------------------"
echo ""

# Afficher l'historique
echo "[INFO] Historique des deploiements:"
echo ""
kubectl rollout history "deployment/$DEPLOYMENT" -n "$NAMESPACE"
echo ""

# Si aucune revision specifiee, demander
if [ -z "$REVISION" ]; then
    echo "Entrez le numero de revision (ou Entree pour la revision precedente):"
    read -r REVISION
fi

# Rollback
if [ -z "$REVISION" ]; then
    echo "[ROLLBACK] Rollback vers la revision precedente..."
    kubectl rollout undo "deployment/$DEPLOYMENT" -n "$NAMESPACE"
else
    echo "[ROLLBACK] Rollback vers la revision $REVISION..."
    kubectl rollout undo "deployment/$DEPLOYMENT" -n "$NAMESPACE" --to-revision="$REVISION"
fi

# Attendre le rollout
echo ""
echo "[WAIT] Attente du rollback..."
if kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=5m; then
    echo "[OK] Rollback reussi!"

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [ -f "${SCRIPT_DIR}/notify.sh" ]; then
        # shellcheck source=scripts/notify.sh
        source "${SCRIPT_DIR}/notify.sh"
        CURRENT_IMAGE=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
        notify_rollback "unknown" "$CURRENT_IMAGE"
    fi
else
    echo "[ERROR] Le rollback a echoue"
    exit 1
fi

echo ""
echo "--------------------------------------------"
echo "   Etat des pods"
echo "--------------------------------------------"
kubectl get pods -n "$NAMESPACE" -l "app=$DEPLOYMENT"

echo ""
echo "[OK] Rollback termine"
