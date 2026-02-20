# 🏗️ Architecture - Auth Service CI/CD

## 📋 Vue d'ensemble

Ce document détaille l'architecture complète du système CI/CD pour le service d'authentification.

## 🎯 Objectifs architecturaux

- ✅ **Haute disponibilité** : Minimum 2 replicas, auto-scaling jusqu'à 5
- ✅ **Zero downtime** : Rolling updates avec health checks
- ✅ **Sécurité** : Scanning automatique, secrets managés, RDS privé
- ✅ **Automatisation** : Pipeline complet de la PR au déploiement
- ✅ **Observabilité** : Logs, monitoring, notifications
- ✅ **Scalabilité** : HPA basé sur CPU/Memory
- ✅ **Infrastructure as Code** : Terraform pour reproductibilité

---

## 🏗️ Architecture Infrastructure (AWS)

### Schéma réseau

```
┌─────────────────────────────────────────────────────────────────────┐
│                              Internet                                │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   Internet Gateway      │
                    └────────────┬────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────┐
│                         VPC (10.0.0.0/16)                             │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                   Public Subnet (10.0.1.0/24)                 │   │
│  │                       AZ: eu-west-3a                          │   │
│  │                                                                │   │
│  │  ┌─────────────────────────────────────────────────────┐     │   │
│  │  │              EC2 Instance (t3.medium)               │     │   │
│  │  │               - 2 vCPU, 4GB RAM                     │     │   │
│  │  │               - K3s Server                          │     │   │
│  │  │               - Auto-updater service                │     │   │
│  │  │                                                      │     │   │
│  │  │  ┌────────────────────────────────────────────┐    │     │   │
│  │  │  │         Kubernetes (K3s)                   │    │     │   │
│  │  │  │                                             │    │     │   │
│  │  │  │  Namespace: production                     │    │     │   │
│  │  │  │  ┌───────────────────────────────────┐    │    │     │   │
│  │  │  │  │  Deployment: auth-service         │    │    │     │   │
│  │  │  │  │  - Replicas: 2 (min) - 5 (max)    │    │    │     │   │
│  │  │  │  │  - Image: auth-service:latest     │    │    │     │   │
│  │  │  │  │  - Port: 8081                      │    │    │     │   │
│  │  │  │  │                                     │    │    │     │   │
│  │  │  │  │  Pod 1 (Running)                   │    │    │     │   │
│  │  │  │  │  ├─ Container: auth-service        │    │    │     │   │
│  │  │  │  │  ├─ CPU: 250m / Memory: 256Mi      │    │    │     │   │
│  │  │  │  │  └─ Health: ✓                      │    │    │     │   │
│  │  │  │  │                                     │    │    │     │   │
│  │  │  │  │  Pod 2 (Running)                   │    │    │     │   │
│  │  │  │  │  ├─ Container: auth-service        │    │    │     │   │
│  │  │  │  │  ├─ CPU: 250m / Memory: 256Mi      │    │    │     │   │
│  │  │  │  │  └─ Health: ✓                      │    │    │     │   │
│  │  │  │  └───────────────────────────────────┘    │    │     │   │
│  │  │  │                                             │    │     │   │
│  │  │  │  ┌───────────────────────────────────┐    │    │     │   │
│  │  │  │  │  Service: auth-service            │    │    │     │   │
│  │  │  │  │  - Type: NodePort                  │    │    │     │   │
│  │  │  │  │  - Port: 8081 → 30081              │    │    │     │   │
│  │  │  │  └───────────────────────────────────┘    │    │     │   │
│  │  │  │                                             │    │     │   │
│  │  │  │  ┌───────────────────────────────────┐    │    │     │   │
│  │  │  │  │  HPA: auth-service-hpa            │    │    │     │   │
│  │  │  │  │  - Min: 2, Max: 5                  │    │    │     │   │
│  │  │  │  │  - CPU Target: 70%                 │    │    │     │   │
│  │  │  │  │  - Memory Target: 80%              │    │    │     │   │
│  │  │  │  └───────────────────────────────────┘    │    │     │   │
│  │  │  └────────────────────────────────────────────┘    │     │   │
│  │  │                                                      │     │   │
│  │  │  Elastic IP: XXX.XXX.XXX.XXX                        │     │   │
│  │  │  Security Group: k3s-sg (22, 80, 443, 6443, 8081)  │     │   │
│  │  └─────────────────────────────────────────────────────┘     │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │              Private Subnet 1 (10.0.10.0/24)                  │   │
│  │                       AZ: eu-west-3a                          │   │
│  │  ┌─────────────────────────────────────────────────────┐     │   │
│  │  │     RDS PostgreSQL (Primary)                        │     │   │
│  │  │     - Instance: db.t3.micro                         │     │   │
│  │  │     - Engine: PostgreSQL 15.5                       │     │   │
│  │  │     - Storage: 20GB (gp3)                           │     │   │
│  │  │     - Security Group: rds-sg (5432 ← k3s-sg)       │     │   │
│  │  └─────────────────────────────────────────────────────┘     │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │              Private Subnet 2 (10.0.11.0/24)                  │   │
│  │                       AZ: eu-west-3b                          │   │
│  │  ┌─────────────────────────────────────────────────────┐     │   │
│  │  │     RDS PostgreSQL (Standby)                        │     │   │
│  │  │     - Multi-AZ for HA                               │     │   │
│  │  └─────────────────────────────────────────────────────┘     │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Composants Infrastructure

| Composant            | Type                 | Configuration   | Rôle               |
| -------------------- | -------------------- | --------------- | ------------------ |
| **VPC**              | aws_vpc              | 10.0.0.0/16     | Réseau isolé       |
| **Public Subnet**    | aws_subnet           | 10.0.1.0/24     | Pour EC2 K3s       |
| **Private Subnets**  | aws_subnet           | 10.0.10-11.0/24 | Pour RDS Multi-AZ  |
| **Internet Gateway** | aws_internet_gateway | -               | Accès Internet     |
| **Route Table**      | aws_route_table      | -               | Routage public     |
| **EC2 K3s**          | aws_instance         | t3.medium       | Serveur Kubernetes |
| **RDS PostgreSQL**   | aws_db_instance      | db.t3.micro     | Base de données    |
| **Elastic IP**       | aws_eip              | -               | IP publique fixe   |
| **Security Groups**  | aws_security_group   | k3s-sg, rds-sg  | Firewall           |

---

## 🔄 Architecture CI/CD Pipeline

### Flux complet

```
┌─────────────────────────────────────────────────────────────────────┐
│                           DÉVELOPPEUR                                │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             │ git push origin main
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         GITHUB REPOSITORY                            │
│  - Code source                                                       │
│  - GitHub Actions workflows                                          │
│  - Secrets (AWS, Docker, etc.)                                       │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             │ Trigger on push
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       GITHUB ACTIONS PIPELINE                        │
│                                                                       │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  JOB 1: BUILD & TEST (2-3 min)                                │ │
│  │  ─────────────────────────────                                │ │
│  │  1. Checkout code                                             │ │
│  │  2. Setup Go 1.23                                             │ │
│  │  3. go mod download                                           │ │
│  │  4. go build ./cmd/main.go                                    │ │
│  │  5. go test ./... -coverage                                   │ │
│  │  6. Upload coverage artifact                                  │ │
│  │                                                                │ │
│  │  Outputs: ✅ Binary, ✅ Coverage report                       │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                             │                                         │
│                             ▼                                         │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  JOB 2: SECURITY SCAN (2-3 min)                               │ │
│  │  ──────────────────────────────                               │ │
│  │  1. govulncheck (vulnerabilities in Go dependencies)          │ │
│  │  2. gosec (static code analysis)                              │ │
│  │  3. nancy (OSS Index check)                                   │ │
│  │  4. Upload SARIF to GitHub Security                           │ │
│  │                                                                │ │
│  │  Outputs: ✅ Security report, ❌ Block if critical            │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                             │                                         │
│                             ▼                                         │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  JOB 3: VERSIONING (30s)                                      │ │
│  │  ────────────────────────                                     │ │
│  │  1. Get latest git tag                                        │ │
│  │  2. Parse SemVer (vX.Y.Z)                                     │ │
│  │  3. Increment version:                                        │ │
│  │     - [patch] → X.Y.Z+1                                       │ │
│  │     - [minor] → X.Y+1.0                                       │ │
│  │     - [major] → X+1.0.0                                       │ │
│  │  4. Create git tag                                            │ │
│  │  5. Create GitHub Release                                     │ │
│  │                                                                │ │
│  │  Outputs: vX.Y.Z (e.g., v1.2.3)                               │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                             │                                         │
│                             ▼                                         │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  JOB 4: DOCKER BUILD & PUSH (3-5 min)                         │ │
│  │  ─────────────────────────────────                            │ │
│  │  1. Setup Docker Buildx (multi-arch)                          │ │
│  │  2. Login to Docker Hub                                       │ │
│  │  3. Build multi-stage image:                                  │ │
│  │     Stage 1: golang:1.25-alpine → build                       │ │
│  │     Stage 2: distroless → runtime                             │ │
│  │  4. Tag:                                                       │ │
│  │     - username/auth-service:vX.Y.Z                            │ │
│  │     - username/auth-service:latest                            │ │
│  │     - username/auth-service:sha-xxxxx                         │ │
│  │  5. Push to Docker Hub                                        │ │
│  │  6. Cache layers for faster rebuilds                          │ │
│  │                                                                │ │
│  │  Outputs: Docker images on Docker Hub                         │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                             │                                         │
│                             ▼                                         │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  JOB 5: DEPLOY TO PRODUCTION (2-3 min)                        │ │
│  │  ──────────────────────────────────                           │ │
│  │  1. Configure AWS credentials                                 │ │
│  │  2. Setup kubectl with KUBE_CONFIG secret                     │ │
│  │  3. kubectl set image deployment/auth-service \               │ │
│  │        auth-service=username/auth-service:vX.Y.Z              │ │
│  │        -n production                                           │ │
│  │  4. kubectl rollout status (wait for ready)                   │ │
│  │  5. Send webhook notification (success/failure)               │ │
│  │                                                                │ │
│  │  Outputs: Application deployed ✅                             │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                                                                       │
└────────────────────────────┬──────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         AWS PRODUCTION                               │
│                                                                       │
│  EC2 K3s Server receives kubectl command                            │
│  ↓                                                                    │
│  Kubernetes pulls new image from Docker Hub                          │
│  ↓                                                                    │
│  Rolling Update:                                                      │
│    - Create new pod with vX.Y.Z                                      │
│    - Wait for health check (/health) → 200 OK                        │
│    - Terminate old pod                                                │
│    - Repeat for replica 2                                             │
│  ↓                                                                    │
│  All pods running ✅                                                 │
│  ↓                                                                    │
│  Auto-updater service detects new version every 5 min               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Système d'auto-update

