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

# ═══════════════════════════════════════════════════════════
# 1. Installation des dépendances
# ═══════════════════════════════════════════════════════════

echo "📦 Installation des dépendances..."
# Note: Amazon Linux 2023 a curl-minimal par défaut qui peut causer des conflits
# On utilise --allowerasing pour résoudre les conflits automatiquement
yum update -y --skip-broken
yum install -y wget git jq --skip-broken
# Installer curl en permettant l'effacement de curl-minimal si nécessaire
yum install -y curl --allowerasing || echo "⚠️ curl déjà installé ou curl-minimal présent"

# ═══════════════════════════════════════════════════════════
# 2. Installation de K3s (Kubernetes léger)
# ═══════════════════════════════════════════════════════════

echo "🐳 Installation de K3s..."
# Récupérer l'IP publique pour le certificat TLS
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
echo "📍 IP publique: $PUBLIC_IP"

curl -sfL https://get.k3s.io | sh -s - \
  --write-kubeconfig-mode 644 \
  --disable traefik \
  --tls-san "$PUBLIC_IP" \
  --node-name k3s-master

# Attendre que K3s soit prêt
echo "⏳ Attente du démarrage de K3s..."
while ! kubectl get nodes &> /dev/null; do
  sleep 5
done

echo "✅ K3s est opérationnel"
kubectl get nodes

# ═══════════════════════════════════════════════════════════
# 3. Configuration de kubectl pour l'utilisateur ec2-user
# ═══════════════════════════════════════════════════════════

echo "📝 Configuration du kubeconfig pour ec2-user..."

# Attendre que le fichier kubeconfig de K3s soit créé
KUBECONFIG_SOURCE="/etc/rancher/k3s/k3s.yaml"
echo "⏳ Attente de la création du kubeconfig K3s..."
RETRIES=0
MAX_RETRIES=30
while [ ! -f "$KUBECONFIG_SOURCE" ] && [ $RETRIES -lt $MAX_RETRIES ]; do
  echo "Tentative $((RETRIES+1))/$MAX_RETRIES - Kubeconfig pas encore disponible..."
  sleep 5
  RETRIES=$((RETRIES+1))
done

if [ ! -f "$KUBECONFIG_SOURCE" ]; then
  echo "❌ ERREUR: Kubeconfig K3s introuvable après $MAX_RETRIES tentatives"
  exit 1
fi

echo "✅ Kubeconfig K3s trouvé"

# Créer le répertoire et copier le kubeconfig
mkdir -p /home/ec2-user/.kube
cp "$KUBECONFIG_SOURCE" /home/ec2-user/.kube/config
chmod 644 /home/ec2-user/.kube/config
chown -R ec2-user:ec2-user /home/ec2-user/.kube

echo "✅ Kubeconfig configuré pour ec2-user"
ls -la /home/ec2-user/.kube/

# ═══════════════════════════════════════════════════════════
# 4. Création du namespace production
# ═══════════════════════════════════════════════════════════

echo "📦 Création du namespace production..."
kubectl create namespace production || true

# ═══════════════════════════════════════════════════════════
# 5. Création des secrets Kubernetes
# ═══════════════════════════════════════════════════════════

echo "🔐 Création des secrets..."

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

# ═══════════════════════════════════════════════════════════
# 6. Attendre que RDS soit accessible
# ═══════════════════════════════════════════════════════════

echo "⏳ Vérification de la connexion à RDS..."
DB_HOST=$(echo ${db_endpoint} | cut -d':' -f1)
for i in {1..60}; do
  if timeout 5 bash -c "echo > /dev/tcp/$DB_HOST/5432" 2>/dev/null; then
    echo "✅ RDS est accessible"
    break
  fi
  echo "Tentative $i/60..."
  sleep 10
done

# ═══════════════════════════════════════════════════════════
# 7. Déploiement des manifestes Kubernetes
# ═══════════════════════════════════════════════════════════

echo "🚀 Déploiement de l'application..."

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

