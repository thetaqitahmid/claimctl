# claimctl CLI

The **claimctl CLI** (`claimctl`) is a command-line interface for managing
resources, reservations, webhooks, and more in the claimctl system. It is
designed for both interactive use and automation scripts.

## Installation

```bash
cd cli
go build -o claimctl .
```

## Authentication

The CLI uses **API token authentication**. Tokens are sent via the
`Authorization: Bearer <token>` header on every request.

**To obtain a token:**

1. Log in to the claimctl web UI.
2. Go to **Profile → Tokens**.
3. Generate a new token and copy it.

**Configure the token** using one of the methods below.

## Configuration

Settings are loaded in the following order of precedence:

1. **Global flags** (per-command overrides):
   - `--url`: Server URL
   - `--token`: API token
2. **Environment variables**:
   - `CLAIMCTL_URL`
   - `CLAIMCTL_TOKEN`
   - `CLAIMCTL_JSON` (`true`/`false`)
3. **Config file**: `~/.config/claimctl/config.json`
   ```json
   {
     "url": "https://claimctl.example.com",
     "token": "your-api-token"
   }
   ```
4. **`.netrc` file** (`~/.netrc`):
   ```
   machine claimctl.example.com
     password <YOUR_API_TOKEN>
   ```

Use the `config` command to set persistent values:

```bash
./claimctl config --url "https://claimctl.example.com" --token "your-token"
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--url` | Server URL |
| `--token` | API token |
| `--netrc` | Use `.netrc` for authentication (default: `true`) |
| `--json` | Output in JSON format |
| `--config` | Path to config file |

---

## Commands

### Resources

```bash
# List all resources
./claimctl resources list

# Filter by type or label expression
./claimctl resources list --type "Meeting Room"
./claimctl resources list --label-expr "gpu AND ubuntu"
./claimctl resources list --label-expr "(frontend OR backend) AND NOT deprecated"

# Create a resource
./claimctl resources create --name "Room A" --type "Room" --label "quiet"

# Bulk create from a JSON file
./claimctl resources create --file resources.json

# Update a resource (properties are merged with existing)
./claimctl resources update <id> --name "New Name" --type "Lab"
./claimctl resources update <id> --property ip=10.0.0.1 --property user=admin

# Delete a resource
./claimctl resources delete <id>

# View reservation history for a resource
./claimctl resources history <id>

# Maintenance mode
./claimctl resources maintenance enable <id> --reason "Hardware upgrade"
./claimctl resources maintenance disable <id>
./claimctl resources maintenance history <id>
```

**Bulk resource creation format (`resources.json`):**

```json
[
  { "name": "Lab 1", "type": "Lab", "labels": ["gpu"] },
  { "name": "Lab 2", "type": "Lab", "labels": ["gpu"], "properties": {"rack": "A1"} }
]
```

---

### Reservations

```bash
# Reserve by ID
./claimctl reserve <id>

# Reserve by name
./claimctl reserve --name "Room A"

# Reserve first available resource of a type or label
./claimctl reserve --type "Desk"
./claimctl reserve --label-expr "dual-monitor"

# Reserve with a duration
./claimctl reserve <id> --duration 2h

# Wait for a queued reservation to become active
./claimctl reserve --type "gpu-server" --wait --timeout 600 --poll-interval 10

# Fail immediately if resource is busy (don't queue)
./claimctl reserve --type "gpu" --no-queue

# Fail if resource is not healthy
./claimctl reserve --type "gpu" --require-healthy

# Quiet mode — only outputs the reservation ID (useful for scripting)
./claimctl reserve --type "gpu" --wait --quiet

# List your active and queued reservations
./claimctl reservations list

# Get status of a specific reservation
./claimctl reservations status <reservation-id>

# Wait for a reservation to become active
./claimctl reservations wait <reservation-id> --timeout 300

# View your reservation history
./claimctl reservations history

# Release (complete) an active reservation
./claimctl release <reservation-id>

# Cancel a pending or queued reservation
./claimctl cancel <reservation-id>
```