### Service systemd

Le serveur K3s exécute un service systemd qui vérifie toutes les 5 minutes s'il y a une nouvelle version.

```
┌──────────────────────────────────────────────────────┐
│         auth-updater.timer (systemd)                 │
│         Runs every 5 minutes                         │
└─────────────────────┬────────────────────────────────┘
                      │
                      ▼
┌──────────────────────────────────────────────────────┐
│         auth-updater.service                         │
│         Executes: /usr/local/bin/check-updates.sh    │
└─────────────────────┬────────────────────────────────┘
                      │
                      ▼
┌──────────────────────────────────────────────────────┐
│  Script: check-updates.sh                            │
│                                                       │
│  1. Get current version from K8s:                    │
│     kubectl get deployment auth-service              │
│        -o jsonpath='{.spec.template.spec             │
│         .containers[0].image}'                       │
│                                                       │
│  2. Get latest version from Docker Hub API:          │
│     curl https://registry.hub.docker.com/            │
│       v2/repositories/USERNAME/auth-service/tags/    │
│                                                       │
│  3. Compare versions                                 │
│                                                       │
│  4. If different:                                     │
│     ├─ Send webhook: "new_version_detected"          │
│     ├─ kubectl set image deployment/auth-service ... │
│     ├─ kubectl rollout status ...                    │
│     └─ Send webhook: "update_completed"              │
│                                                       │
│  5. If error:                                         │
│     └─ Send webhook: "update_failed"                 │
└──────────────────────────────────────────────────────┘
```

