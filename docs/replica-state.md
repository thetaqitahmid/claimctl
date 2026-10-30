# Replica state and deployment

## Shared realtime invalidation

Migration25 adds `claimctl.realtime_events`, `claimctl.shared_storage` and
`claimctl.rate_limits`. These are runtime state, not backup export fields.
Restore clears them in its transaction, suppresses import events and emits one
payload-free `resync` after a successful import. Failed restore rolls back all
clearing. Quiesce replicas before restore, as for durable webhook delivery.
Migration25 down discards runtime state but preserves business/history data.

Every resource/reservation mutation emits a scoped invalidation in the SAME
transaction. Resources use `maintenance_change`; reservations use
`reservation_update`. Payloads contain only UUID `resource_id` and `space_id`,
not names, actors, reservation details or secrets. A resource deletion captures
its OLD space before deletion. Reservation cascades with unknown scope and
resource moves emit only payload-free `resync`. Account credential/role/status,
space, group, membership, permission and API-token access changes emit
payload-free `resync`. Token last-used and login metadata never emit resync,
avoiding read/refetch feedback loops; unchanged access fields emit nothing.

Each process polls PostgreSQL every250ms, fetching at most100 records per poll.
It starts at the latest committed cursor. SSE includes an `id` for durable log
records, but does NOT promise per-client replay or notification delivery.
`Last-Event-ID` is intentionally not a replay request. Each subscription sends
`{"type":"resync","payload":null}` so clients refetch authorized snapshots.
Reconnect, restart, retention gaps and slow-client overflow also require full
refetch. Native EventSource reconnect remains enabled. Old-principal callbacks
are ignored after logout, principal change or unmount.

Per-client queues hold at most100 events. Overflow replaces queued events with
payload-free resync rather than silently losing invalidations. Log history has
one-hour retention and a hard approximate ceiling of10100 records, enforced
inside the serialized insert transaction. Cleanup also removes expired CSRF
and rate-limit rows in batches of1000 each minute. Poll/cleanup DB calls have
5s contexts. DB errors cause resync without advancing the cursor; credential
revalidation fails closed and prevents unauthorized delivery.

Before EVERY SSE event, the server refreshes current account status, password
credential version, role, token expiry/scopes and space restrictions. Scoped
events require current space authorization. Existing resources must still
belong to the captured space; deleted resources use the safe captured scope.
Revoked credentials disconnect the stream. Heartbeats revalidate every30s.
Streams end at credential expiry or15min, whichever comes first. Client
socket failures cancel subscriptions when a frame/heartbeat flush detects the
closed connection. Process shutdown cancels subscriptions independently of
socket detection. Nginx buffering/cache are disabled; read timeout is960s.

The frontend hook handles resync, maintenance and reconnect by invalidating
resource/reservation queries without fabricated notifications. Broader
principal-owned API cache clearing remains the frontend integration stage's
responsibility; closing the SSE stream does not itself erase Redux cache.

## Commit ordering and lock ownership

A PostgreSQL transaction advisory lock,725025, is acquired BEFORE any SSE ID is
allocated. All emitting table statements and log inserts acquire that same
lock before statement execution; capture triggers acquire it again before
inserting. `SQLStore.ExecTx` acquires it before business row locks. Backup
restore acquires it before truncation. The sequence therefore cannot expose a
larger committed ID while a lower-ID writer remains uncommitted. Rollback may
leave sequence gaps; those gaps are not lost committed events.

The correctness tradeoff is serialization of store transactions, including
some unrelated refresh/health/cleanup operations. This can reduce mutation
throughput across replicas. Lock waits fail after5s instead of waiting forever.
No network operation runs while this SSE lock is held. Transactions outside
`SQLStore.ExecTx` that lock business rows BEFORE writing emitting tables MUST
first acquire `pg_advisory_xact_lock(725025)`. Do not bypass this lock order
with raw administrative writers. The application's only raw mutating
transaction, backup restore, follows that order. Read-only queries are not
serialized.

## CSRF and distributed counters

Browser CSRF challenges are shared in PostgreSQL with a sliding one-hour TTL.
The production routes never use an in-memory fallback. Nil/unavailable storage
fails closed with503; missing, expired or mismatched challenges fail with403.
Only previously verified bearer credentials bypass CSRF. Challenge retrieval
on one replica and cookie/header validation on another require no affinity.

