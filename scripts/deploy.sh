#!/bin/bash
# Script de deploiement manuel
# Usage: ./deploy.sh [version]

set -euo pipefail

# Configuration
NAMESPACE="production"
DEPLOYMENT="auth-service"
IMAGE_NAME="${DOCKER_USERNAME}/auth-service"
VERSION="${1:-latest}"

echo "--------------------------------------------"
echo "   Deploiement de Auth Service"
echo "--------------------------------------------"
echo ""
echo "  Service:    $DEPLOYMENT"
echo "  Namespace:  $NAMESPACE"
echo "  Version:    $VERSION"
echo "  Image:      $IMAGE_NAME:$VERSION"
echo ""

# Verifier kubectl
if ! command -v kubectl &>/dev/null; then
    echo "[ERROR] kubectl n'est pas installe"
    exit 1
fi

# Verifier la connexion au cluster
echo "[CHECK] Connexion au cluster..."
if ! kubectl cluster-info &>/dev/null; then
    echo "[ERROR] Impossible de se connecter au cluster Kubernetes"
    echo "Assurez-vous que KUBECONFIG est correctement configure"
    exit 1
fi
echo "[OK] Connecte au cluster"
echo ""

# Verifier que le namespace existe
echo "[CHECK] Namespace..."
if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
    echo "[INFO] Le namespace $NAMESPACE n'existe pas - creation..."
    kubectl create namespace "$NAMESPACE"
fi
echo "[OK] Namespace OK"
echo ""

# Sauvegarder la version actuelle
CURRENT_IMAGE=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || echo "none")
echo "[INFO] Version actuelle: $CURRENT_IMAGE"
echo ""

# Notification de debut
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "${SCRIPT_DIR}/notify.sh" ]; then
    # shellcheck source=scripts/notify.sh
    source "${SCRIPT_DIR}/notify.sh"
    notify_deployment_start "$VERSION"
fi

# Mise a jour de l'image
echo "[DEPLOY] Mise a jour de l'image Docker..."
kubectl set image "deployment/$DEPLOYMENT" \
    "auth-service=$IMAGE_NAME:$VERSION" \
    -n "$NAMESPACE"
echo "[OK] Image mise a jour"
echo ""

# Attendre le rollout
echo "[WAIT] Attente du deploiement (timeout: 5m)..."
if kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=5m; then
    echo "[OK] Deploiement reussi!"

    if [ -f "${SCRIPT_DIR}/notify.sh" ]; then
        notify_deployment_success "$VERSION"
    fi
else
    echo "[ERROR] Le deploiement a echoue"

    if [ -f "${SCRIPT_DIR}/notify.sh" ]; then
        notify_deployment_failed "$VERSION" "Timeout"
    fi

    echo ""
    echo "Voulez-vous effectuer un rollback? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo "[ROLLBACK] Rollback en cours..."
        kubectl rollout undo "deployment/$DEPLOYMENT" -n "$NAMESPACE"
        kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE"
        echo "[OK] Rollback effectue"
    fi

    exit 1
fi

echo ""
echo "--------------------------------------------"
echo "   Etat des pods"
echo "--------------------------------------------"
kubectl get pods -n "$NAMESPACE" -l "app=$DEPLOYMENT"

echo ""
echo "[OK] Deploiement termine avec succes!"