---

## 🔐 Gestion des secrets

### Flux des secrets

```
┌─────────────────────────────────────────────────────────┐
│                    DÉVELOPPEUR                          │
│  - Configure GitHub Secrets                             │
│  - Configure terraform.tfvars (local, not committed)    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ├──────────────┐
                     │              │
                     ▼              ▼
┌──────────────────────────┐  ┌────────────────────────────┐
│   GITHUB SECRETS         │  │   TERRAFORM VARIABLES      │
│   ─────────────────      │  │   ───────────────────      │
│                          │  │                            │
│   • AWS_ACCESS_KEY_ID    │  │   • db_password            │
│   • AWS_SECRET_ACCESS_   │  │   • docker_username        │
│   • DOCKER_USERNAME      │  │   • webhook_url            │
│   • DOCKER_PASSWORD      │  │                            │
│   • KUBE_CONFIG          │  │   ↓                        │
│   • WEBHOOK_URL          │  │   Passed to user-data.sh   │
│                          │  │   ↓                        │
│   ↓                      │  │   Creates K8s secrets      │
│   Used in GitHub Actions │  │                            │
└──────────────────────────┘  └────────────────────────────┘
                     │                      │
                     │                      ▼
                     │         ┌────────────────────────────┐
                     │         │   KUBERNETES SECRETS       │
                     │         │   ──────────────────       │
                     │         │                            │
                     │         │   Secret: db-credentials   │
                     │         │   ├─ host                  │
                     │         │   ├─ database              │
                     │         │   ├─ username              │
                     │         │   └─ password              │
                     │         │                            │
                     │         │   Secret: webhook-config   │
                     │         │   └─ url                   │
                     │         │                            │
                     │         │   ↓                        │
                     │         │   Mounted as env vars in   │
                     │         │   Application Pods         │
                     │         └────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│              APPLICATION RUNTIME                         │
│                                                           │
│  Pod reads secrets from environment variables:           │
│  - DB_HOST, DB_NAME, DB_USER, DB_PASSWORD                │
│  - WEBHOOK_URL                                            │
│                                                           │
│  ⚠️ Never logged, never exposed in API responses         │
└──────────────────────────────────────────────────────────┘
```

