# CLI transport and reservation lifecycle

## Transport and configuration

All commands use the same transport configuration. Flags override matching
`CLAIMCTL_*` environment variables, which override the configuration file.
Environment names replace flag hyphens with underscores, for example
`CLAIMCTL_REQUEST_TIMEOUT=15s` and `CLAIMCTL_CA_CERT=/path/to/ca.pem`.

The default file is `$XDG_CONFIG_HOME/claimctl/config.json`, or
`$HOME/.config/claimctl/config.json` when XDG configuration is unset.
An explicit `--config` must exist and parse successfully, except that
`config` can create a new file. Missing default configuration is
allowed. Malformed files and readable-by-others credential files are rejected.
Existing credential files may need `chmod 600` before upgrading the CLI.

`claimctl config` reads the token without echo, writes atomically with mode
`0600`, honors `--config`, and preserves transport settings. It is interactive
and cannot be combined with `--json`. Do not place tokens in shell history.
Prefer `--token-file` or `CLAIMCTL_TOKEN_FILE` pointing to an owner-only file.
An explicit `--token` takes precedence over a configured token file. Otherwise
credentials come from the token file, configured token, then `.netrc` when
`--netrc` is enabled. `.netrc` also requires owner-only permissions.

Remote servers require HTTPS. Plain HTTP is allowed for loopback development;
other HTTP servers need explicit `--allow-insecure-http`. This sends credentials
unencrypted and should only be used on a trusted network. `--ca-cert` adds a
trusted PEM CA without disabling certificate or hostname verification.
Redirects are rejected rather than following them with credentials.
`--request-timeout` defaults to `30s` per HTTP attempt and must be positive.
Responses are bounded to 64 MiB, including backup downloads.

Reads and claim/bundle creation with an idempotency key can retry transport
failures, HTTP 429 and server errors. Other mutations are never automatically
replayed. Retry responses are closed before backoff. Backoff honors
`Retry-After`, capped at one minute, and stops on cancellation. Random keys are
created once per logical claim/bundle operation and reused for all its retries.
Do not reuse explicit keys across database restore or retention boundaries.

JSON results do not include search/progress chatter. Empty histories and
reservation lists return `[]`. Nullable reservation deadlines and epoch times
remain null instead of fabricated zero timestamps. Errors from executable CLI
invocations go to stderr; JSON mode emits a JSON error object there. HTTP 409
maps to exit code 6, authentication failures to 5, and missing objects to 4.

## Atomic acquisition

```sh
claimctl policy
claimctl reserve --type gpu --label-expr 'linux AND fast' \
  --require-healthy --no-queue --duration 1h --json
claimctl reserve --name runner --wait --quiet
claimctl bundle RESOURCE_A RESOURCE_B --duration 1h --json
claimctl renew RESERVATION_ID
claimctl heartbeat RESERVATION_ID --lease-duration 2m
claimctl metrics RESOURCE_ID --from 1700000000 --to 1700086400
claimctl tokens generate --name worker --scopes read,reserve \
  --space-ids SPACE_ID --expires-in 30d
```

Selectors, health checks, quota, availability and queue decisions are evaluated
atomically by the server, not by a client-side list/search race. Selectors
combine with AND. `--no-queue` never inserts a queue entry, including when
combined with `--json`, `--quiet` or `--wait`. Quiet mode emits only the
reservation ID; JSON takes precedence if both output options are supplied.

`reserve` accepts `--lease-duration`, `--indefinite` and `--idempotency-key`.
Indefinite acquisition requires server policy opt-in and excludes duration.
Manual claims with a lease need explicit renewal; use `exec` for automatic
heartbeats. Waiting for a newly created reservation cleans it up on timeout,
cancellation or polling failure, with a separate bounded cleanup context.
Failures to clean up are reported, not silently discarded.

Bundles acquire all requested resources or none. `metrics` without an ID reads
admin aggregate metrics and requires the corresponding account/token access.
Policy, bundle, renewal and metrics commands emit structured JSON directly.

## Managed execution

```sh
claimctl exec --type runner --duration 2h --require-healthy -- \
  make integration-test
claimctl exec RESOURCE_ID --no-queue --lease-duration 1m \
  --heartbeat-interval 15s -- python3 job.py
```

Separate the child executable and arguments with `--`. The child is launched
directly, not through an implicit shell. Use an explicit shell if needed.
`exec` streams child stdin/stdout/stderr and therefore rejects `--json`.

The command acquires a finite hold and a heartbeat lease, renews while queued
and running, and starts the child only after activation. It verifies current
reservation state rather than trusting an idempotency replay snapshot. Default
lease is `30s`, heartbeat is `10s`, activation timeout is 300 seconds, and queue
polling is every 5 seconds. Heartbeat must be less than half the requested
lease.
Activation refreshes local deadlines before execution. Renewal never extends
the requested hold or the absolute hard deadline.

The child receives `CLAIMCTL_RESERVATION_ID` and `CLAIMCTL_RESOURCE_ID`.
`CLAIMCTL_TOKEN` and `CLAIMCTL_TOKEN_FILE` are removed from its environment.
This is not a sandbox: the child retains the user's filesystem and other
ambient privileges.

On Unix, interactive children get the foreground terminal, which is restored
before cleanup. SIGINT and SIGTERM are forwarded to the child's process group.
Irrecoverable heartbeat failure or any lease, hold or hard deadline stops the
process group. After `--kill-grace` (default `5s`), remaining processes are
killed. Descendants left behind on normal child exit are also killed before
release. Other platforms use best-effort direct-child signaling/termination.

Active claims are completed on success, child failure, startup failure,
interrupt or renewal failure. Pending claims are cancelled if activation
fails. Cleanup has an independent 10-second timeout. Child exit codes are
preserved, and Unix signal exits use `128 + signal`. Cleanup failure is
reported; a failed child retains its original exit code. If cleanup cannot reach
the server or the CLI receives SIGKILL, lease expiry is the server-side safety
net.

## Validation

```sh
cd cli
go test -race ./...
go test -shuffle=on -count=10 ./cmd ./pkg/api
python3 tests/exec_pty_test.py
```

The PTY checks use Python's Unix terminal support, a disposable loopback HTTP
fixture and a temporary CLI binary. They cover foreground input and terminal
Ctrl+C, including cleanup and exit status, without touching real resources.