Rate counters use ONE atomic `INSERT ... ON CONFLICT ... RETURNING` statement,
not Fiber's Get/Set limiter. Windows start on the first request and expire by
PostgreSQL time. Denied requests count within that window. A hashed canonical
client IP is namespaced by global/login/refresh limiter. Limits remain60 per
30s globally,5 per minute for login/OIDC login and10 per minute for refresh.
429 includes `Retry-After`; counter storage failures return503, never allow.
A100-request concurrent regression across two actual HTTP instances permits
exactly17 requests for a configured17-request test window.

## Shared signing and encryption keys

JWT initialization uses a single database transaction and advisory lock725026.
Concurrent startups reuse one matching RSA key pair; both encrypted-private and
public settings persist atomically. Environment-provided pairs also persist
atomically. Partial, corrupt, mismatched or undecryptable existing keys cause
startup failure, never replacement. Database/persistence failures are fatal.

ALL replicas must receive the SAME `APP_ENCRYPTION_KEY`, a base64-encoded32-byte
key. Production compose requires it explicitly. Retain it securely for restore;
changing it casually makes existing secrets/settings unreadable. Local
single-instance file fallback uses a filesystem lock and atomic rename, with
0600 permissions. It is not a cross-host key distribution mechanism. A damaged
key file fails closed. Concurrent independent-process regressions verify the
same stable local key; real DB startup regressions verify the shared RSA pair.
The file fallback uses Unix locking. Native Windows backend compilation needs
a portability adaptation and was not part of this Linux deployment scope.

## Trusted proxy and external origin

`TRUSTED_PROXY_CIDRS` is a comma-separated CIDR allowlist for immediate peers
and trusted intermediary hops. Empty means no proxy trust. Invalid CIDRs fail
startup. Untrusted immediate peers supplying forwarded headers receive400.
Trusted requests accept only one `http`/`https` proto and a valid external
host, rejecting malformed/comma-separated origins and unsupported `Forwarded`.
Client-IP traversal proceeds right to left and stops at the first untrusted
hop; attacker-controlled earlier addresses cannot partition rate counters.
Only sanitized canonical values reach Fiber's proxy-aware helpers.

Frontend Nginx trusts `X-Forwarded-Proto` ONLY from `TRUSTED_INGRESS_CIDR`.
Otherwise it uses its own connection scheme. The default127.0.0.1/32 trusts no
remote TLS ingress. Set this to the actual ingress CIDR as observed by Nginx.
The ingress must preserve the original Host, including any nondefault port,
and send one normalized external proto. Nginx forwards that host/proto to the
backend, preserving HTTPS origin through an HTTP frontend/backend hop. Backend
trust must include Nginx and trusted ingress hops needed for client-IP walking.
Do not trust all Internet addresses or untrusted neighboring containers.

Production exposes only frontend port80; backend3000 is container-internal.
Clients must use TLS ingress, not direct host:3000. Production secure cookies
require HTTPS. Development retains backend3000 and frontend5173; its private
network uses172.30.0.0/24 for trusted proxy peers. Configure a different private
subnet if that range conflicts. Nginx service-DNS addresses are resolved at
startup; reload Nginx when the set of backend replica addresses changes.

## Database URL and compose configuration

Native startup gives explicit `DATABASE_URL` precedence. Otherwise DB fields
are converted with URL-safe user/password/database escaping and IPv6-safe host
formatting. The two bundled compose files explicitly clear `DATABASE_URL` and
supply consistent internal `DB_*` values, so an env-file URL cannot redirect the
backend away from their managed database. External database deployments need
an explicit reviewed compose override, not a conflicting env-file URL.

Production has NO database password fallback. Required values are
`DB_PASSWORD`, `APP_ENCRYPTION_KEY`, `TRUSTED_PROXY_CIDRS` and
`POSTGRES_PGDATA`.
Use `docker compose --env-file backend/.env -f docker-compose.prod.yaml config`
to validate interpolation before any startup. Service `env_file` alone does
not supply compose interpolation variables. Quote literal dollars appropriately
in dotenv files; config output escapes dollars for round-trip serialization.
Synthetic config/runtime tests verify punctuation and dollar preservation.
Never print resolved production configuration into public logs.

