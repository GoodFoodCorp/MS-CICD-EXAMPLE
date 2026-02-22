#!/bin/bash

# ═══════════════════════════════════════════════════════════
# Script de monitoring
# Affiche l'état du déploiement, les logs, les métriques
# ═══════════════════════════════════════════════════════════

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
NAMESPACE="production"
DEPLOYMENT="auth-service"

clear

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   📊 Monitoring - Auth Service${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# ═══════════════════════════════════════════════════════════
# 1. État du Deployment
# ═══════════════════════════════════════════════════════════

echo -e "${CYAN}═══ État du Deployment ═══${NC}"
kubectl get deployment $DEPLOYMENT -n $NAMESPACE
echo ""

# ═══════════════════════════════════════════════════════════
# 2. État des Pods
# ═══════════════════════════════════════════════════════════

echo -e "${CYAN}═══ État des Pods ═══${NC}"
kubectl get pods -n $NAMESPACE -l app=$DEPLOYMENT -o wide
echo ""

# ═══════════════════════════════════════════════════════════
# 3. Image actuelle
# ═══════════════════════════════════════════════════════════

CURRENT_IMAGE=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
echo -e "${CYAN}═══ Image actuelle ═══${NC}"
echo -e "  ${GREEN}$CURRENT_IMAGE${NC}"
echo ""

# ═══════════════════════════════════════════════════════════
# 4. Replicas
# ═══════════════════════════════════════════════════════════

DESIRED=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.spec.replicas}')
CURRENT=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.status.replicas}')
READY=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')

echo -e "${CYAN}═══ Replicas ═══${NC}"
echo -e "  Désiré:  ${BLUE}$DESIRED${NC}"
echo -e "  Actuel:  ${BLUE}${CURRENT:-0}${NC}"
echo -e "  Prêt:    ${GREEN}${READY:-0}${NC}"
echo ""

# ═══════════════════════════════════════════════════════════
# 5. HPA Status
# ═══════════════════════════════════════════════════════════

echo -e "${CYAN}═══ Horizontal Pod Autoscaler ═══${NC}"
if kubectl get hpa auth-service-hpa -n $NAMESPACE &> /dev/null; then
  kubectl get hpa auth-service-hpa -n $NAMESPACE
else
  echo -e "  ${YELLOW}HPA non configuré${NC}"
fi
echo ""

# ═══════════════════════════════════════════════════════════
# 6. Service
# ═══════════════════════════════════════════════════════════

echo -e "${CYAN}═══ Service ═══${NC}"
kubectl get service $DEPLOYMENT -n $NAMESPACE
echo ""

# ═══════════════════════════════════════════════════════════
# 7. Événements récents
# ═══════════════════════════════════════════════════════════

echo -e "${CYAN}═══ Événements récents ═══${NC}"
kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | grep $DEPLOYMENT | tail -n 10
echo ""

# ═══════════════════════════════════════════════════════════
# 8. Logs des pods (dernières lignes)
# ═══════════════════════════════════════════════════════════

echo -e "${CYAN}═══ Logs récents ═══${NC}"
POD_NAME=$(kubectl get pods -n $NAMESPACE -l app=$DEPLOYMENT -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ ! -z "$POD_NAME" ]; then
  echo -e "  Pod: ${GREEN}$POD_NAME${NC}"
  echo ""
  kubectl logs $POD_NAME -n $NAMESPACE --tail=20 | sed 's/^/  /'
else
  echo -e "  ${RED}Aucun pod en cours d'exécution${NC}"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}💡 Commandes utiles:${NC}"
echo -e "  • Logs en temps réel:    ${CYAN}kubectl logs -f $POD_NAME -n $NAMESPACE${NC}"
echo -e "  • Décrire un pod:        ${CYAN}kubectl describe pod $POD_NAME -n $NAMESPACE${NC}"
echo -e "  • Shell dans le pod:     ${CYAN}kubectl exec -it $POD_NAME -n $NAMESPACE -- /bin/sh${NC}"
echo -e "  • Historique déploiement: ${CYAN}kubectl rollout history deployment/$DEPLOYMENT -n $NAMESPACE${NC}"
echo ""
