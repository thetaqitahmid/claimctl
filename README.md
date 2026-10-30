# claimctl

[![License: Apache 2.0][license-badge]][apache-license]

[license-badge]: https://img.shields.io/badge/License-Apache_2.0-blue.svg
[apache-license]: https://www.apache.org/licenses/LICENSE-2.0

**Coordinate access to shared test hardware and environments for developers
and CI jobs.**

claimctl is a self-hosted resource reservation tool with a web UI, CLI, and
API. See who is using a resource, claim it for a limited time, and queue when
it is busy. Give people and automation a shared view of ownership instead
of coordinating through Slack messages, spreadsheets, or separate CI locks.

> [!IMPORTANT]
> claimctl is under active development. Breaking changes may occur. Evaluate
> it with noncritical resources before relying on it for production workflows.

## When to use claimctl

claimctl is intended for teams that share scarce resources and need exclusive
access while working:

1. **Test hardware**: Coordinate access to development boards, network
   appliances, physical servers, or shared testbeds.
2. **Shared environments**: Track ownership of staging environments,
   development databases, or test clusters that cannot safely support
   conflicting changes.
3. **Manual work and CI**: Let developers and pipeline jobs participate in
   the same reservation workflow, even when they use different tools.

A typical problem: a developer is debugging on a test appliance while a CI
job tries to reconfigure it. claimctl provides a place to record ownership
and wait for a turn before starting work.

If a lock inside a single CI system already solves your problem, you may not
need another service. claimctl is most relevant when ownership needs to be
visible and coordinated across people and tools.

## How it works

1. An administrator adds resources and organizes them into spaces.
2. A developer or CI job requests a reservation through the UI, CLI, or API.
3. An available resource can be used immediately; requests for a busy
   resource enter its queue.
4. The holder releases the resource when finished. Timed reservations can
   expire automatically, and the queue advances as resources become free.
5. Other users can see reservation updates and ownership history.

For example, a developer can reserve a test board through the browser while
an automated job waits for the same board through the CLI.

### What a reservation does not do

claimctl coordinates ownership **inside claimctl**. It does not, by itself,
block SSH access, stop a running test, provision infrastructure, or prevent
someone from using equipment outside the reservation workflow.

Webhooks can connect reservation events to your own setup, teardown, or
access control automation. Those integrations must implement the actual
operations and enforcement. Reservation expiry does not guarantee that the
previous holder's workload has stopped.

The core workflow is **claim, wait, use, and release**, not a calendar for
booking future time slots. See [Reservations & Queues](docs/RESERVATIONS.md)
for lifecycle details.

See [release validation](docs/VALIDATION.md) for standard checks, disposable
database integration tests, deployment fixtures, and browser acceptance.

## Capabilities

| Area | Available capabilities |
| --- | --- |
| Reservations | Timed ownership, automatic expiry, queues, release, history |
| Resource discovery | Types, labels, filtering, properties, spaces |
| Team visibility | Web UI, live updates via SSE, maintenance mode |
| Automation | CLI, API tokens, reservation event webhooks |
| Access management | Local authentication, OIDC SSO, access groups |
| Operations | Audit logs, encrypted secrets, PostgreSQL native recovery |
| Notifications | Email, Slack, and Microsoft Teams dispatch support |
| Health monitoring | Ping, HTTP, and TCP checks with retries |

Health probes require an explicit egress allowlist. Configure permitted
networks and ports before enabling checks against your infrastructure.

## Try it locally

This setup builds the application from source using Docker Compose. You
need Git, Docker Engine, and the Docker Compose plugin; Go and Node.js are
not required for this container-based evaluation.

The development configuration uses local HTTP, development database
credentials, and published ports. **Do not expose it to an untrusted network
or use it as a production deployment.**

### 1. Clone the repository

```bash
git clone https://github.com/thetaqitahmid/claimctl.git
cd claimctl
```

### 2. Set a bootstrap administrator password

In your shell, set a unique password before starting the application:

```bash
export ADMIN_PASSWORD='ReplaceThisWithYourOwnStrongPassword1!'
```

Replace the example value. The password must contain 8 to 72 bytes, including
uppercase and lowercase letters, a digit, and a supported special character
such as `!`.

### 3. Start the stack

Run this command from the repository root. The inline Compose override
passes the bootstrap credentials to the backend:

```bash
docker compose -f docker-compose-dev.yaml -f - up -d --build <<'YAML'
services:
  backend:
    environment:
      ADMIN_EMAIL: admin@example.com
      ADMIN_PASSWORD: ${ADMIN_PASSWORD:?Set ADMIN_PASSWORD first}
YAML
```

The backend applies database migrations and creates an administrator on
first startup when no administrator exists. The bootstrap password does
not reset an existing account.

Open <http://localhost:5173> and sign in as `admin@example.com` with the
password you chose. The backend is available at <http://localhost:3000>.