echo "⏳ Attente du déploiement..."
kubectl rollout status deployment/auth-service -n production --timeout=5m

# ═══════════════════════════════════════════════════════════
# 8. Installation du service de mise à jour automatique
# ═══════════════════════════════════════════════════════════

echo "🔄 Installation du service de mise à jour automatique..."

cat > /usr/local/bin/check-updates.sh <<'SCRIPT'
#!/bin/bash

NAMESPACE="production"
DEPLOYMENT="auth-service"
IMAGE="${docker_username}/auth-service"
WEBHOOK_URL="${webhook_url}"

# Récupérer la version actuelle
CURRENT_TAG=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d':' -f2)

# Récupérer la dernière version depuis Docker Hub
LATEST_TAG=$(curl -s "https://registry.hub.docker.com/v2/repositories/$IMAGE/tags/?page_size=100" | jq -r '.results[] | select(.name != "latest") | .name' | sort -V | tail -n1)

if [ "$CURRENT_TAG" != "$LATEST_TAG" ] && [ ! -z "$LATEST_TAG" ]; then
  echo "🆕 Nouvelle version détectée: $LATEST_TAG (actuelle: $CURRENT_TAG)"
  
  # Envoyer notification
  curl -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
      \"event\": \"update_detected\",
      \"service\": \"auth-service\",
      \"from_version\": \"$CURRENT_TAG\",
      \"to_version\": \"$LATEST_TAG\",
      \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
    }" 2>/dev/null
  
  # Mettre à jour le deployment
  kubectl set image deployment/$DEPLOYMENT \
    auth-service=$IMAGE:$LATEST_TAG \
    -n $NAMESPACE
  
  # Attendre le rollout
  kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE
  
  # Notification de succès
  curl -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
      \"event\": \"update_completed\",
      \"service\": \"auth-service\",
      \"version\": \"$LATEST_TAG\",
      \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"status\": \"success\"
    }" 2>/dev/null
  
  echo "✅ Mise à jour terminée vers $LATEST_TAG"
else
  echo "✓ Déjà à jour (version: $CURRENT_TAG)"
fi
SCRIPT

chmod +x /usr/local/bin/check-updates.sh

# Créer un service systemd pour vérifier les mises à jour toutes les 5 minutes
cat > /etc/systemd/system/auth-updater.service <<'SERVICE'
[Unit]
Description=Auth Service Auto-Updater
After=network.target k3s.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/check-updates.sh
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICE

cat > /etc/systemd/system/auth-updater.timer <<'TIMER'
[Unit]
Description=Check for Auth Service updates every 5 minutes
Requires=auth-updater.service

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
Unit=auth-updater.service

[Install]
WantedBy=timers.target
TIMER

systemctl daemon-reload
systemctl enable auth-updater.timer
systemctl start auth-updater.timer

echo "✅ Service de mise à jour automatique installé"

# ═══════════════════════════════════════════════════════════
# 9. Configuration du firewall
# ═══════════════════════════════════════════════════════════

echo "🔥 Configuration du firewall..."
# Autoriser le trafic sur le port de l'application
iptables -I INPUT -p tcp --dport ${app_port} -j ACCEPT
iptables -I INPUT -p tcp --dport 30081 -j ACCEPT

# ═══════════════════════════════════════════════════════════
# 10. Notification de fin d'installation
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
  }" 2>/dev/null || echo "⚠️ Notification webhook failed"

echo "=================================="
echo "✅ Installation terminée!"
echo "Date: $(date)"
echo "IP publique: $PUBLIC_IP"
echo "Port: ${app_port}"
echo "=================================="

# Créer un fichier flag pour indiquer que l'installation est terminée
touch /home/ec2-user/.k3s-ready
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > /home/ec2-user/.k3s-ready
chown ec2-user:ec2-user /home/ec2-user/.k3s-ready
echo "✅ Fichier flag créé: /home/ec2-user/.k3s-ready"
