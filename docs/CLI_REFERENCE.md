# claimctl CLI

The **claimctl CLI** (`claimctl`) is a powerful command-line interface
for managing resources, reservations, and webhooks in the claimctl system.
It is designed for both interactive use and automation scripts.

## Installation

```bash
cd cli
go build -o claimctl .
```

## Configuration

The CLI supports flexible configuration to suit different environments (local,
CI/CD, servers). It loads settings in the following order of precedence:

1. **Global Flags**: Specify per-command overrides.

- `--url`: Server URL
- `--token`: API Token

2. **Environment Variables**:

- `claimctl_URL`
- `claimctl_TOKEN`
- `claimctl_JSON` (true/false)

3. **Config File**: `~/.config/claimctl/config.json`

- A JSON file containing your persistent settings.
- Example:

```json
{
  "url": "https://api.github.com/thetaqitahmid/claimctl",
  "token": "your-api-token"
}
```

4. **.netrc File**:

- Useful for systems where credentials are centrally managed in `~/.netrc`.
- Add an entry matching your server's hostname:

```
machine api.github.com/thetaqitahmid/claimctl
password <YOUR_API_TOKEN>
```

## Global Flags

- `--json`: Output results in JSON format (ideal for automation/piping). Can
  also be set via `claimctl_JSON=true`.
- `--url`: Server URL.
- `--token`: API Token.
- `--netrc`: Use .netrc for authentication (default `true`).

## Usage

### General

```bash
# Print CLI version
./claimctl version
```

### Resource Management

Manage the physical or digital resources available for reservation.

**List Resources**

```bash
# List all resources
./claimctl resources list

# Filter by Type or Label
./claimctl resources list --type "Meeting Room"
./claimctl resources list --label "projector"

# Filter using Boolean Label Expressions (AND, OR, NOT, brackets)
./claimctl resources list --label-expr "gpu AND ubuntu"
./claimctl resources list --label-expr "(frontend OR backend) AND NOT deprecated"
```

**Create Resources**

```bash
# Single Creation
./claimctl resources create --name "Room A" --type "Room" --label "quiet"

# Bulk Creation (from JSON file)
./claimctl resources create --file resources.json
```

**Delete Resources**

```bash
./claimctl resources delete <resource-id>
```

### Reservations

Make and manage bookings.

**Reserve a Resource**

```bash
# Reserve by ID (Default)
./claimctl reserve 123

# Reserve by Name
./claimctl reserve --name "Room A"

# Reserve First Available (Auto-Assignment)
./claimctl reserve --type "Desk"
./claimctl reserve --label "dual-monitor"

# Specify Duration
./claimctl reserve 123 --duration 2h

# Wait for queued reservation to become active
./claimctl reserve --type "gpu-server" --wait

# With custom timeout and polling interval
./claimctl reserve --type "gpu-server" --wait --timeout 600 \
 --poll-interval 10

# Fail fast if resource is busy (don't queue)
./claimctl reserve --type "gpu" --no-queue

# Fail if resource is not healthy
./claimctl reserve --type "gpu" --require-healthy

# Quiet mode for scripting (only outputs reservation ID)
./claimctl reserve --type "gpu" --wait --quiet
```

> **Note**: If a resource is busy, you will be added to the queue >
> automatically. Use `--wait` to automatically wait for activation.

**Manage My Reservations**

```bash
# List active and queued reservations
./claimctl reservations list

# Get status of specific reservation
./claimctl reservations status <reservation-id>

# Wait for a reservation to become active
./claimctl reservations wait <reservation-id> --timeout 300

# Release (End) an active reservation (JSON output supported)
./claimctl release <reservation-id> --json

# Cancel a pending/queued reservation (JSON output supported)
./claimctl cancel <reservation-id> --json
```

### Webhooks (Admin)

Manage webhooks for integrating with external systems (Slack, Email, etc.).

**Manage Definitions**

```bash
# List Webhooks
./claimctl webhooks list

# Create Webhook (Single)
./claimctl webhooks create --name "Slack" --url "https://hooks.slack.com/..."

# Create Webhooks (Bulk)
./claimctl webhooks create --file webhooks.json

# Delete Webhook
./claimctl webhooks delete <webhook-id>
```

**Format for webhooks.json:**

````json
[
 {
 "name": "Slack Notification",
 "url": "https://hooks.slack.com/services/...",
 "method": "POST",
 "headers": {"Content-Type": "application/json"},
 "template": "{\"text\": \"{{.message}}\"}",
 "description": "Slack webhook for notifications"
 }
]

# Attach
./claimctl webhooks attach <resource-id> <webhook-id> \
 --events "reservation.created,reservation.cancelled"

