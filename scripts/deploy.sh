#!/bin/bash

# ═══════════════════════════════════════════════════════════
# Script de déploiement manuel
# Usage: ./deploy.sh [version]
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
IMAGE_NAME="${DOCKER_USERNAME:-votreusername}/auth-service"
VERSION="${1:-latest}"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   [DEPLOY] Déploiement de Auth Service${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  Service:    ${GREEN}$DEPLOYMENT${NC}"
echo -e "  Namespace:  ${GREEN}$NAMESPACE${NC}"
echo -e "  Version:    ${GREEN}$VERSION${NC}"
echo -e "  Image:      ${GREEN}$IMAGE_NAME:$VERSION${NC}"
echo ""

# Vérifier kubectl
if ! command -v kubectl &> /dev/null; then
  echo -e "${RED}[ERROR] kubectl n'est pas installé${NC}"
  exit 1
fi

# Vérifier la connexion au cluster
echo -e "${YELLOW}[CHECK] Vérification de la connexion au cluster...${NC}"
if ! kubectl cluster-info &> /dev/null; then
  echo -e "${RED}[ERROR] Impossible de se connecter au cluster Kubernetes${NC}"
  echo "Assurez-vous que KUBECONFIG est correctement configuré"
  exit 1
fi
echo -e "${GREEN}[OK] Connecté au cluster${NC}"
echo ""

# Vérifier que le namespace existe
echo -e "${YELLOW}[CHECK] Vérification du namespace...${NC}"
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
  echo -e "${YELLOW}[WARNING]  Le namespace $NAMESPACE n'existe pas. Création...${NC}"
  kubectl create namespace $NAMESPACE
fi
echo -e "${GREEN}[OK] Namespace OK${NC}"
echo ""

# Sauvegarder la version actuelle
CURRENT_IMAGE=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || echo "none")
echo -e "${BLUE}[INFO]  Version actuelle: $CURRENT_IMAGE${NC}"
echo ""

# Notification de début
if [ -f "$(dirname "$0")/notify.sh" ]; then
  source "$(dirname "$0")/notify.sh"
  notify_deployment_start "$VERSION"
fi

# Mise à jour de l'image
echo -e "${YELLOW} Mise à jour de l'image Docker...${NC}"
kubectl set image deployment/$DEPLOYMENT \
  auth-service=$IMAGE_NAME:$VERSION \
  -n $NAMESPACE

echo -e "${GREEN}[OK] Image mise à jour${NC}"
echo ""

# Attendre le rollout
echo -e "${YELLOW}[WAIT] Attente du déploiement...${NC}"
if kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE --timeout=5m; then
  echo -e "${GREEN}[OK] Déploiement réussi!${NC}"
  
  # Notification de succès
  if [ -f "$(dirname "$0")/notify.sh" ]; then
    notify_deployment_success "$VERSION"
  fi
else
  echo -e "${RED}[ERROR] Le déploiement a échoué${NC}"
  
  # Notification d'échec
  if [ -f "$(dirname "$0")/notify.sh" ]; then
    notify_deployment_failed "$VERSION" "Timeout"
  fi
  
  # Proposer un rollback
  echo ""
  echo -e "${YELLOW}Voulez-vous effectuer un rollback? (y/N)${NC}"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo -e "${YELLOW}🔙 Rollback en cours...${NC}"
    kubectl rollout undo deployment/$DEPLOYMENT -n $NAMESPACE
    kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE
    echo -e "${GREEN}[OK] Rollback effectué${NC}"
  fi
  
  exit 1
fi

echo ""

# Afficher l'état des pods
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   État des pods${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
kubectl get pods -n $NAMESPACE -l app=$DEPLOYMENT

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}[OK] Déploiement terminé avec succès!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
