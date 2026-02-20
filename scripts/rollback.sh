#!/bin/bash

# ═══════════════════════════════════════════════════════════
# Script de rollback
# Usage: ./rollback.sh [revision]
# ═══════════════════════════════════════════════════════════

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
NAMESPACE="production"
DEPLOYMENT="auth-service"
REVISION="${1}"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   🔙 Rollback de Auth Service${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Afficher l'historique
echo -e "${YELLOW}📜 Historique des déploiements:${NC}"
echo ""
kubectl rollout history deployment/$DEPLOYMENT -n $NAMESPACE

echo ""

# Si aucune révision spécifiée, demander
if [ -z "$REVISION" ]; then
  echo -e "${YELLOW}Entrez le numéro de révision pour le rollback (ou appuyez sur Entrée pour la révision précédente):${NC}"
  read -r REVISION
fi

# Rollback
echo ""
if [ -z "$REVISION" ]; then
  echo -e "${YELLOW}🔙 Rollback vers la révision précédente...${NC}"
  kubectl rollout undo deployment/$DEPLOYMENT -n $NAMESPACE
else
  echo -e "${YELLOW}🔙 Rollback vers la révision $REVISION...${NC}"
  kubectl rollout undo deployment/$DEPLOYMENT -n $NAMESPACE --to-revision=$REVISION
fi

# Attendre le rollout
echo ""
echo -e "${YELLOW}⏳ Attente du rollback...${NC}"
if kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE --timeout=5m; then
  echo -e "${GREEN}✅ Rollback réussi!${NC}"
  
  # Notification
  if [ -f "$(dirname "$0")/notify.sh" ]; then
    source "$(dirname "$0")/notify.sh"
    CURRENT_IMAGE=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
    notify_rollback "unknown" "$CURRENT_IMAGE"
  fi
else
  echo -e "${RED}❌ Le rollback a échoué${NC}"
  exit 1
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   État des pods${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
kubectl get pods -n $NAMESPACE -l app=$DEPLOYMENT

echo ""
echo -e "${GREEN}✅ Rollback terminé${NC}"