# Detach
./claimctl webhooks detach <resource-id> <webhook-id>

### Secrets Management

Manage secrets for use in webhook templates.

```bash
# List all secrets
./claimctl secrets list

# Create a secret
./claimctl secrets create --key "SLACK_TOKEN" --value "xoxb-..." \ --desc
"Slack API token"

# Update a secret
./claimctl secrets update <secret-id> --value "new-value" \ --desc "Updated
description"

# Delete a secret
./claimctl secrets delete <secret-id>
````

### Health Checks

Configure and monitor health checks for resources.

```bash
# Configure health check for a resource
./claimctl healthcheck config <resource-id> --type http \ --target
"https://example.com/health"

# Get health check configuration
./claimctl healthcheck get <resource-id>

# Get current health status
./claimctl healthcheck status <resource-id>

# View health check history
./claimctl healthcheck history <resource-id> --limit 20

# Trigger immediate health check
./claimctl healthcheck trigger <resource-id>

# Delete health check configuration
./claimctl healthcheck delete <resource-id>
```

## CI/CD Integration

The CLI is designed for seamless CI/CD pipeline integration with built-in
wait functionality, standard exit codes, and automatic retries.

### Built-in Robustness

- **API Retries**: The CLI automatically retries failed API requests
  (exponential backoff) to handle transient network issues in CI environments.
- **Fail Fast**: Use `--no-queue` to fail immediately if a resource is busy,
  rather than blocking the pipeline.
- **Quality Gates**: Use `--require-healthy` to ensure you don't reserve a
  broken environment.
- **Non-Interactive Guard**: The `config` command fails fast in non-interactive
  terminals to prevent pipelines from hanging.

### Built-in Wait Functionality

No external polling scripts needed! The `--wait` flag handles everything:

```bash
# Reserve and wait for activation in one command
./claimctl reserve --type "test-environment" --wait --timeout 600
```

**Available Options:**

- `--wait` - Wait for reservation to become active
- `--timeout <seconds>` - Maximum wait time (default: 300s)
- `--poll-interval <seconds>` - Polling interval (default: 5s)
- `--quiet` - Only output reservation ID

### Exit Codes

The CLI uses standard exit codes for different scenarios:

- `0` - Success
- `1` - General error
- `2` - Timeout waiting for resource
- `3` - Reservation was cancelled
- `4` - Resource/reservation not found
- `5` - Authentication failed
- `6` - Resource busy (with --no-queue flag)

**Example with error handling:**

```bash
if ./claimctl reserve --type "gpu" --wait --timeout 60; then echo "Resource
acquired" else exit_code=$? case $exit_code in 2) echo "Timeout - resource still
busy" ;; 3) echo "Reservation was cancelled" ;; *) echo "Error occurred" ;; esac
exit $exit_code fi
```

### Pipeline Example

```bash
#!/bin/bash
set -e

# Reserve and wait for activation
reservation_id=$(./claimctl reserve \ --type "test-environment" \ --label
"gpu" \ --duration "30m" \ --wait \ --quiet)

# Ensure cleanup on exit
trap "./claimctl release $reservation_id" EXIT INT TERM

# Run your tests
pytest tests/

# Resource is automatically released by trap
```

See [`CLI_PIPELINE_INTEGRATION.md`](CLI_PIPELINE_INTEGRATION.md)
for comprehensive CI/CD integration guide.

See [`CLI_EXIT_CODES.md`](CLI_EXIT_CODES.md) for detailed
exit code documentation and examples.

## Automation & Scripting

The CLI is built for automation. Use the `--json` flag to parse output
in your scripts.

**Example: Find a Resource ID and Reserve It**

```bash
RESOURCE_ID=$(./claimctl resources list --type "Desk" --json | jq
'.[0].id') ./claimctl reserve $RESOURCE_ID --duration 8h
```

**Example: Reserve and Wait with JSON Output**

```bash
reservation_json=$(./claimctl reserve \ --type "gpu-server" \ --wait \
--json)

reservation_id=$(echo "$reservation_json" | jq -r '.id') resource_id=$(echo
"$reservation_json" | jq -r '.resourceId')

echo "Reserved resource $resource_id with reservation $reservation_id"
```

**Example: Bulk Import**
Create a `resources.json` file:

```json
[
  { "name": "Lab 1", "type": "Lab", "labels": ["gpu"] },
  { "name": "Lab 2", "type": "Lab", "labels": ["gpu"] }
]
```

Run:

```bash
./claimctl resources create --file resources.json
```

**Example: Check Reservation Status**

```bash
status=$(./claimctl reservations status 42 --json | jq -r '.status') if [
"$status" = "active" ]; then echo "Reservation is active" fi
```
