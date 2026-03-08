#!/bin/bash
# Script de test pour l'Auth Service

set -euo pipefail

echo "[TEST] Auth Service - Test Suite"
echo "================================="
echo ""

# Executer tous les tests
echo "[1/3] Execution de tous les tests..."
if go test ./... -v; then
    echo "[OK] Tous les tests sont passes"
else
    echo "[ERROR] Certains tests ont echoue"
    exit 1
fi

echo ""

# Rapport de couverture
echo "[2/3] Rapport de couverture..."
go test ./... -coverprofile=coverage.out -covermode=count
go tool cover -func=coverage.out

echo ""

# Rapport HTML
echo "[3/3] Rapport HTML..."
go tool cover -html=coverage.out -o coverage.html
echo "[OK] Rapport genere: coverage.html"

echo ""
echo "[OK] Suite de tests completee avec succes"
echo ""
echo "[INFO] Pour ouvrir le rapport HTML:"
echo "   open coverage.html"
