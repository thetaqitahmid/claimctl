# Deployment & Configuration

claimctl is designed to be easily deployed using Docker and Docker Compose.

## Prerequisites

- Docker Engine
- Docker Compose
- Make (optional, for convenience commands)

## Quick Start (Development)

Run these commands from the repository root. Source development requires
Go, Node.js with npm, Bash 4.3 or later, OpenSSL, curl, the PostgreSQL client
(`psql`), Make, Docker Engine, and the Docker Compose plugin.

```bash
make dev_up
```

On first use, the command creates a private `backend/.env` with a generated
administrator password and encryption key. Open that file to find
`ADMIN_PASSWORD`, then sign in as `admin@example.com`. Existing configuration
is never overwritten. Bootstrap credentials do not reset an existing account.

The command starts PostgreSQL using `tests/db/postgres-compose.yaml`, waits
for its health check, installs frontend dependencies when needed, and builds
the backend. The backend applies migrations to the database configured in
`backend/.env` and creates the initial administrator and default space.
Once the backend is ready, demo spaces and resources are seeded automatically.

The services are available at:

1. PostgreSQL: `127.0.0.1:5432`
2. Backend: `http://localhost:3000`
3. Frontend: `http://localhost:5173`

Keep `DATABASE_URL` in `backend/.env` consistent with the database you intend
to use. The generated default matches the local Compose database. Ensure
ports 5432, 3000, and 5173 are free before starting. Vite fails rather than
silently switching ports, which would break the configured origin policy.

Press Ctrl+C to stop both application processes. If either process fails,
the other is also stopped. PostgreSQL remains running for the next startup.
To stop the database container afterward:

```bash
make dev_down
```

For container based evaluation without installing Go or Node.js, use the
[README instructions](../README.md#try-it-locally) instead.

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

`make dev_up` seeds four spaces and 16 resources automatically. Repeated
runs preserve existing records without duplicates. Use `make seed` to rerun
it manually, then refresh the Resources page.

### Indefinite reservations

Indefinite reservations are supported but disabled by default. To enable
the explicit Indefinite choice in the reservation dialog, add this to
`backend/.env` and restart the backend:

```bash
RESERVATION_ALLOW_INDEFINITE=true
```

Omitting a reservation duration uses the server default of one hour, not
an indefinite reservation. This policy is global, not specific to a
resource. See [reservation claims](reservation-claims.md) for all policy
options.

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
- Scale out: `replicaCount: 3` or enable autoscaling, add a
  `PodDisruptionBudget` (prefer `maxUnavailable`), and configure
  `topologySpreadConstraints` /
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
     # Optional, defaults to "db-password"
     existingSecretPasswordKey: "db-password"
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

## Database backup and recovery

Use PostgreSQL native backups, not application JSON exports. Follow
[Backup and recovery](BACKUP_RESTORE.md) for `pg_dump`, `pg_restore`, offline
runtime cleanup, and validation of a fresh claimctl deployment.

Restore the full database before starting application replicas or migration
jobs. Supply the original `APP_ENCRYPTION_KEY`, the same manifest and Secret
values, and a compatible application image. A new instance reads existing
accounts, resources, settings, and encrypted signing keys from the recovered
database; no application import or seed script is required. Startup does not
clear runtime state automatically, so complete offline cleanup first.