---

## 📊 Monitoring & Observabilité

### Points de monitoring

1. **Application Level**
   - Health check endpoint: `GET /health`
   - Returns: `{"status": "OK", "uptime": "Running"}`
   - Vérifie la connexion DB avec `sqlDB.Ping()`

2. **Kubernetes Level**
   - Liveness Probe: `/health` every 10s
   - Readiness Probe: `/health` every 5s
   - HPA metrics: CPU & Memory utilization

3. **Infrastructure Level**
   - CloudWatch: EC2 metrics (CPU, Memory, Disk, Network)
   - RDS metrics: Connections, CPU, Storage

4. **Pipeline Level**
   - GitHub Actions logs
   - Webhook notifications pour chaque événement

### Logging

```
Application Logs
    ↓
Stdout/Stderr
    ↓
Kubernetes captures logs
    ↓
kubectl logs <pod>
    ↓
(Optionnel) Forward to:
  - CloudWatch Logs
  - ELK Stack
  - Grafana Loki
```

---

## 🔄 Rolling Update Strategy

Kubernetes effectue les mises à jour sans interruption :

```
État initial:
┌─────────┐ ┌─────────┐
│ Pod v1  │ │ Pod v1  │  ← 2 replicas en v1
└─────────┘ └─────────┘
     ↑           ↑
     └───────┬───┘
         Service (Load Balancer)

Update déclenché:
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Pod v1  │ │ Pod v1  │ │ Pod v2  │  ← Création nouveau pod v2
└─────────┘ └─────────┘ └─────────┘
     ↑           ↑           ↓
     └───────┬───┘     Health check
         Service        en cours...

Health check OK:
┌─────────┐ ┌─────────┐
│ Pod v1  │ │ Pod v2  │  ← Pod v2 prêt, pod v1 #1 supprimé
└─────────┘ └─────────┘
     ↑           ↑
     └───────┬───┘
         Service

Création 2e pod v2:
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Pod v1  │ │ Pod v2  │ │ Pod v2  │  ← 2e pod v2 en création
└─────────┘ └─────────┘ └─────────┘
     ↑           ↑           ↓
     └───────┬───┘     Health check
         Service        en cours...

État final:
┌─────────┐ ┌─────────┐
│ Pod v2  │ │ Pod v2  │  ← 2 replicas en v2
└─────────┘ └─────────┘
     ↑           ↑
     └───────┬───┘
         Service (Load Balancer)

✅ Zero downtime! Le service reste accessible pendant tout le processus
```

Configuration dans le Deployment:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1 # Maximum 1 pod de plus pendant la mise à jour
    maxUnavailable: 0 # Toujours au moins 2 pods disponibles
```

---

## 🚨 Gestion des erreurs

### Niveaux de fallback

1. **Health Check Failure**

   ```
   Health Check Failed
      ↓
   Kubernetes marks pod as not ready
      ↓
   Traffic stopped to this pod
      ↓
   Kubernetes tries to restart container
      ↓
   After 3 failures: Crash Loop Backoff
   ```

2. **Deployment Failure**

   ```
   New version deployed
      ↓
   Health checks failing
      ↓
   Pods not becoming ready
      ↓
   GitHub Actions detects timeout
      ↓
   Prompt for rollback
   ```

3. **Auto-Update Failure**
   ```
   New version detected
      ↓
   Update command fails
      ↓
   Send webhook notification (error)
      ↓
   Keep current version running
      ↓
   Retry on next check (5 min)
   ```

---

## 📈 Scalabilité

### Horizontal Pod Autoscaler (HPA)

```
Métriques collectées:
  - CPU: 70% threshold
  - Memory: 80% threshold

