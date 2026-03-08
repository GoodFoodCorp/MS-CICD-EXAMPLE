#!/bin/bash
# Surveillance continue du service deploye
# Usage: ./monitor.sh [ip] (defaut: 35.153.116.70)

set -euo pipefail

SERVER="${1:-35.153.116.70}"
PORT="30081"
BASE_URL="http://${SERVER}:${PORT}"

echo "=== SURVEILLANCE CONTINUE - ${SERVER}:${PORT} ==="
echo "Demarrage: $(date)"
echo ""

ITERATION=0
PREV_VERSION=""

while true; do
    ITERATION=$((ITERATION + 1))
    TS=$(date +%H:%M:%S)

    VERSION=$(curl -m 3 -s "${BASE_URL}/version" 2>/dev/null | grep -o '"version":"[^"]*"' | cut -d'"' -f4 || echo "")
    HEALTH=$(curl -m 3 -s "${BASE_URL}/health" 2>/dev/null | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "")
    DBHEALTH=$(curl -m 3 -s "${BASE_URL}/health/db" 2>/dev/null | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "")

    VERSION="${VERSION:-TIMEOUT}"
    HEALTH="${HEALTH:-DOWN}"
    DBHEALTH="${DBHEALTH:-UNREACHABLE}"

    # Detecter un changement de version
    if [ -n "${PREV_VERSION}" ] && [ "${VERSION}" != "${PREV_VERSION}" ]; then
        echo ""
        echo "!!! CHANGEMENT DE VERSION: ${PREV_VERSION} -> ${VERSION} !!!"
        echo ""
    fi
    PREV_VERSION="${VERSION}"

    echo "[${TS}] #${ITERATION} | version=${VERSION} | health=${HEALTH} | db=${DBHEALTH}"

    # Tests complets toutes les 6 iterations
    if [ $(( ITERATION % 6 )) -eq 1 ]; then
        echo "  --> Tests endpoints:"

        R=$(curl -m 3 -s -X POST "${BASE_URL}/api/auth/login" \
            -H "Content-Type: application/json" \
            -d '{"email":"nobody@test.com","password":"bad"}' 2>/dev/null || echo "ERR")
        echo "      login(bad creds):  ${R}"

        R=$(curl -m 3 -s -X POST "${BASE_URL}/api/auth/register" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"bot${ITERATION}@test.com\",\"password\":\"Monitor1234!\",\"name\":\"Bot\"}" 2>/dev/null || echo "ERR")
        echo "      register:          ${R}"

        R=$(curl -m 3 -s "${BASE_URL}/api/admin/roles" 2>/dev/null || echo "ERR")
        echo "      GET /admin/roles:  ${R}"

        R=$(curl -m 3 -s "${BASE_URL}/api/admin/users" 2>/dev/null || echo "ERR")
        echo "      GET /admin/users:  ${R}"

        R=$(curl -m 3 -s "${BASE_URL}/api/user/me" 2>/dev/null || echo "ERR")
        echo "      GET /user/me:      ${R}"

        SWHOST=$(curl -m 3 -s "${BASE_URL}/api-docs/swagger.json" 2>/dev/null | \
            python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('host','N/A'))" 2>/dev/null || echo "ERR")
        echo "      swagger host:      ${SWHOST}"
        echo ""
    fi

    sleep 15
done
