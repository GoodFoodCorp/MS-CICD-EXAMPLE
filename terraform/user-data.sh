#!/bin/bash
set -e

# ═══════════════════════════════════════════════════════════
# Script d'initialisation du serveur K3s
# ═══════════════════════════════════════════════════════════

LOG_FILE="/var/log/user-data.log"
exec > >(tee -a $LOG_FILE) 2>&1

echo "=================================="
echo "Début de l'initialisation K3s"
echo "Date: $(date)"
echo "=================================="

# ---------------------------------------------------------
# 1. Installation des dépendances
# ---------------------------------------------------------

echo "[INSTALL] Installing system dependencies..."
yum update -y --skip-broken
yum install -y wget git jq --skip-broken
yum install -y curl --allowerasing || echo "[WARNING] curl déjà installé ou curl-minimal présent"

# ---------------------------------------------------------
# 2. Installation de K3s (Kubernetes léger)
# ---------------------------------------------------------

echo "[INSTALL] Installing K3s..."
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
echo "[INFO] IP publique: $PUBLIC_IP"

curl -sfL https://get.k3s.io | sh -s - \
  --write-kubeconfig-mode 644 \
  --disable traefik \
  --tls-san "$PUBLIC_IP" \
  --node-name k3s-master

echo "[WAIT] Attente du démarrage de K3s..."
while ! kubectl get nodes &> /dev/null; do
  sleep 5
done

echo "[OK] K3s est opérationnel"
kubectl get nodes

# ═══════════════════════════════════════════════════════════
# 3.[CONFIG] de kubectl pour l'utilisateur ec2-user
# ═══════════════════════════════════════════════════════════

echo "[CONFIG][CONFIG] du kubeconfig pour ec2-user..."

KUBECONFIG_SOURCE="/etc/rancher/k3s/k3s.yaml"
echo "[WAIT] Attente de la création du kubeconfig K3s..."
RETRIES=0
MAX_RETRIES=30
while [ ! -f "$KUBECONFIG_SOURCE" ] && [ $RETRIES -lt $MAX_RETRIES ]; do
  echo "Tentative $((RETRIES+1))/$MAX_RETRIES - Kubeconfig pas encore disponible..."
  sleep 5
  RETRIES=$((RETRIES+1))
done

if [ ! -f "$KUBECONFIG_SOURCE" ]; then
  echo "[ERROR] ERREUR: Kubeconfig K3s introuvable après $MAX_RETRIES tentatives"
  exit 1
fi

echo "[OK] Kubeconfig K3s trouvé"

# Créer le répertoire et copier le kubeconfig
mkdir -p /home/ec2-user/.kube
cp "$KUBECONFIG_SOURCE" /home/ec2-user/.kube/config
chmod 644 /home/ec2-user/.kube/config
chown -R ec2-user:ec2-user /home/ec2-user/.kube

echo "[OK] Kubeconfig configuré pour ec2-user"
ls -la /home/ec2-user/.kube/

# ═══════════════════════════════════════════════════════════
# La CI peut maintenant récupérer le kubeconfig et continuer
# Le reste se termine en arrière-plan
# ═══════════════════════════════════════════════════════════

touch /home/ec2-user/.k3s-ready
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > /home/ec2-user/.k3s-ready
chown ec2-user:ec2-user /home/ec2-user/.k3s-ready
echo "[OK] Flag .k3s-ready créé - CI peut continuer"

# ═══════════════════════════════════════════════════════════
# 4.[CREATE] du namespace production
# ═══════════════════════════════════════════════════════════

echo "[CREATE] du namespace production..."
kubectl create namespace production || true

# ═══════════════════════════════════════════════════════════
# 5.[CREATE] des secrets Kubernetes
# ═══════════════════════════════════════════════════════════

echo "[CREATE] des secrets..."

# Secret pour la base de données
kubectl create secret generic db-credentials \
  --from-literal=host="${db_endpoint}" \
  --from-literal=database="${db_name}" \
  --from-literal=username="${db_username}" \
  --from-literal=password="${db_password}" \
  --namespace=production \
  --dry-run=client -o yaml | kubectl apply -f -

