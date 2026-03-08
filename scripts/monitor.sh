#!/bin/bash
# Script de monitoring - Auth Service
# Affiche l'etat du deploiement, les logs et les metriques

set -euo pipefail

# Configuration
NAMESPACE="production"
DEPLOYMENT="auth-service"

clear

echo "====================================================="
echo "   Monitoring - Auth Service"
echo "====================================================="
echo ""

# 1. Etat du Deployment
echo "--- Deployment ---"
kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE"
echo ""

# 2. Etat des Pods
echo "--- Pods ---"
kubectl get pods -n "$NAMESPACE" -l "app=$DEPLOYMENT" -o wide
echo ""

# 3. Image actuelle
CURRENT_IMAGE=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "--- Image actuelle ---"
echo "  ${CURRENT_IMAGE}"
echo ""

# 4. Replicas
DESIRED=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
CURRENT=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.status.replicas}' 2>/dev/null || echo "0")
READY=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")

echo "--- Replicas ---"
echo "  Desire:  ${DESIRED}"
echo "  Actuel:  ${CURRENT:-0}"
echo "  Pret:    ${READY:-0}"
echo ""

# 5. HPA Status
echo "--- Horizontal Pod Autoscaler ---"
if kubectl get hpa auth-service-hpa -n "$NAMESPACE" &>/dev/null; then
    kubectl get hpa auth-service-hpa -n "$NAMESPACE"
else
    echo "  HPA non configure"
fi
echo ""

# 6. Service
echo "--- Service ---"
kubectl get service "$DEPLOYMENT" -n "$NAMESPACE"
echo ""

# 7. Evenements recents
echo "--- Evenements recents ---"
kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' | grep "$DEPLOYMENT" | tail -n 10 || true
echo ""

# 8. Logs des pods
echo "--- Logs recents ---"
POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l "app=$DEPLOYMENT" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -n "$POD_NAME" ]; then
    echo "  Pod: ${POD_NAME}"
    echo ""
    kubectl logs "$POD_NAME" -n "$NAMESPACE" --tail=20 | sed 's/^/  /'
else
    echo "  [ERROR] Aucun pod en cours d'execution"
fi

echo ""
echo "====================================================="
echo ""
echo "Commandes utiles:"
echo "  Logs en continu:        kubectl logs -f ${POD_NAME:-<pod>} -n ${NAMESPACE}"
echo "  Decrire un pod:         kubectl describe pod ${POD_NAME:-<pod>} -n ${NAMESPACE}"
echo "  Shell dans le pod:      kubectl exec -it ${POD_NAME:-<pod>} -n ${NAMESPACE} -- /bin/sh"
echo "  Historique deploiement: kubectl rollout history deployment/${DEPLOYMENT} -n ${NAMESPACE}"
echo ""
