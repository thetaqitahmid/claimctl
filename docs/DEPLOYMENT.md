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
| `LDAP_HOST`          | LDAP Server Host   | -                     |
| `LDAP_PORT`          | LDAP Server Port   | -                     |
| `LDAP_BIND_DN`       | LDAP Bind User     | -                     |
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

### Database Credentials

By default, the chart uses the values in `values.yaml` for database credentials.
For production, it is recommended to use an existing Kubernetes Secret.

**Using an Existing Secret:**

1. Create a secret containing your database user and password:
   ```bash
   kubectl create secret generic my-db-secret --from-literal=db-user=postgres
       --from-literal=db-password=securepassword
   ```
2. Configure `values.yaml` to use this secret:
   ```yaml
   db:
     existingSecret: "my-db-secret"
     existingSecretUserKey: "db-user" # Optional, defaults to "db-user"
     existingSecretPasswordKey: "db-password" # Optional, defaults to "db-password"
   ```

### App Encryption Key

The `APP_ENCRYPTION_KEY` is crucial for decrypting sensitive data (like
sessions). If not provided, the backend generates a random key on startup.

**Important:** For multiple replicas, **you must ensure all replicas use the
same key**.

**Option 1: Manual Configuration**

Set the key directly in `values.yaml` (or via `--set`):

```yaml
appEncryptionKey: "your-32-byte-base64-key"
```

**Option 2: Auto-Generation (Recommended)**

Enable the init container to automatically generate a random key and store it in
a Kubernetes Secret. This ensures consistency across replicas without handling
the key manually.

```yaml
keyGeneration:
  enabled: true
  # image: bitnami/kubectl:latest # Optional: configure kubectl image
```

_Note: This creates a ServiceAccount, Role, and RoleBinding to allow the Pod to
manage Secrets in its namespace._
