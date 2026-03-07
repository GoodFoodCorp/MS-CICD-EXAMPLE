# Auth Service - CI/CD Complete

> Microservice d'authentification en Go avec pipeline CI/CD automatisé sur AWS + Kubernetes

[![CI/CD Pipeline](https://img.shields.io/badge/CI%2FCD-GitHub%20Actions-blue)](https://github.com/features/actions)
[![Infrastructure](https://img.shields.io/badge/Infrastructure-Terraform-purple)](https://www.terraform.io/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-K3s-blue)](https://k3s.io/)
[![Cloud](https://img.shields.io/badge/Cloud-AWS-orange)](https://aws.amazon.com/)
[![Go](https://img.shields.io/badge/Go-1.23-blue)](https://golang.org/)

## Features

- **Service d'authentification complet** (JWT, OAuth, 2FA)
- **CI/CD automatisé** avec GitHub Actions
- **Infrastructure as Code** avec Terraform
- **Kubernetes** (K3s) avec auto-scaling (2-5 replicas)
- **RDS PostgreSQL** sur AWS
- **Déploiement continu** avec rolling updates
- **Système de notifications** webhook
- **Monitoring** et health checks
- **Security scanning** automatique
- **Versioning automatique** avec tags GitHub

## Quick Start

```bash
# 1. Setup automatique
make setup

# 2. Déployer l'infrastructure
cd terraform && terraform init && terraform apply

# 3. Configurer GitHub Secrets (voir documentation)

# 4. Push et le pipeline se lance !
git push origin main
```

Voir [QUICKSTART.md](QUICKSTART.md) pour le guide complet.

## Documentation

- **[QUICKSTART.md](QUICKSTART.md)** - Démarrage rapide (5 min)
- **[CICD-GUIDE.md](CICD-GUIDE.md)** - Guide complet CI/CD
- **[Makefile](Makefile)** - Commandes disponibles

## Architecture

```
GitHub Actions → Build & Test → Security Scan → Docker Build → Deploy K8s
                                                                    ↓
                     AWS: VPC + EC2 (K3s) + RDS PostgreSQL + Auto-Update
```

### Stack technique

- **Backend** : Go 1.23 + Gin + GORM
- **Database** : PostgreSQL (RDS)
- **Containerization** : Docker + Kubernetes (K3s)
- **CI/CD** : GitHub Actions
- **Infrastructure** : Terraform + AWS
- **Monitoring** : Kubernetes health checks + HPA

## Pipeline CI/CD

Le pipeline se déclenche automatiquement sur push `main` :

1. **Build & Test** - Compilation Go + tests + coverage
2. **Security Scan** - govulncheck + gosec + nancy
3. **Versioning** - Tag automatique (SemVer)
4. **Docker Build** - Multi-stage + push Docker Hub
5. **Deploy** - Rolling update Kubernetes
6. **Notification** - Webhook de confirmation

**Temps total** : ~10-15 minutes

## Auto-Update

Un service systemd vérifie toutes les 5 minutes s'il y a une nouvelle version sur Docker Hub et met à jour automatiquement l'application avec zero downtime.

## Commandes utiles

```bash
make help          # Afficher toutes les commandes
make test          # Lancer les tests
make monitor       # Dashboard de monitoring
make deploy        # Déployer manuellement
make rollback      # Rollback en cas de problème
make ssh           # SSH vers le serveur
```

## API Endpoints

```
GET  /health              - Health check
POST /api/auth/register   - Inscription
POST /api/auth/login      - Connexion
GET  /api/profile         - Profile utilisateur
...
```

Documentation complète : `http://<server-ip>:8081/docs`

## Sécurité

- Security scanning automatique (gosec, govulncheck)
- Secrets Kubernetes
- RDS dans subnet privé
- Security groups restrictifs
- Image Docker distroless
- User non-root

## Coûts AWS

Estimation : **~$53/mois**

- EC2 t3.medium : ~$30
- RDS db.t3.micro : ~$15
- EIP + Data Transfer : ~$8

## Développement local

```bash
# Installer les dépendances
go mod download

# Lancer en local
make run

# Avec Docker
make docker-run

# Tests
make test
```

## Structure du projet

```
.
├── .github/workflows/    # Pipeline CI/CD
├── cmd/                  # Point d'entrée
├── internal/             # Code métier
│   ├── controllers/      # Contrôleurs API
│   ├── services/         # Logique métier
│   ├── models/           # Modèles de données
│   └── middleware/       # Middlewares
├── terraform/            # Infrastructure as Code
├── k8s/                  # Manifestes Kubernetes
├── scripts/              # Scripts de déploiement
├── tests/                # Tests
└── docs/                 # Documentation API
```

## 🔧 Configuration

### Variables d'environnement

```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=authdb
DB_USER=authuser
DB_PASSWORD=secret
APP_PORT=8081
JWT_SECRET=your-secret-key
```

### GitHub Secrets

| Secret                  | Description                |
| ----------------------- | -------------------------- |
| `AWS_ACCESS_KEY_ID`     | AWS access key             |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key             |
| `DOCKER_USERNAME`       | Docker Hub username        |
| `DOCKER_PASSWORD`       | Docker Hub token           |
| `KUBE_CONFIG`           | Kubernetes config (base64) |
| `WEBHOOK_URL`           | URL webhook notifications  |

## 🐛 Dépannage

```bash
# Vérifier l'état
make monitor

# Logs Kubernetes
make k8s-logs

# Logs Docker local
make docker-logs

# Rollback
make rollback
```

Voir [CICD-GUIDE.md](CICD-GUIDE.md#-dépannage) pour plus de détails.

## 🤝 Contribution

1. Fork le projet
2. Créer une branche (`git checkout -b feature/AmazingFeature`)
3. Commit (`git commit -m 'feat: Add AmazingFeature'`)
4. Push (`git push origin feature/AmazingFeature`)
5. Pull Request

## 📝 License

Ce projet est sous licence MIT.

## 👥 Auteurs

**MAALSI Team**

- Projet pédagogique BLOC 1
- CI/CD avec Terraform + Kubernetes

## 🙏 Remerciements

- [K3s](https://k3s.io/) - Kubernetes léger
- [Terraform](https://www.terraform.io/) - Infrastructure as Code
- [GitHub Actions](https://github.com/features/actions) - CI/CD
- [AWS](https://aws.amazon.com/) - Cloud provider

---

**Made with ❤️ by MAALSI Team**

# Infrastructure re-déployée avec fix curl