## PostgreSQL18 MANUAL layout or major-version upgrade

Both compose files mount the named volume at `/var/lib/postgresql`, the parent
of the PostgreSQL18 data directory. `PGDATA` must be exactly
`/var/lib/postgresql/18/docker`. Production requires explicit
`POSTGRES_PGDATA` only after an operator has verified the volume layout.
The existing named volumes are retained, not replaced with empty versioned
volumes. Development binds database access only to127.0.0.1:55432.

The mounted `deployment/postgres-preflight.sh` runs BEFORE the official image
entrypoint. It refuses a legacy root `PG_VERSION`, unexpected populated root
or version directory, symlinked layout, populated data without `PG_VERSION`,
or a data version other than18. Thus rebinding an old volume cannot silently
initialize an empty replacement cluster. Empty verified volumes and canonical
18/docker layouts pass. This guard NEVER moves or upgrades user data.

Before applying the mount change to an existing installation:

1. Obtain and independently verify an off-volume backup and rollback plan.
2. Stop application replicas and database services in a maintenance window.
3. Inspect the existing named volume READ ONLY. Record its real layout and
   PostgreSQL major version. Do not assume an old `/data` mount is compatible.
4. Manually perform a supported dump/restore or `pg_upgrade` procedure into the
   canonical18/docker layout, using verified backups and an operator-reviewed
   procedure. An18 cluster with an old layout still needs manual layout work.
5. Verify data, ownership and version before setting `POSTGRES_PGDATA` and
   validating compose configuration. Start only after the preflight passes.
6. Retain the original backup/rollback material until acceptance is complete.

Do not run `down -v`, rename volumes, move directories automatically, or treat a
new empty cluster as successful upgrade. These source changes were validated
only with disposable fixtures; no user volume was mounted or modified.

## Helm replica configuration

The chart requires explicit `backend.trustedProxyCIDRs`, including actual
frontend pod peers and any ingress hops. There is no universally safe cluster
CIDR default. Do not substitute `0.0.0.0/0` or `::/0`. With TLS ingress, set
`frontend.trustedIngressCIDR` to its observed peer network if API traffic passes
through frontend Nginx. Default loopback trust does not trust remote ingress.

Frontend proxy requests always use the internal backend Service and its actual
port, never the public ingress hostname. The chart clears `DATABASE_URL` and
sets `COOKIE_SECURE` from `backend.cookieSecure` (default true). Plaintext local
port forwarding requires an explicit development-only false setting. Shared
key Secrets remain mandatory. Nginx ingress defaults disable response buffering
and set a 960-second read timeout for SSE. Other ingress controllers need
corresponding operator-managed settings. Avoid duplicate security/environment
names in `extraEnv`; use the named chart settings instead.

Validate before applying, using the actual trusted networks:

```sh
helm lint charts/claimctl --set backend.trustedProxyCIDRs=10.244.0.0/16
python3 deployment/helm_replica_test.py
```

The Helm regression uses synthetic manifests only, requires Helm and PyYAML,
and never applies a release or starts a database.

## Focused validation

Run backend tests with `SECURITY_TEST_DATABASE_URL` pointing ONLY to an owned
loopback PostgreSQL cluster on55432/55433. The helper creates and drops isolated
UUID-named databases. Real replica tests cover two independent pools/apps and
full `NewServer` router/worker instances sharing login, CSRF and signing keys,
delayed commits, rollback, scoped deletion, hidden-space suppression, token
revocation/expiry, mixed bundle/access/autocommit lock order,
retention/overflow,
CSRF expiry/outage, atomic limiter concurrency and concurrent key startup.

`TestReplicaNginxTLSIngressAndSSERealDatabase` also needs Docker and an already
available `nginx:alpine` image. It uses a disposable loopback Nginx container,
no named volumes, and verifies actual API routing, trusted/untrusted ingress
origin behavior and receipt of a tiny SSE frame before stream completion.

Additional deployment checks, using ONLY synthetic fixtures:

```sh
python3 deployment/replica_config_test.py
sh deployment/postgres-preflight_test.sh
```

These checks need Docker plus `nginx:alpine` and `postgres:18.1-alpine` images.
No production compose startup, upgrade or deployment is part of validation.
