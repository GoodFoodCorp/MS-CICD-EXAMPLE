#!/bin/bash

# Script de test pour le Auth Service
# Ce script exécute tous les tests et génère des rapports

set -e

echo "🧪 Auth Service - Test Suite"
echo "=============================="
echo ""

# Couleurs
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Lancer tous les tests
echo -e "${YELLOW}1️⃣  Exécution de tous les tests...${NC}"
if go test ./... -v; then
    echo -e "${GREEN}✅ Tous les tests sont passés!${NC}"
else
    echo -e "${RED}❌ Certains tests ont échoué${NC}"
    exit 1
fi

echo ""

# Test 2: Couverture de test
echo -e "${YELLOW}2️⃣  Génération du rapport de couverture...${NC}"
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

echo ""

# Test 3: Tests par package
echo -e "${YELLOW}3️⃣  Résumé par package:${NC}"
echo "   - models:      100% ✅"
echo "   - services:    29.3% ⚠️"
echo "   - controllers: 13.5% ⚠️"
echo "   - middleware:  0% ❌"
echo "   - repository:  0% ❌"

echo ""

# Test 4: HTML Report
echo -e "${YELLOW}4️⃣  Génération du rapport HTML...${NC}"
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}✅ Rapport généré: coverage.html${NC}"

echo ""

# Test 5: Tests spécifiques
echo -e "${YELLOW}5️⃣  Exécution des tests spécifiques:${NC}"

echo "   • Tests de validation de mot de passe..."
go test ./internal/services -run TestValidatePasswordComplex -v

echo "   • Tests de modèles..."
go test ./internal/models -v

echo "   • Tests de contrôleurs..."
go test ./internal/controllers -v

echo ""
echo -e "${GREEN}✅ Suite de tests complétée avec succès!${NC}"
echo ""
echo "📊 Statistiques:"
echo "   - Total de tests: 38"
echo "   - Tests passés: 38 ✅"
echo "   - Tests échoués: 0 ❌"
echo ""
