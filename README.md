# 🔐 Auth Service - Microservice d'authentification avec CI/CD

> Microservice d'authentification moderne en Go avec pipeline CI/CD complet, déploiement automatisé sur AWS et Kubernetes

[![CI/CD Pipeline](https://img.shields.io/badge/CI%2FCD-GitHub%20Actions-2088FF?logo=github-actions)](https://github.com/features/actions)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://golang.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-K3s-326CE5?logo=kubernetes)](https://k3s.io/)
[![Terraform](https://img.shields.io/badge/Terraform-IaC-7B42BC?logo=terraform)](https://www.terraform.io/)
[![AWS](https://img.shields.io/badge/AWS-Cloud-FF9900?logo=amazon-aws)](https://aws.amazon.com/)

---

## 📋 Table des matières

- [✨ Fonctionnalités](#-fonctionnalités)
- [🚀 Démarrage rapide](#-démarrage-rapide)
- [🏗️ Architecture](#️-architecture)
- [🔄 Pipeline CI/CD](#-pipeline-cicd)
- [📦 Structure du projet](#-structure-du-projet)
- [🛠️ Développement local](#️-développement-local)
- [🔐 Credentials par défaut](#-credentials-par-défaut)
- [🌐 API Documentation](#-api-documentation)
- [⚙️ Configuration](#️-configuration)

---

## ✨ Fonctionnalités

### Backend
- 🔒 **Authentification JWT** avec refresh tokens
- 👥 **Multi-tenant** avec gestion de rôles (admin, manager, user)
- 📧 **Vérification email** et reset password
- 🔐 **Sécurité renforcée** (bcrypt, rate limiting)
- 🗄️ **PostgreSQL** avec migrations automatiques
- 📝 **Seeding automatique** des données de test

### DevOps
- 🚀 **CI/CD complet** avec GitHub Actions
- 🏗️ **Infrastructure as Code** avec Terraform
- ☸️ **Kubernetes (K3s)** avec auto-scaling (2-5 replicas)
- 🔄 **Zero downtime** avec rolling updates
- 🐳 **Docker** multi-stage optimisé (distroless)
- 📊 **Health checks** et HPA (CPU/Memory)
- 🔔 **Notifications webhook** de déploiement
- 🔐 **Security scanning** automatique

---

## 🚀 Démarrage rapide

### Prérequis
- Go 1.24+
- Docker & Docker Compose
- Terraform 1.6+
- AWS CLI configuré
- kubectl

### Installation locale

```bash
# 1. Cloner le repo
git clone <your-repo-url>
cd auth-service

# 2. Installer les dépendances
make deps

# 3. Configurer l'environnement
cp .env.example .env
# Éditer .env avec vos paramètres

# 4. Lancer avec Docker Compose
docker-compose up -d

# 5. Tester l'API
curl http://localhost:8081/health
```

### Déploiement sur AWS

```bash
# 1. Configurer AWS credentials
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_SESSION_TOKEN="your-token"  # Si nécessaire
export AWS_REGION="us-east-1"

# 2. Déployer l'infrastructure
cd terraform
terraform init
terraform apply -auto-approve

# 3. Configurer les GitHub Secrets (voir section Configuration)

# 4. Push sur main → Déploiement automatique !
git push origin main
```

**🎯 Résultat** : API accessible sur `http://<votre-ip>:30081/scalar`

---

## 🏗️ Architecture

### Vue d'ensemble

```
┌─────────────────────────────────────────────────────────────────┐
│                         GITHUB ACTIONS                           │
│  Build → Test → Security → Docker → Terraform → Deploy          │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                          AWS CLOUD                               │
│                                                                   │
│  ┌──────────────────┐        ┌──────────────────┐              │
│  │   VPC 10.0.0.0/16│        │  Internet Gateway│              │
│  └────────┬─────────┘        └────────┬─────────┘              │
│           │                            │                         │
│  ┌────────▼────────────────────────────▼─────────────────┐     │
│  │         Public Subnet 10.0.1.0/24                      │     │
│  │                                                         │     │
│  │  ┌────────────────────────────────────────────┐       │     │
│  │  │  EC2 t3.medium (K3s Server)                │       │     │
│  │  │  ┌──────────────────────────────────────┐  │       │     │
│  │  │  │  Kubernetes K3s                      │  │       │     │
│  │  │  │  ├─ Namespace: production            │  │       │     │
│  │  │  │  ├─ Deployment: auth-service         │  │       │     │
│  │  │  │  │  ├─ Replicas: 2-5 (HPA)          │  │       │     │
│  │  │  │  │  └─ Image: cocostee/auth:0.0.X   │  │       │     │
│  │  │  │  ├─ Service: NodePort 30081          │  │       │     │
│  │  │  │  └─ HPA: CPU 70% / Memory 80%        │  │       │     │
│  │  │  └──────────────────────────────────────┘  │       │     │
│  │  │  📡 Elastic IP: XXX.XXX.XXX.XXX            │       │     │
│  │  └────────────────────────────────────────────┘       │     │
│  └─────────────────────────────────────────────────────────┘     │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │    Private Subnets (10.0.10.0/24, 10.0.11.0/24)        │    │
│  │    ┌─────────────────────────────────────────────┐     │    │
│  │    │  RDS PostgreSQL 15                          │     │    │
│  │    │  ├─ Instance: db.t3.micro                   │     │    │
│  │    │  ├─ Storage: 20GB gp3                       │     │    │
│  │    │  └─ Multi-AZ: Standby replica               │     │    │
│  │    └─────────────────────────────────────────────┘     │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Stack technique

| Composant | Technologie | Description |
|-----------|-------------|-------------|
| **Backend** | Go 1.24 + Gin | API REST haute performance |
| **Database** | PostgreSQL 15 (RDS) | Base de données relationnelle |
| **ORM** | GORM | Migrations et seeding automatiques |
| **Container** | Docker + K3s | Orchestration légère |
| **CI/CD** | GitHub Actions | Pipeline automatisé |
| **IaC** | Terraform | Infrastructure reproductible |
| **Cloud** | AWS (VPC, EC2, RDS, EIP) | Infrastructure cloud |
| **Docs** | Swagger + Scalar | Documentation API interactive |

### Sécurité

- 🔐 **Secrets Kubernetes** pour credentials
- 🔒 **RDS dans subnets privés** (pas d'accès internet)
- 🛡️ **Security Groups restrictifs** (ports minimaux)
- 🐳 **Image distroless** (surface d'attaque réduite)
- 👤 **User non-root** dans les conteneurs
- 🔍 **Security scanning** (govulncheck, gosec)
- 🔑 **JWT avec refresh tokens**
- 🚫 **Rate limiting** (100 req/min)

---

## 🔄 Pipeline CI/CD

Le pipeline GitHub Actions se déclenche automatiquement sur chaque push `main` :

```
┌─────────────────┐
│  1. Versioning  │  → Génère version SemVer automatique (v0.0.X)
└────────┬────────┘
         │
┌────────▼────────┐
│ 2. Build & Test │  → Compile Go + Tests unitaires + Coverage
└────────┬────────┘
         │
┌────────▼────────┐
│ 3. Security     │  → govulncheck + gosec + SARIF upload
└────────┬────────┘
         │
┌────────▼────────┐
│ 4. Docker       │  → Build multi-stage + Push Docker Hub
└────────┬────────┘
         │
┌────────▼────────┐
│ 5. Terraform    │  → Deploy infrastructure (EC2, RDS, VPC)
└────────┬────────┘
         │
┌────────▼────────┐
│ 6. Kubernetes   │  → Rolling update + Health checks
└────────┬────────┘
         │
┌────────▼────────┐
│ 7. Notification │  → Webhook avec URLs cliquables
└─────────────────┘
```

**⏱️ Durée totale** : ~8-12 minutes

**📊 En fin de déploiement**, GitHub affiche :
- 🔗 Lien Scalar (documentation interactive)
- 💚 Lien Health Check
- 📄 Lien Swagger JSON
- 🔐 Liens Register/Login

---

## 📦 Structure du projet

```
auth-service/
├── .github/workflows/          # Pipeline CI/CD
│   └── ci-cd.yml
├── cmd/                        # Point d'entrée
│   └── main.go
├── internal/                   # Code métier
│   ├── controllers/            # Handlers API
│   ├── middleware/             # Middlewares
│   ├── models/                 # Modèles de données
│   ├── repository/             # Couche données
│   ├── services/               # Logique métier
│   └── seeder/                 # Données de test
├── tests/                      # Tests unitaires
├── docs/                       # Documentation Swagger
│   ├── swagger.json
│   └── swagger.yaml
├── k8s/                        # Manifestes Kubernetes
│   ├── deployment.yaml
│   └── namespace.yaml
├── terraform/                  # Infrastructure as Code
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   └── user-data.sh
├── Dockerfile                  # Image multi-stage
├── docker-compose.yml          # Dev local
├── Makefile                    # Commandes dev
└── README.md
```

---

## 🛠️ Développement local

### Commandes Make

```bash
make help           # Afficher toutes les commandes
make build          # Build l'application
make run            # Lancer en local
make test           # Tests avec coverage
make test-coverage  # Rapport HTML coverage
make lint           # Linter le code
make docker-build   # Build image Docker
make docker-run     # Run conteneur Docker
make clean          # Nettoyer les artifacts
make deps           # Installer dépendances
```

### Avec Docker Compose

```bash
# Démarrer tous les services (app + PostgreSQL)
docker-compose up -d

# Voir les logs
docker-compose logs -f auth-service

# Arrêter
docker-compose down
```

### Tests

```bash
# Tous les tests
go test ./... -v

# Avec coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Ou via Make
make test-coverage
```

---

## 🔐 Credentials par défaut

**✅ Le seeding est automatique au démarrage !**

### 👤 Admin (tous les droits)
```
Email:    admin@example.com
Password: Admin123!
Tenant:   default
```

### 🏪 Manager (gestion restaurant)
```
Email:    manager@example.com
Password: Manager123!
Tenant:   default
```

### 👥 User (client standard)
```
Email:    user@example.com
Password: User1234!
Tenant:   aucun
```

---

## 🌐 API Documentation

### Endpoints principaux

| Méthode | Endpoint | Description | Auth |
|---------|----------|-------------|------|
| `GET` | `/health` | Health check | - |
| `GET` | `/scalar` | Documentation Scalar | - |
| `POST` | `/api/auth/register` | Inscription | - |
| `POST` | `/api/auth/login` | Connexion | - |
| `POST` | `/api/auth/refresh` | Refresh token | - |
| `POST` | `/api/auth/logout` | Déconnexion | - |
| `GET` | `/api/user/me` | Profil utilisateur | JWT |
| `GET` | `/api/admin/users` | Liste utilisateurs | Admin |
| `POST` | `/api/admin/roles` | Créer rôle | Admin |
| `GET` | `/internal/users` | Users (service-to-service) | - |

### Documentation interactive

Accédez à la documentation Scalar :

**🌐 En production** : `http://<votre-ip>:30081/scalar`

**🖥️ En local** : `http://localhost:8081/scalar`

### Exemple d'utilisation

```bash
# 1. Register
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234!",
    "tenant_id": "default-tenant-id"
  }'

# 2. Login
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin123!",
    "tenant_id": "default-tenant-id"
  }'

# 3. Profile (avec token)
curl http://localhost:8081/api/user/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## ⚙️ Configuration

### Variables d'environnement (.env)

```env
# Database
DATABASE_URL=postgresql://user:password@host:5432/dbname?sslmode=require

# Application
PORT=8081
GIN_MODE=release
JWT_SECRET=your-super-secret-key-change-this

# CORS
CORS_ORIGINS=http://localhost:3000,http://localhost:5173

# Email (optionnel)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

### GitHub Secrets

Configurer dans `Settings → Secrets and variables → Actions` :

| Secret | Description | Exemple |
|--------|-------------|---------|
| `AWS_ACCESS_KEY_ID` | AWS Access Key | `AKIAIOSFODNN7EXAMPLE` |
| `AWS_SECRET_ACCESS_KEY` | AWS Secret Key | `wJalrXUtnFEMI/K7MDENG/...` |
| `AWS_SESSION_TOKEN` | AWS Session (si nécessaire) | `IQoJb3JpZ2luX2VjEPT//...` |
| `AWS_REGION` | Région AWS | `us-east-1` |
| `DOCKER_USERNAME` | Docker Hub username | `cocostee` |
| `DOCKER_PASSWORD` | Docker Hub token | `dckr_pat_xxx...` |
| `WEBHOOK_URL` | URL notifications | `https://discord.com/api/webhooks/...` |

---

## 📊 Monitoring et Coûts

### Health Checks

```bash
# Application health
curl http://<your-ip>:30081/health

# Kubernetes pods
kubectl get pods -n production

# Logs en temps réel
kubectl logs -f -n production -l app=auth-service

# Metrics HPA
kubectl get hpa -n production auth-service-hpa
```

### Estimation des coûts AWS

| Service | Instance | Prix/mois |
|---------|----------|-----------|
| **EC2** | t3.medium (2 vCPU, 4GB) | ~$30 |
| **RDS** | db.t3.micro (1 vCPU, 1GB) | ~$15 |
| **EIP** | Elastic IP | ~$3.60 |
| **Data Transfer** | 10GB sortant | ~$0.90 |
| **Total** | | **~$50/mois** |

---

## 🤝 Contribution

1. Fork le projet
2. Créer une branche (`git checkout -b feature/AmazingFeature`)
3. Commit (`git commit -m 'feat: Add AmazingFeature'`)
4. Push (`git push origin feature/AmazingFeature`)
5. Ouvrir une Pull Request

### Conventions de commit

- `feat:` Nouvelle fonctionnalité
- `fix:` Correction de bug
- `docs:` Documentation
- `chore:` Maintenance
- `refactor:` Refactoring
- `test:` Tests

---

## 📝 License

Ce projet est sous licence MIT.

---

## 👥 Auteurs

**MAALSI Team - BLOC 1**

Projet pédagogique de déploiement CI/CD avec Terraform + Kubernetes

---

## 🙏 Remerciements

- [Go](https://golang.org/) - Langage backend
- [Gin](https://gin-gonic.com/) - Framework web
- [K3s](https://k3s.io/) - Kubernetes léger
- [Terraform](https://www.terraform.io/) - Infrastructure as Code
- [GitHub Actions](https://github.com/features/actions) - CI/CD
- [Scalar](https://scalar.com/) - Documentation API moderne
- [AWS](https://aws.amazon.com/) - Cloud provider

---

<div align="center">

**Made with ❤️ by MAALSI Team**

⭐ Star le projet si vous l'aimez !

</div>
