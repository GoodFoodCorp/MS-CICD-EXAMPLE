# Auth Service

Microservice **Go** d'**authentification** : identifiants, jetons et rôles. C'est
lui qui émet les JWT que **tous** les autres services valident.

|                         |                                                   |
| ----------------------- | ------------------------------------------------- |
| **Langage / techno**    | Go 1.26, Gin, GORM, PostgreSQL, JWT HS256, bcrypt |
| **Base de données**     | PostgreSQL (interne au compose, non exposée)      |
| **Port HTTP**           | `8081`                                            |
| **Documentation API**   | http://localhost:8081/scalar                      |
| **Explorateur de base** | http://localhost:8090 (adminer, dev)              |

> ℹ️ **Ce service ne gère plus les restaurants.** Ils appartenaient
> historiquement à ce service sous le nom de « tenants » ; ils ont été déplacés
> dans **`franchise-service`**. `auth-service` ne conserve que le `tenant_id` de
> l'utilisateur, comme simple référence (sans jointure).

---

## Architecture — en couches

```
cmd/main.go                # Démarrage, migrations GORM, routes, seeder
internal/
├── controllers/           # Handlers HTTP (auth, admin, rôles, profil)
├── services/              # Logique métier + clients d'autres services
│                          #   franchise_client.go → récupère les restaurants
│                          #   user_client.go      → crée le profil à l'inscription
│                          #   email_service.go    → emails de vérification/reset
├── repository/            # Accès aux données via GORM
├── models/                # User, Role, UserRole, RefreshToken,
│                          # PasswordResetToken, EmailVerificationToken
├── middleware/            # Validation JWT, contrôle de rôle, rate limiting
└── seeder/                # Rôles et comptes de démonstration
tests/                     # Tests unitaires et d'intégration (testify)
```

C'est un service **préexistant** au projet, conservé dans son style d'origine
(couches classiques) plutôt que réécrit en hexagonal comme les services créés
ensuite.

---

## Fonctionnalités

### Authentification

- **Inscription** d'un client (rôle `user` attribué automatiquement)
- **Connexion** : renvoie un jeton d'accès (15 min) et un jeton de
  rafraîchissement (7 jours), **à la fois en cookies HttpOnly et dans le corps de
  la réponse** — le web utilise les cookies, le mobile le `Bearer`
- **Rafraîchissement** du jeton d'accès
- **Déconnexion** avec révocation du jeton de rafraîchissement
- **Vérification d'email** par lien
- **Mot de passe oublié** et **réinitialisation** par jeton
- Contrôle de complexité du mot de passe (8+ caractères, lettres et chiffres)
- Hachage bcrypt (coût 12)
- **Rate limiting** sur les routes d'authentification

### Rôles et administration

- Création de rôles, attribution et retrait à un utilisateur
- Consultation des rôles d'un utilisateur
- Liste, recherche et consultation des utilisateurs (siège)
- Promotion d'un utilisateur en administrateur

### Intégrations sortantes

- À l'inscription, demande à **`user-service`** de créer un profil vierge
  (`POST /internal/profiles`, non bloquant)
- Au démarrage, récupère les restaurants depuis **`franchise-service`** pour
  rattacher les managers au bon restaurant

### Comptes créés au démarrage (seeder)

| Rôle                     | Email                  | Mot de passe  |
| ------------------------ | ---------------------- | ------------- |
| `admin`                  | `admin@example.com`    | `Admin123!`   |
| `manager` (République)   | `manager@example.com`  | `Manager123!` |
| `manager` (Montparnasse) | `manager2@example.com` | `Manager123!` |
| `user`                   | `user@example.com`     | `User1234!`   |
| `livreur`                | `livreur@example.com`  | `Livreur123!` |

---

## Endpoints

