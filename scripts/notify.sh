#!/bin/bash
# Script de notification pour les deploiements
# Configure WEBHOOK_URL pour activer les notifications

set -euo pipefail

# Configuration
WEBHOOK_URL="${WEBHOOK_URL:-}"
SERVICE_NAME="${SERVICE_NAME:-auth-service}"
ENVIRONMENT="${ENVIRONMENT:-production}"

send_notification() {
    local event_type="$1"
    local message="$2"
    local status="${3:-info}"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    case $status in
        success) echo "[OK]      [$event_type] $message" ;;
        error|failed) echo "[ERROR]   [$event_type] $message" ;;
        warning) echo "[WARNING] [$event_type] $message" ;;
        *) echo "[INFO]    [$event_type] $message" ;;
    esac

    [ -z "$WEBHOOK_URL" ] && return 0

    local payload
    payload=$(printf '{"service":"%s","environment":"%s","event":"%s","message":"%s","status":"%s","timestamp":"%s"}' \
        "$SERVICE_NAME" "$ENVIRONMENT" "$event_type" "$message" "$status" "$timestamp")

    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" -d "$payload" 2>/dev/null || echo "000")

    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo "         -> Notification envoyee (HTTP ${http_code})"
    else
        echo "         -> Echec notification (HTTP ${http_code})"
    fi
}

notify_deployment_start()   { send_notification "deployment_start"   "Deploiement version $1 demarre"  "info"; }
notify_deployment_success() { send_notification "deployment_success"  "Deploiement version $1 reussi"   "success"; }
notify_deployment_failed()  { send_notification "deployment_failed"   "Deploiement version $1 echoue: $2" "error"; }
notify_rollback()           { send_notification "rollback"            "Rollback de $1 vers $2"           "warning"; }

# Execution directe pour test
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    send_notification "test" "Test de notification" "info"
fi
