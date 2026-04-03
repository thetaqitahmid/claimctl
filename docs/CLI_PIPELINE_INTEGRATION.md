# CLI Reservation Capabilities and Pipeline Integration

## Overview

The claimctl CLI fully supports reserving resources by **name**, **type**,
and **labels**. When no resource is available, users are automatically placed in
a queue. The CLI includes built-in wait functionality for seamless CI/CD
pipeline integration.

## Key Capabilities

- **Wait Functionality**: Blocks until resource is active.
- **Fail-Fast Options**: `--no-queue` to skip queuing if busy.
- **Quality Gates**: `--require-healthy` to avoid broken resources.
- **Robustness**: Automatic API retries for network resilience.

## Reservation Methods

### 1. Reserve by Resource ID

```bash
claimctl reserve 12
```

Direct reservation using the resource's numeric ID.

### 2. Reserve by Name

```bash
claimctl reserve --name "Meeting Room A"
```

- Case-insensitive exact name matching
- Searches all resources to find the match
- Fails if no resource with that exact name exists

### 3. Reserve by Type

```bash
claimctl reserve --type "Conference Room"
```

- Finds the **first available** resource of the specified type
- Only considers resources with status "Available"
- Useful when you don't care which specific resource you get

### 4. Reserve by Label

```bash
claimctl reserve --label "gpu"
```

- Finds the **first available** resource with the specified label
- Case-insensitive label matching
- Resources can have multiple labels

### 5. Combined Type and Label

```bash
claimctl reserve --type "test-environment" --label "gpu"
```

- Finds first available resource matching BOTH criteria
- Allows precise resource selection without knowing the ID

### 6. Timed Reservations

```bash
claimctl reserve --type "build-server" --duration "2h"
```

- Automatically releases after the specified duration
- Supports formats: `1h`, `30m`, `1h30m`, etc.

## Built-in Wait Functionality

### Reserve and Wait

The `--wait` flag makes the CLI wait for queued reservations to become active:

```bash
claimctl reserve --type "gpu-server" --wait
```

**Options:**

- `--wait` - Wait for reservation to become active
- `--timeout <seconds>` - Maximum wait time (default: 300s)
- `--poll-interval <seconds>` - Polling interval (default: 5s)
- `--quiet` - Only output reservation ID (useful for scripting)

**Example with all options:**

```bash
claimctl reserve \
  --type "gpu-server" \
  --label "cuda" \
  --duration "1h" \
  --wait \
  --timeout 600 \
  --poll-interval 10 \
  --json
```

### Wait for Existing Reservation

If you already have a reservation ID, use the wait subcommand:

```bash
claimctl reservations wait 42
```

**Options:**

- `--timeout <seconds>` - Maximum wait time (default: 300s)
- `--poll-interval <seconds>` - Polling interval (default: 5s)

## Queue Behavior

When a resource is busy, the CLI automatically queues the reservation:

```bash
$ claimctl reserve --type "gpu-server"
Searching for resource... (Type: gpu-server, Label: )
Found available resource: GPU-01 (ID: 5)
Successfully reserved resource 5. Reservation ID: 42

NOTE: Resource is currently busy. You have been added to the queue at
position 2.
To cancel this request, run: ./claimctl cancel 42
```

### Queue Information

- `status`: "queued" when waiting, "active" when resource is acquired
- `queuePosition`: Your position in the queue (1 = next in line)
- `id`: Reservation ID for tracking and cancellation

## Checking Reservation Status

### List Your Reservations

```bash
claimctl reservations list
```

Output:

```
ID  RESOURCE    TYPE           STATUS   QUEUE POS  CREATED
42  GPU-01      gpu-server     Queued   2          2026-02-06 13:15
43  Build-03    build-server   Active   -          2026-02-06 13:10
```

### Get Specific Reservation Status

```bash
claimctl reservations status 42
```

Output:

```
Reservation ID: 42
Resource ID: 5
Status: Queued
Queue Position: 2
Created At: 2026-02-06 13:15:30
```

### JSON Output for Parsing

```bash
claimctl reservations list --json
claimctl reservations status 42 --json
```

## Exit Codes

The CLI uses standard exit codes for different scenarios:

- `0` - Success
- `1` - General error
- `2` - Timeout waiting for resource
- `3` - Reservation was cancelled
- `4` - Resource/reservation not found
- `5` - Authentication failed
- `5` - Authentication failed
- `6` - Resource busy (with --no-queue flag)

**Example usage in scripts:**