Current State:
  2 pods @ 40% CPU → OK, no scaling

Load increases:
  2 pods @ 75% CPU → Triggers scale up
  ↓
  Create pod #3
  ↓
  3 pods @ 50% CPU → Balanced

Extreme load:
  3 pods @ 80% CPU → Scale up again
  ↓
  Create pod #4, #5
  ↓
  5 pods @ 45% CPU → Max reached

Load decreases:
  5 pods @ 30% CPU for 5 minutes → Scale down
  ↓
  Remove pod #5
  ↓
  4 pods @ 35% CPU
  ↓
  Continue scaling down gradually to min 2 pods
```

Configuration:

```yaml
minReplicas: 2
maxReplicas: 5
targetCPUUtilizationPercentage: 70
targetMemoryUtilizationPercentage: 80
```

---

## 🔐 Sécurité

### Couches de sécurité

```
Layer 1: Network (AWS Security Groups)
  ├─ Public: SSH (22), HTTP (80), HTTPS (443), K8s API (6443), App (8081)
  └─ Private: PostgreSQL (5432) ← Only from K3s SG

Layer 2: Infrastructure
  ├─ RDS in private subnet (no public access)
  ├─ EC2 with Elastic IP (DDoS protected)
  └─ VPC isolation

Layer 3: Kubernetes
  ├─ Namespaces isolation
  ├─ Secrets for sensitive data
  ├─ Resource limits (CPU, Memory)
  └─ Non-root user in containers

Layer 4: Application
  ├─ JWT for authentication
  ├─ Rate limiting
  ├─ Input validation
  └─ CORS configuration

Layer 5: CI/CD
  ├─ Security scanning (gosec, govulncheck, nancy)
  ├─ Vulnerability detection
  ├─ SARIF reports to GitHub Security
  └─ Block on critical vulnerabilities
```

---

## 💰 Estimation des coûts

### Coûts mensuels AWS (région eu-west-3)

| Service             | Configuration       | Usage    | Coût mensuel     |
| ------------------- | ------------------- | -------- | ---------------- |
| **EC2 t3.medium**   | 2 vCPU, 4GB RAM     | 24/7     | $30.37           |
| **EBS gp3**         | 30GB                | 30GB     | $2.40            |
| **RDS db.t3.micro** | 1 vCPU, 1GB RAM     | 24/7     | $15.33           |
| **RDS Storage**     | 20GB gp3            | 20GB     | $2.30            |
| **Elastic IP**      | 1 IP                | Attached | $3.60            |
| **Data Transfer**   | Out to Internet     | 50GB     | $4.50            |
| **VPC**             | 1 VPC, subnets, IGW | Free     | $0.00            |
| **CloudWatch**      | Basic monitoring    | -        | $0.00            |
| **Backup**          | RDS automated       | 7 days   | $0.00            |
| **Total**           |                     |          | **~$58.50/mois** |

### Coûts variables

- **Data Transfer** : $0.09/GB supplémentaire
- **Scaling** : Si scale up to 5 pods, pas de coût supplémentaire (même EC2)
- **Snapshots** : Si plus de 7 jours de backup
- **Load Balancer** : $16.20/mois si ajouté (non inclus actuellement)

### Optimisations possibles

- **Reserved Instances** : -40% sur EC2 et RDS (engagement 1 an)
- **Spot Instances** : -70% mais moins de uptime garantie
- **t4g instances** : ARM64, -20% de coût
- **Auto-stop** : Arrêter la nuit/weekend pour environnements de test

---

## 🎯 Améliorations futures

### Court terme

- [ ] Ajouter un Load Balancer (ALB) pour SSL/TLS
- [ ] Intégrer Prometheus + Grafana pour monitoring avancé
- [ ] Ajouter des tests d'intégration dans le pipeline
- [ ] Implémenter Blue/Green deployment

### Moyen terme

- [ ] Multi-région pour disaster recovery
- [ ] EKS au lieu de K3s pour production scale
- [ ] ElastiCache Redis pour sessions
- [ ] S3 pour stockage de fichiers statiques

### Long terme

- [ ] Service Mesh (Istio) pour observabilité avancée
- [ ] GitOps avec ArgoCD
- [ ] Infrastructure multi-cloud
- [ ] Chaos Engineering (tests de résilience)

---

**Architecture conçue pour : Haute disponibilité • Scalabilité • Sécurité • Observabilité**