# Secret pour le webhook
kubectl create secret generic webhook-config \
  --from-literal=url="${webhook_url}" \
  --namespace=production \
  --dry-run=client -o yaml | kubectl apply -f -

# Secret pour DATABASE_URL
# Utilisation de Python pour l'URL encoding car il est disponible sur Amazon Linux 2023
DB_PASSWORD_ENCODED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${db_password}', safe=''))")
DATABASE_URL="postgresql://${db_username}:$${DB_PASSWORD_ENCODED}@${db_endpoint}/${db_name}?sslmode=require"
kubectl create secret generic database-url \
  --from-literal=url="$DATABASE_URL" \
  --namespace=production \
  --dry-run=client -o yaml | kubectl apply -f -

echo "[OK] Secrets créés avec succès"

# ═══════════════════════════════════════════════════════════
# 6. Attendre que RDS soit accessible
# ═══════════════════════════════════════════════════════════

echo "[WAIT] Vérification de la connexion à RDS..."
DB_HOST=$(echo ${db_endpoint} | cut -d':' -f1)
for i in {1..60}; do
  if timeout 5 bash -c "echo > /dev/tcp/$DB_HOST/5432" 2>/dev/null; then
    echo "[OK] RDS est accessible"
    break
  fi
  echo "Tentative $i/60..."
  sleep 10
done

# ═══════════════════════════════════════════════════════════
# 7.[DEPLOY] des manifestes Kubernetes
# ═══════════════════════════════════════════════════════════

echo "[DEPLOY][DEPLOY] de l'application..."

# Créer les manifestes K8s
cat > /tmp/auth-deployment.yaml <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: auth-service
  namespace: production
  labels:
    app: auth-service
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: auth-service
  template:
    metadata:
      labels:
        app: auth-service
    spec:
      containers:
      - name: auth-service
        image: ${docker_username}/auth-service:latest
        ports:
        - containerPort: ${app_port}
          name: http
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: host
        - name: DB_NAME
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: database
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: password
        - name: APP_PORT
          value: "${app_port}"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: ${app_port}
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: ${app_port}
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: auth-service
  namespace: production
spec:
  type: NodePort
  selector:
    app: auth-service
  ports:
  - port: ${app_port}
    targetPort: ${app_port}
    nodePort: 30081
    protocol: TCP
    name: http
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: auth-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: auth-service
  minReplicas: 2
  maxReplicas: 5
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
EOF

# Appliquer les manifestes
kubectl apply -f /tmp/auth-deployment.yaml

echo "[WAIT] Attente du déploiement..."
kubectl rollout status deployment/auth-service -n production --timeout=5m

echo "[INFO] Déploiement initial terminé. Les mises à jour sont gérées par CI/CD."

# ═══════════════════════════════════════════════════════════
# 8.[CONFIG] du firewall
# ═══════════════════════════════════════════════════════════

echo "[CONFIG] du firewall..."
# Autoriser le trafic sur le port de l'application
iptables -I INPUT -p tcp --dport ${app_port} -j ACCEPT
iptables -I INPUT -p tcp --dport 30081 -j ACCEPT

# ═══════════════════════════════════════════════════════════
# 9.[NOTIFY] de fin d'installation
# ═══════════════════════════════════════════════════════════

PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

curl -X POST "${webhook_url}" \
  -H "Content-Type: application/json" \
  -d "{
    \"event\": \"server_initialized\",
    \"service\": \"auth-service\",
    \"public_ip\": \"$PUBLIC_IP\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"status\": \"ready\"
  }" 2>/dev/null || echo "[WARNING][NOTIFY] webhook failed"

echo "=================================="
echo "[OK][INSTALL] terminée!"
echo "Date: $(date)"
echo "IP publique: $PUBLIC_IP"
echo "Port: ${app_port}"
echo "=================================="