---

### Spaces (Admin)

```bash
./claimctl spaces list
./claimctl spaces create --name "Lab Floor 2" --desc "Second floor lab"
./claimctl spaces update <id> --name "New Name" --desc "Updated description"
./claimctl spaces delete <id>
```

---

### Groups (Admin)

```bash
./claimctl groups list
./claimctl groups create --name "DevOps" --desc "DevOps team"
./claimctl groups update <id> --name "Platform" --desc "Platform team"
./claimctl groups delete <id>

# Members
./claimctl groups members <group-id>
./claimctl groups add-member <group-id> <user-id>
./claimctl groups remove-member <group-id> <user-id>
```

---

### API Tokens

```bash
# List your tokens
./claimctl tokens list

# Generate a new token
./claimctl tokens generate --name "CI Pipeline"
./claimctl tokens generate --name "CI Pipeline" --expires-in 30d

# Revoke a token
./claimctl tokens revoke <token-id>
```

> The token value is only shown once at generation time. Store it securely.

---

### Webhooks (Admin)

```bash
./claimctl webhooks list
./claimctl webhooks create --name "Slack" --url "https://hooks.slack.com/..."
./claimctl webhooks update <id> --name "Slack Prod" --url "https://..."
./claimctl webhooks delete <id>

# Attach/detach a webhook to a resource
./claimctl webhooks attach <resource-id> <webhook-id> \
  --events "reservation.created,reservation.cancelled"
./claimctl webhooks detach <resource-id> <webhook-id>
```

**Bulk webhook creation format (`webhooks.json`):**

```json
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
```

---

### Secrets (Admin)

```bash
./claimctl secrets list
./claimctl secrets create --key "SLACK_TOKEN" --value "xoxb-..." --desc "Slack API token"
./claimctl secrets update <id> --value "new-value" --desc "Updated description"
./claimctl secrets delete <id>
```

---

### Health Checks

```bash
# Configure a health check
./claimctl healthcheck config <resource-id> --type http \
  --target "https://example.com/health"

# Get configuration
./claimctl healthcheck get <resource-id>

# Get current status
./claimctl healthcheck status <resource-id>

# View history
./claimctl healthcheck history <resource-id> --limit 20

# Trigger an immediate check
./claimctl healthcheck trigger <resource-id>

# Delete configuration
./claimctl healthcheck delete <resource-id>
```

---

### Audit Logs (Admin)

```bash
./claimctl audit-logs
./claimctl audit-logs --limit 100 --offset 50
./claimctl audit-logs --json
```

---

### Backup & Restore (Admin)

```bash
./claimctl backup --output backup.json
./claimctl restore --file backup.json
```

---

## CI/CD Integration

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Timeout waiting for resource |
| `3` | Reservation was cancelled |
| `4` | Resource/reservation not found |
| `5` | Authentication failed |
| `6` | Resource busy (`--no-queue` specified) |

### Example Pipeline Script

```bash
# Reserve a test environment and wait for it to become active
RESERVATION_ID=$(./claimctl reserve --type "test-env" --wait \
  --timeout 600 --quiet)

if [ $? -ne 0 ]; then
  echo "Failed to acquire test environment"
  exit 1
fi

echo "Running tests on reservation $RESERVATION_ID"
# ... run tests ...

./claimctl release "$RESERVATION_ID"
```

### Example: Reserve with JSON output

```bash
reservation=$(./claimctl reserve --type "gpu-server" --wait --json)
reservation_id=$(echo "$reservation" | jq -r '.id')
resource_id=$(echo "$reservation" | jq -r '.resourceId')
echo "Reserved resource $resource_id (reservation: $reservation_id)"
```

### Example: Find and reserve by label

```bash
RESOURCE_ID=$(./claimctl resources list --label-expr "gpu AND ubuntu" \
  --json | jq -r '.[0].id')
./claimctl reserve "$RESOURCE_ID" --duration 4h
```
