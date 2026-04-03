# claimctl

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

> [!IMPORTANT]
> This project is currently under development. The code may change frequently
> and may not be stable. Please prepare for breaking changes.

claimctl is a resource management tool designed to streamline the
reservation and organization of resources within teams or organizations. It
allows users to view, reserve, and release resources, making it easier to manage
availability and scheduling in a centralized and efficient manner. claimctl
is ideal for workplaces, libraries, and labs where resources such as servers,
testbeds, books, equipment, meeting rooms, or other shared assets need to be
tracked and managed collaboratively.

## Features

- **Browse Available Resources**: Quickly see the resources available and their
  types.
- **Reserve and Release Resources**: Easily book resources when needed and
  release them after use.
- **Filter Resources by Category**: Find specific types of resources with
  filters to improve search efficiency.
- **Search by Keyword**: Quickly locate resources based on name or type using
  search functionality.
- **Detailed Resource View**: Access detailed information about each resource,
  including its name, type, status, and labels.
- **Track Reservation Status**: Monitor who currently has reserved a resource,
  along with easy-to-use Reserve and Release buttons.
- **Authentication and User Management**: Allow user authentication and access
  based resource management.
- **Resource History**: View previous reservation history to better manage high-
  demand resources with detailed audit trails.
- **Timed Reservations**: Reserve resources for specific durations (1hr, 2hr,
  4hr, or custom) with automatic expiry.
- **Queue System**: Automatic queue management for busy resources with first-
  come-first-served ordering.
- **Real-Time Updates**: Live reservation status updates via Server-Sent Events
  (SSE) for instant notifications.
- **Webhook Integration**: Automate workflows by triggering webhooks on
  reservation events (create, activate, complete, cancel).
- **Secret Management**: Securely store API keys and tokens for webhook
  integrations with encrypted storage.
- **Multiple Authentication Methods**: Support for local auth, LDAP, and OpenID
  Connect (OIDC) for enterprise SSO.
- **Command-Line Interface**: Full-featured CLI (`claimctl`) for
  automation, scripting, and headless operations.
- **Admin Panel**: Comprehensive administration interface for managing users,
  resources, webhooks, and secrets.
- **API Token Authentication**: Generate API tokens for programmatic access and
  CLI usage.
- **Health Check Monitoring**: Automated monitoring of resource availability
  with support for ping, HTTP, and TCP checks with configurable intervals and
  retry logic.

## Use Cases

### 1. **Lab Equipment Scheduling**

- In lab environments, claimctl allows lab managers and researchers to
  reserve equipment like microscopes, chemical storage, or other lab-specific
  resources.
- Keep track of high-cost equipment and maximize utilization efficiency.

### 2. **CI/CD Pipeline Resource Management**

- Manage shared test environments, build servers, or deployment slots in CI/CD
  pipelines.
- Prevent conflicts when multiple teams need access to the same testing
  infrastructure.
- Integrate with automation tools via webhooks to trigger environment
  setup/teardown.

### 3. **IoT and Smart Home Integration**

- Control IoT devices (smart plugs, lights, locks) when resources are reserved
  or released.
- Automate physical access control for labs, studios, or equipment rooms.
- Use webhooks to integrate with Home Assistant, MQTT, or custom IoT platforms.

### 4. **DevOps and Infrastructure Management**

- Reserve cloud resources, VMs, or Kubernetes clusters for development and
  testing.
- Track usage of shared development databases or staging environments.
- Automate resource provisioning through webhook integrations with cloud
  providers.

### 5. **Healthcare Equipment Management**

- Track and reserve medical equipment such as ultrasound machines, wheelchairs,
  or diagnostic tools.
- Monitor equipment health status to ensure availability and reliability.
- Maintain audit trails for compliance and equipment usage tracking.

### 6. **Data Center and Server Management**

- Reserve physical servers, network switches, or rack space for maintenance or
  testing.
- Monitor server health with automated ping, HTTP, or TCP checks.
- Coordinate access to shared infrastructure across multiple teams.
- Integrate with monitoring systems via webhooks for incident response.

### 7. **Manufacturing and Production Line Resources**

- Manage access to production equipment, testing stations, or quality control
  tools.
- Schedule equipment maintenance windows with timed reservations.
- Track equipment utilization and availability across shifts.

### 8. **Vehicle Fleet Management**

- Reserve company vehicles, delivery trucks, or specialized equipment.
- Track vehicle availability and usage patterns.
- Automate notifications when vehicles are reserved or released.

### 9. **Research Facility Resource Sharing**

- Enable researchers to reserve specialized instruments, cleanrooms, or testing
  facilities.
- Queue system ensures fair access to high-demand research equipment.
- Integration with lab management systems via webhooks and API.

## Command-Line Interface

claimctl includes a powerful CLI for automation and scripting:

```bash
# Reserve a resource by ID or criteria
claimctl reserve 123 --duration 2h
claimctl reserve --type "Desk" --label "dual-monitor"

# List and manage resources
claimctl resources list --type "Meeting Room"
claimctl resources create --name "Lab 1" --type "Lab"

# Webhook management
claimctl webhooks list
claimctl webhooks attach <resource-id> <webhook-id>

# Health check management
claimctl healthcheck config <resource-id> --type http --target
"https://example.com"
claimctl healthcheck status <resource-id>
claimctl healthcheck history <resource-id> --limit 20
```

See [CLI Reference](docs/CLI.md) for complete CLI documentation.

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- [Authentication Guide](docs/AUTHENTICATION.md) - Local, LDAP, and OIDC setup
- [Reservations & Queue System](docs/RESERVATIONS.md) - How reservations work
- [Webhooks & Automation](docs/WEBHOOKS.md) - Integrate with external systems
- [Resource Management](docs/RESOURCES.md) - Managing resources and labels
- [Administration](docs/ADMINISTRATION.md) - Admin panel features
- [Deployment Guide](docs/DEPLOYMENT.md) - Docker, Kubernetes, and Helm charts
- [CLI Reference](docs/CLI.md) - Full CLI documentation and examples

## Prerequisites

Before building or running the project locally, ensure you have the following
installed:

- Go (1.21 or later recommending)
- Node.js (18 or later recommending)
- Docker and Docker Compose
- PostgreSQL (if not using Docker for the database)

## Quick Start

### Using Docker Compose

```bash
docker-compose up -d
```

### Using Helm (Kubernetes)

```bash
helm install claimctl ./charts/claimctl
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed deployment
instructions.

## Technology Stack

- **Backend**: Go with PostgreSQL database
- **Frontend**: React with TypeScript
- **Authentication**: JWT, LDAP, OpenID Connect
- **Real-time**: Server-Sent Events (SSE)
- **Deployment**: Docker, Kubernetes, Helm

## Development

claimctl includes several `Makefile` commands to simplify development:

### Root Makefile Commands

- `make dev_up`: Start full development environment (backend + frontend + db)
- `make backend_up` / `make backend_down`: Start/stop backend with database
- `make frontend_up`: Start frontend only
- `make db_up` / `make db_down`: Start/stop PostgreSQL container
- `make migrate_up`: Run database migrations
- `make sqlc`: Regenerate sqlc code
- `make test`: Run all tests

For more detailed development guidelines, please refer to the internal
development documentation.

## License

This project is licensed under the GNU Affero General Public License v3.0
(AGPLv3) - see the `LICENSE` file for details.