To inspect startup or stop the stack:

```bash
docker compose -f docker-compose-dev.yaml logs -f backend
docker compose -f docker-compose-dev.yaml down
```

Stopping the stack preserves the database volume. This is an evaluation
setup: configure persistent encryption keys before storing secrets you
need to retain across container recreation. See the
[Encryption Guide](docs/ENCRYPTION.md).

### 4. Exercise the reservation workflow

1. Add a resource such as `Test Board A` from the administration interface.
2. Reserve it for a short duration.
3. Use another user account to join the queue for that resource.
4. Release the first reservation and observe the queue advance.
5. Review the resource's reservation history.

## Use the CLI

The CLI uses API tokens for authentication. Generate a token from
**Profile → Tokens** in the web UI and keep it private. For CI, store the
token in your CI system's secret storage.

To build the CLI from the repository root, use Go matching the version in
[`cli/go.mod`](cli/go.mod):

```bash
(cd cli && go build -o claimctl .)
export CLAIMCTL_URL='http://localhost:3000'
export CLAIMCTL_TOKEN='<your API token>'
./cli/claimctl resources list
```

Copy a resource UUID from the list, then reserve it and wait if it is busy:

```bash
RESOURCE_ID='<resource UUID from the list>'
./cli/claimctl reserve "$RESOURCE_ID" --duration 30m --wait --timeout 600
```

After a successful reservation, use the returned reservation ID to release
it when your work is finished:

```bash
./cli/claimctl release '<reservation UUID>'
```

For automation, start work only after confirming that the reservation is
active. Arrange cleanup on job completion or failure, and choose a duration
that covers the work. Expiry is a fallback for abandoned ownership, not a
substitute for stopping workloads safely.

The CLI also supports discovery by name, type, and label expression:

```bash
./cli/claimctl reserve --name 'Test Board A' --duration 1h
./cli/claimctl reserve --type 'Test Board' --label-expr 'arm AND linux'
./cli/claimctl reservations list
./cli/claimctl resources history "$RESOURCE_ID"
```

Selection by type or labels searches for an available matching resource;
it is not a waiting queue for an entire pool. Use a resource ID to wait
for a specific busy resource.

See the [CLI Reference](docs/CLI.md) for commands and configuration options,
and [Webhooks & Automation](docs/WEBHOOKS.md) for event integrations.

## Deployment

The repository includes a production Compose configuration at
[`docker-compose.prod.yaml`](docker-compose.prod.yaml) and a Helm chart at
[`charts/claimctl`](charts/claimctl).

Before deploying beyond a local evaluation, configure TLS, database
credentials, a persistent encryption key, authentication, and backups.
Review the configuration rather than assuming development defaults are
suitable for your environment.

See the [Deployment Guide](docs/DEPLOYMENT.md),
[Authentication Guide](docs/AUTHENTICATION.md),
[Encryption Guide](docs/ENCRYPTION.md),
[Backup and Recovery](docs/BACKUP_RESTORE.md), and
[Security Policy](SECURITY.md).

Backups use PostgreSQL tools, not an application JSON export. A fresh
claimctl deployment can use a restored database with the original encryption
key and deployment configuration after the documented offline cleanup.

## Documentation

| Guide | Contents |
| --- | --- |
| [Resources](docs/RESOURCES.md) | Properties, labels, and organization |
| [Reservations](docs/RESERVATIONS.md) | Reservation lifecycle and queues |
| [CLI](docs/CLI.md) | Commands, authentication, and scripting options |
| [Webhooks](docs/WEBHOOKS.md) | Reservation events and external automation |
| [Authentication](docs/AUTHENTICATION.md) | Local accounts and OIDC setup |
| [Administration](docs/ADMINISTRATION.md) | Administrative operations |
| [Deployment](docs/DEPLOYMENT.md) | Deployment configuration and Helm |
| [Encryption](docs/ENCRYPTION.md) | Key configuration and encrypted storage |
| [Recovery](docs/BACKUP_RESTORE.md) | Native backups and fresh deployment |

## Development and contributions

The backend and CLI use Go, the frontend uses React and TypeScript, and
reservation data is stored in PostgreSQL.

For local development outside containers, see the
[Backend Development Guide](docs/BACKEND_DEVELOPMENT.md) and
[Frontend Development Guide](docs/FRONTEND_DEVELOPMENT.md). Use the toolchain
versions specified by the Go modules and frontend dependencies.

Run the repository checks with the development dependencies installed:

```bash
make test
```

This runs backend and CLI tests, plus frontend lint, tests, and a build.
See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

Trying claimctl with a real team? Please
[open an issue](https://github.com/thetaqitahmid/claimctl/issues) describing
what you share, how you coordinate access today, and what made adoption
useful or difficult. Reports about installation friction and missing
workflow support are especially valuable.

## License

claimctl is licensed under [Apache 2.0](LICENSE).
