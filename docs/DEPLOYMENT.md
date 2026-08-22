# Deployment & Configuration

claimctl is designed to be easily deployed using Docker and Docker Compose.

## Prerequisites

- Docker Engine
- Docker Compose
- Make (optional, for convenience commands)

## Quick Start (Development)

To run the full stack (Frontend + Backend + Database) locally:

```bash
make dev_up
```

This command uses `docker-compose.yml` to spin up:

- Postgres Database (Port 5432)
- Backend API (Port 3000)
- Frontend App (Port 5173/80)

To stop the environment:

```bash
make dev_down
```

## Docker Compose Deployment

You can also deploy the application using Docker Compose commands directly.

### Start the Application

To start the backend, frontend, and database in detached mode:

```bash
docker-compose up -d --build
```

The services will be available at:

- Backend: `http://localhost:3000`
- Frontend: `http://localhost:5173`
- Database: `localhost:5432`

### Stop the Application

To stop and remove the containers:

```bash
docker-compose down
```

### View Logs

To tail the logs of all services:

```bash
docker-compose logs -f
```

## Configuration (.env)

The backend is configured via environment variables. Create a `.env` file in the
`backend/` directory or pass them to your container.

| Variable             | Description        | Default               |
| -------------------- | ------------------ | --------------------- |
| `PORT`               | API Port           | `3000`                |
| `DB_HOST`            | Database Host      | `localhost`           |
| `DB_USER`            | Database User      | `devuser`             |
| `DB_PASSWORD`        | Database Password  | `devpass`             |
| `DB_NAME`            | Database Name      | `devdb`               |
| `OIDC_ISSUER`        | OIDC Issuer URL    | -                     |
| `OIDC_CLIENT_ID`     | OIDC Client ID     | -                     |
| `OIDC_CLIENT_SECRET` | OIDC Client Secret | -                     |
| `APP_ENCRYPTION_KEY` | App Encryption Key | Random (per instance) |

## Database Management

### Migrations

Database schema changes are managed via migrations.

```bash
# Run migrations manually
migrate -path migrations -database "postgresql://..." up
```

The `make backend_up` command handles migrations automatically for dev
environments.

### Seeding

To populate the database with initial test data:

```bash
psql "postgresql://..." -f backend/database/seed.sql
```

## Kubernetes Deployment (Helm)

The `charts/claimctl` directory contains a Helm chart for deploying the
application to Kubernetes.

> The chart defaults target development/small-installation setups: they bundle a
> single-replica PostgreSQL (`postgres.enabled: true`) and auto-generate the
> encryption key. The chart fails closed when required values are missing, so a
> misconfigured release is rejected before anything is applied.

### Production Checklist

- Use an external managed database: `postgres.enabled: false` with `db.*` set.
- Provide existing Kubernetes Secrets for DB credentials
  (`db.existingSecret`) and the encryption key
  (`appEncryption.existingSecret`).
- Pin immutable image tags (`backend.image.tag`, `frontend.image.tag`) instead
  of relying on the chart `appVersion`.
- Enable TLS on the Ingress (`ingress.tls`) and terminate HTTPS at the
  ingress controller.
- Scale out: `replicaCount: 3` or enable autoscaling, add a `PodDisruptionBudget`
  (prefer `maxUnavailable`), and configure `topologySpreadConstraints` /
  pod anti-affinity.
- Enable `networkPolicy.enabled` and configure the
  `networkPolicy.ingressController` selector for your ingress controller.
- Set realistic `resources`, `startupProbe`, `readinessProbe`, and
  `livenessProbe` values for your workload.

### Database Credentials

**External database (recommended for production):**

1. Create a secret containing your database user, password, and name:
   ```bash
   kubectl create secret generic my-db-secret \
       --from-literal=db-user=postgres \
       --from-literal=db-password=securepassword \
       --from-literal=db-name=claimctl
   ```
2. Configure `values.yaml` to use this secret:
   ```yaml
   postgres:
     enabled: false
   db:
     host: "your-db-host"
     port: 5432
     name: "claimctl"
     existingSecret: "my-db-secret"
     existingSecretUserKey: "db-user" # Optional, defaults to "db-user"
     existingSecretPasswordKey: "db-password" # Optional, defaults to "db-password"
   ```

**Without an existing secret** the chart creates a
`<release>-db-credentials` Secret from the provided `db.user`, `db.password`,
and `db.name` values and references it via `secretKeyRef`, so credentials are
never injected into the Deployment as plaintext environment variables.

**Bundled database (development only):** when `postgres.enabled: true`, the
single-replica PostgreSQL dependency is used. This is not a production database
(no replication, backups, or failover). Credentials are configured under
`postgres.userDatabase` and `postgres.settings.superuserPassword`, and should
come from an existing Secret (`postgres.userDatabase.existingSecret`,
`postgres.settings.existingSecret`) in production-like environments.

### App Encryption Key

The `APP_ENCRYPTION_KEY` is used to encrypt sensitive data (sessions, webhook
secrets, API tokens). A random key generated on first startup is **not
persistent**: data becomes undecryptable and replicas diverge. The chart
therefore requires one of the following:

**Option 1: Existing Secret (recommended for production)**

Create a secret containing a 32-byte base64-encoded key:

```bash
kubectl create secret generic claimctl-encryption \
    --from-literal=app-encryption-key="$(head -c 32 /dev/urandom | base64)"
```

```yaml
appEncryption:
  existingSecret: "claimctl-encryption"
  secretKey: "app-encryption-key"
```

**Option 2: Auto-Generation (development convenience)**

An init container generates the key once and stores it in a Kubernetes Secret,
keeping it stable across replicas and restarts:

```yaml
keyGeneration:
  enabled: true
```

_Note: This creates a ServiceAccount, Role, and RoleBinding scoped to the
single generated Secret name so the Pod can create/manage it in its namespace.
Prefer Option 1 when secrets are managed outside Helm._