| Méthode  | Route                                                                   | Accès                              |
| -------- | ----------------------------------------------------------------------- | ---------------------------------- |
| POST     | `/api/auth/register`                                                    | public                             |
| POST     | `/api/auth/login`                                                       | public                             |
| POST     | `/api/auth/refresh`                                                     | public (jeton de rafraîchissement) |
| POST     | `/api/auth/logout`                                                      | public                             |
| GET      | `/api/auth/verify-email`                                                | public (lien email)                |
| POST     | `/api/auth/forgot-password`                                             | public                             |
| POST     | `/api/auth/reset-password`                                              | public (jeton)                     |
| GET      | `/api/user/me`                                                          | authentifié                        |
| GET      | `/api/admin/users`, `/users/:id`, `/search`                             | `admin`                            |
| POST     | `/api/admin/promote`                                                    | `admin`                            |
| POST/GET | `/api/admin/roles`, `/roles/assign`, `/roles/remove`, `/roles/user/:id` | `admin`                            |
| GET      | `/internal/users`                                                       | interne (service à service)        |
| GET      | `/health`, `/health/db`, `/version`                                     | public                             |

> ⚠️ Ce service expose **`/health`** (et non `/healthz` comme les autres services)
> — c'est la convention d'origine, conservée.

---

## Jeton émis

```json
{
  "sub": "<userId>",
  "email": "user@example.com",
  "tenant_id": "<restaurantId ou chaîne vide>",
  "roles": ["manager"],
  "role_slugs": ["manager"],
  "exp": 1234567890
}
```

Signature **HS256** avec `JWT_SECRET`, partagé avec tous les services.

---

## Dépendances

> **Légende** — 🔴 indispensable (le service ne démarre pas ou ne sert à rien) ·
> 🟠 nécessaire à une fonctionnalité (le reste continue de marcher) ·
> 🟡 optionnelle (dégradation silencieuse, journalisée)

| Dépendance                 | Type | Conséquence si absente                                                                                                                                                                                                                                         |
| -------------------------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **PostgreSQL** (`auth-db`) | 🔴   | Le service démarre en mode dégradé et renvoie `503`                                                                                                                                                                                                            |
| **franchise-service**      | 🟡   | Au démarrage uniquement : sert à rattacher les managers à leur restaurant. Après 5 essais, le seeder abandonne avec un avertissement et les managers n'ont **pas** de `tenant_id` (leur portail sera vide). Un redémarrage de `auth-service` rattrape le coup. |
| **user-service**           | 🟡   | À l'inscription uniquement : la création du profil vierge est un appel **non bloquant**. Sans lui, le compte est créé normalement et le profil sera généré à la volée au premier accès.                                                                        |

**Aucune autre dépendance.** `auth-service` fonctionne seul avec sa base.

### Qui dépend de ce service

**Tous les services** valident les jetons qu'il émet, et le front en a besoin
pour la connexion.

> ℹ️ **La validation du JWT est locale** : chaque service vérifie la signature
> avec le secret partagé, **sans appel réseau à `auth-service`**. Si
> `auth-service` tombe, les jetons déjà émis continuent donc de fonctionner —
> seules la connexion et l'inscription sont impossibles.

---

## Lancement

```bash
docker network create microservices-net   # une seule fois, partagé
cp .env.example .env
docker compose up -d --build
```

### Variables d'environnement

| Variable                | Requis     | Description                                                                    |
| ----------------------- | ---------- | ------------------------------------------------------------------------------ |
| `PORT`                  | non (8081) | Port HTTP                                                                      |
| `DATABASE_URL`          | oui        | Chaîne GORM/Postgres                                                           |
| `JWT_SECRET`            | oui        | Secret HS256 **partagé par tous les services**                                 |
| `FRANCHISE_SERVICE_URL` | non        | Défaut `http://franchise-service:8089`                                         |
| `USER_SERVICE_URL`      | non        | Création du profil à l'inscription                                             |
| `AUTO_VERIFY_EMAIL`     | non        | `true` en dev : compte vérifié d'office (pas de SMTP)                          |
| `FRONTEND_URL`          | non        | URL front utilisée dans les liens d'email (vérification et reset mot de passe) |
| `SMTP_*`                | non        | Envoi des emails de vérification et de réinitialisation                        |

---

## Tests

```bash
go test ./tests/          # tests unitaires et d'intégration (SQLite en mémoire)
```

---

## CI/CD

C'est **le seul projet doté d'un pipeline** (`.github/workflows/ci-cd.yml`, hérité
du projet d'origine) : build, tests, analyse de sécurité, versionnement, build et
publication Docker, déploiement Kubernetes.

> ⚠️ Les **9 autres services n'ont aucune CI** — voir le README racine.
