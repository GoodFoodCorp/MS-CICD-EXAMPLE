#!/bin/bash

# ═══════════════════════════════════════════════════════════
# Script de notification pour les déploiements
# Simule un système de notification (Slack, Discord, Email, etc.)
# ═══════════════════════════════════════════════════════════

set -e

# Configuration
WEBHOOK_URL="${WEBHOOK_URL:-https://webhook.site/your-unique-id}"
SERVICE_NAME="${SERVICE_NAME:-auth-service}"
ENVIRONMENT="${ENVIRONMENT:-production}"

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ═══════════════════════════════════════════════════════════
# Fonction pour envoyer une notification
# ═══════════════════════════════════════════════════════════

send_notification() {
  local event_type="$1"
  local message="$2"
  local status="${3:-info}"
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  
  # Construire le payload JSON
  local payload=$(cat <<EOF
{
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "event": "$event_type",
  "message": "$message",
  "status": "$status",
  "timestamp": "$timestamp",
  "hostname": "$(hostname)",
  "metadata": {
    "user": "$(whoami)",
    "pwd": "$(pwd)"
  }
}
EOF
)
  
  # Afficher dans la console avec couleur
  case $status in
    success)
      echo -e "${GREEN}✅ [$event_type] $message${NC}"
      ;;
    error|failed)
      echo -e "${RED}❌ [$event_type] $message${NC}"
      ;;
    warning)
      echo -e "${YELLOW}⚠️  [$event_type] $message${NC}"
      ;;
    *)
      echo -e "${BLUE}ℹ️  [$event_type] $message${NC}"
      ;;
  esac
  
  # Envoyer au webhook
  response=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "$payload" 2>&1 || echo "000")
  
  http_code=$(echo "$response" | tail -n1)
  
  if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
    echo "  → Notification envoyée avec succès (HTTP $http_code)"
  else
    echo "  → Échec de l'envoi de la notification (HTTP $http_code)"
  fi
}

# ═══════════════════════════════════════════════════════════
# Types d'événements prédéfinis
# ═══════════════════════════════════════════════════════════

notify_deployment_start() {
  local version="$1"
  send_notification "deployment_start" "Démarrage du déploiement de la version $version" "info"
}

notify_deployment_success() {
  local version="$1"
  send_notification "deployment_success" "Déploiement réussi de la version $version" "success"
}

notify_deployment_failed() {
  local version="$1"
  local error="$2"
  send_notification "deployment_failed" "Échec du déploiement de la version $version: $error" "error"
}

notify_new_version_detected() {
  local from_version="$1"
  local to_version="$2"
  send_notification "new_version_detected" "Nouvelle version disponible: $from_version → $to_version" "info"
}

notify_auto_update_started() {
  local version="$1"
  send_notification "auto_update_started" "Mise à jour automatique démarrée vers la version $version" "info"
}

notify_auto_update_completed() {
  local version="$1"
  send_notification "auto_update_completed" "Mise à jour automatique terminée vers la version $version" "success"
}

notify_health_check_failed() {
  local message="$1"
  send_notification "health_check_failed" "Échec du health check: $message" "error"
}

notify_scaling_event() {
  local from_replicas="$1"
  local to_replicas="$2"
  send_notification "scaling_event" "Scaling: $from_replicas → $to_replicas replicas" "info"
}

notify_server_initialized() {
  local ip="$1"
  send_notification "server_initialized" "Serveur initialisé avec succès (IP: $ip)" "success"
}

notify_rollback() {
  local from_version="$1"
  local to_version="$2"
  send_notification "rollback" "Rollback effectué: $from_version → $to_version" "warning"
}

# ═══════════════════════════════════════════════════════════
# Tests
# ═══════════════════════════════════════════════════════════

if [ "$1" == "test" ]; then
  echo "🧪 Test du système de notifications"
  echo "===================================="
  echo ""
  
  notify_deployment_start "v1.2.3"
  sleep 1
  notify_new_version_detected "v1.2.2" "v1.2.3"
  sleep 1
  notify_auto_update_started "v1.2.3"
  sleep 1
  notify_deployment_success "v1.2.3"
  sleep 1
  notify_scaling_event "2" "3"
  
  echo ""
  echo "✅ Tests terminés"
  exit 0
fi

# ═══════════════════════════════════════════════════════════
# Exposition des fonctions pour utilisation externe
# ═══════════════════════════════════════════════════════════

# Si le script est sourcé, les fonctions sont disponibles
# Exemple: source notify.sh && notify_deployment_start "v1.0.0"

# Si exécuté directement avec des arguments
if [ $# -gt 0 ]; then
  case "$1" in
    deployment_start)
      notify_deployment_start "$2"
      ;;
    deployment_success)
      notify_deployment_success "$2"
      ;;
    deployment_failed)
      notify_deployment_failed "$2" "$3"
      ;;
    new_version)
      notify_new_version_detected "$2" "$3"
      ;;
    auto_update_started)
      notify_auto_update_started "$2"
      ;;
    auto_update_completed)
      notify_auto_update_completed "$2"
      ;;
    health_check_failed)
      notify_health_check_failed "$2"
      ;;
    scaling)
      notify_scaling_event "$2" "$3"
      ;;
    server_init)
      notify_server_initialized "$2"
      ;;
    rollback)
      notify_rollback "$2" "$3"
      ;;
    custom)
      send_notification "$2" "$3" "$4"
      ;;
    *)
      echo "Usage: $0 {test|deployment_start|deployment_success|deployment_failed|...} [args]"
      exit 1
      ;;
  esac
fi