```bash
if claimctl reserve --type "gpu" --wait --timeout 60; then
  echo "Resource acquired"
else
  exit_code=$?
  case $exit_code in
    2) echo "Timeout - resource still busy" ;;
    3) echo "Reservation was cancelled" ;;
    *) echo "Error occurred" ;;
  esac
  exit $exit_code
fi
```

## Pipeline Integration

### Simple Pipeline Example

### Simple Pipeline Example

With built-in wait functionality, pipelines become much simpler:

```bash
#!/bin/bash
set -e

# Reserve and wait for activation
reservation_json=$(claimctl reserve \
    --type "test-environment" \
    --label "gpu" \
    --duration "30m" \
    --wait \
    --json)

reservation_id=$(echo "$reservation_json" | jq -r '.id')

# Ensure cleanup on exit
trap "claimctl release $reservation_id" EXIT INT TERM

# Run your tests
pytest tests/

# Resource is automatically released by trap
```

### Advanced Pipeline Example

See [`examples/pipeline_example.sh`](file:///home/taqi/web-
dev/claimctl/examples/pipeline_example.sh) for a complete example with
error handling.

### Fail-Fast Pipeline Example

For CI jobs that shouldn't wait in a queue (e.g., quick linting checks or low-
priority builds), use `--no-queue`:

```bash
if claimctl reserve --type "gpu" --no-queue --quiet; then
  echo "Resource acquired immediately"
  # Run tests...
else
  exit_code=$?
  if [ $exit_code -eq 6 ]; then
    echo "Resource busy - skipping non-critical job"
    exit 0
  else
    echo "reservation failed"
    exit $exit_code
  fi
fi
```

### Quality Gates (Health Checks)

Ensure your pipeline only runs on healthy infrastructure using `--require-
healthy`:

```bash
# Will fail if the assigned resource (or found resource) is not "healthy"
if claimctl reserve --type "gpu" --require-healthy --wait; then
  echo "Acquired healthy resource"
else
  echo "Failed to acquire healthy resource"
  exit 1
fi
```

### Robustness & Reliability

- **API Retries**: The CLI automatically retries failed requests (exponential
  backoff) to handle transient network issues common in CI environments.
- **Non-Interactive Guard**: The `config` command detects non-interactive shells
  and fails fast instead of hanging, preventing pipeline timeouts.

## API Endpoints Used

The CLI and polling scripts use these backend endpoints:

- `GET /api/resources/with-status` - List resources with availability
- `POST /api/reservations` - Create reservation
- `POST /api/reservations/timed` - Create timed reservation
- `GET /api/reservations/:id` - Get specific reservation status
- `GET /api/reservations` - List user's reservations
- `PATCH /api/reservations/:id/complete` - Release reservation
- `PATCH /api/reservations/:id/cancel` - Cancel queued reservation

## Environment Variables

Configure the CLI using:

```bash
export claimctl_URL="https://claimctl.example.com"
export claimctl_TOKEN="your-api-token"
export claimctl_JSON="true"
```

Or use a `.netrc` file:

```
machine claimctl.example.com
password your-api-token
```

## Implementation Details

### Resource Selection Logic (from `cli/cmd/reserve.go`)

1. **Priority Order**:
   - Positional argument (ID) takes highest priority
   - `--name` flag searches for exact name match
   - `--type` and `--label` flags find first available match

2. **Availability Check**:
   - For type/label searches, only "Available" resources are considered
   - Name search doesn't check availability (allows queuing)

3. **Queue Handling**:
   - Backend automatically queues if resource is busy
   - CLI detects `status: "queued"` or `queuePosition > 0`
   - Provides helpful message with cancellation command

### Polling Implementation

The polling script uses the reservation detail endpoint:

```bash
GET /api/reservations/:id
Authorization: Bearer <token>
```

Response:

```json
{
  "id": 42,
  "resourceId": 5,
  "userId": 10,
  "status": "active",
  "queuePosition": null,
  "startTime": 1738845300,
  "endTime": 0,
  "createdAt": 1738845200
}
```

Status values:

- `"queued"` - Waiting in queue
- `"active"` - Resource acquired and in use
- `"completed"` - Reservation finished
- `"cancelled"` - Reservation cancelled

## Next Steps

To use in your pipeline:

1. Install `jq`: `sudo apt-get install jq`
2. Set environment variables or configure `.netrc`
3. Make scripts executable: `chmod +x examples/*.sh`
4. Integrate into your CI/CD workflow

For questions or issues, refer to the main README or CLI help:

```bash
claimctl reserve --help
```
