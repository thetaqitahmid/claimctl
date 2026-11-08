# Authentication and resource security

## Upgrade behavior

Migration `000021_scoped_api_tokens` adds token scopes and optional space IDs.
Existing tokens receive `read` and `reserve`. Existing unlimited expiries become
30 days from migration; later expiries are capped at 90 days from migration.
The down migration removes scope columns but does not restore unlimited expiry.

Existing access JWTs without `credential_version` require authentication again.
JWTs are RS256, require issuer `claimctl` and expiration, and check the current
account on every request. JWT role and status claims do not grant
privileges. Deleted or inactive accounts fail authentication; demotion takes
effect immediately. The credential version is an opaque SHA256 digest of the
stored password hash, never the hash itself. Password updates invalidate old
access JWTs and revoke refresh tokens in the same database transaction.

Resource access uses current space permission, including group membership and
the existing public `Default Space` convention. Admin accounts bypass space
membership, but never an API token's explicit space restriction. Both resource
list forms filter inaccessible resources. Queues, reservations, health data,
history, maintenance history, resource webhooks, and SSE check resource access.
Manual health checks, queue processing, and maintenance writes require admin.
Resource webhook reads also require admin because their records contain secrets.

SSE checks current account and resource permission before each delivery and
account validity every heartbeat. Revoked API tokens cannot continue receiving
events. Streams close on credential revocation, JWT expiry, or after 15 minutes.
Clients may reconnect with refreshed credentials. Unrecognized event targets are
not delivered. Trusted background service contexts have no request principal;
HTTP routes must bind the principal through the authentication middleware.

## Cookie CSRF and CLI migration

`ENABLE_CSRF` defaults to `true`. Keep it enabled in production. Cookies use
`COOKIE_SECURE=true` in production, with `COOKIE_SAMESITE=Strict` by default.
Use `Lax` only where the OIDC redirect flow needs it. Cross-site hosting may
need `None` plus HTTPS. Configure exact origins in `CORS_ALLOWED_ORIGINS`.
Wildcard origins are not appropriate for credentialed requests.

Before cookie-based login or any unsafe request, call `GET /api/auth/csrf` with
credentials included. Its JSON is `{ "csrfToken": "<opaque token>" }`. Preserve
the `csrf_` cookie, then send the returned token as `X-CSRF-Token` alongside
cookies for POST, PUT, PATCH, and DELETE. The token is reusable and renewed for
one hour. The CSRF cookie is intentionally readable; auth cookies stay HTTPOnly.
The frontend base query performs this bootstrap and adds the header, including
on refresh and logout. No frontend component changes are needed.

A rejected challenge is HTTP 403 with `code: "CSRF_INVALID"`. The request has
not reached its handler, so a client can bootstrap and retry once. Do not retry
mutations for arbitrary 403 or 500 responses. Origin headers, when supplied,
must match the API origin or an explicitly configured CORS origin. CLI requests
without Origin still require the challenge cookie and matching header.

The CLI uses bearer API tokens, not cookie login. Browser cookie clients retain
cookies and include the challenge header on refresh, logout, and unsafe calls.
Repeat bootstrap on typed CSRF rejection. This stage does not edit the CLI.

Only successfully verified bearer requests bypass CSRF. Invalid or malformed
Authorization never falls back to cookies. Verified bearer requests discard
access and refresh request cookies; `/api/auth/refresh` cannot use cookie
credentials in that mode. Use cookie mode for refresh, or generate a new API
token when necessary.

`CSRFMiddleware(fiber.Storage)` is the shared storage integration seam. Route
setup currently supplies `nil` for process-local storage. Multi-replica use
requires the later shared-storage stage to supply shared storage; otherwise a
challenge created on another replica fails closed and needs bootstrap retry.
Storage errors fail closed with HTTP 503.

## Scoped API tokens

`POST /api/tokens` requires session JWT for token management. JSON accepts
`name`, optional `expires_in` such as `30d` or `720h`, optional `scopes`, and
optional `space_ids` containing UUID strings. Empty or omitted scopes default
to `["read", "reserve"]`. Default expiry is 30 days; requested expiry must be
positive and at most 90 days. Invalid expiry syntax is HTTP 400, never
unlimited.
The generation response includes `scopes` and `spaceIds`, alongside its existing
fields. List records now also include these fields. Raw tokens are shown once.

`read` covers ordinary resource, space, history, preference, and SSE reads.
Administrative user reads require explicit `users:read` and the current admin
role. User response DTOs never serialize passwords or stored password hashes.
`reserve` allows reservation creation and owner completion/cancellation.
Administrative capabilities require both the current admin role and an explicit
scope. Supported scopes are:

```text
resources:write       health:write        webhooks:write
secrets:read          secrets:write       audit:read
users:read            users:write         spaces:write
groups:write          settings:read       settings:write
reservations:write    metrics:read
```

`webhooks:write` and `groups:write` include their administrative reads.
`reservations:write` covers queue processing, activation, admin reservation
listing, and bulk cancellation. `resources:write` includes maintenance writes.
Specifying scopes replaces defaults, so include `read` and `reserve` explicitly
when desired. Non-admin accounts cannot generate administrative scopes.
Account credentials, channel settings, and token management are not API-token
capabilities. A scoped admin CLI can still manage infrastructure by opting into
only its needed capabilities.

`space_ids` restricts resource/space access further; it never grants membership.
An empty list means all spaces otherwise accessible to the current account.
Space-restricted tokens cannot access global admin, secret, webhook, user/group
administration, or create new spaces. This prevents global operations from
leaking resources outside the token's allowed spaces.

## Bootstrap and refresh recovery

Only first-admin creation requires a strong `ADMIN_PASSWORD` of 8 to 72 bytes.
Existing admins permit restart without that bootstrap environment variable.
Missing or weak first-admin passwords fail startup; no generated secret is
logged. Bootstrap does not reset or promote an existing account with the same
email. Recover through a supervised offline database password reset using a
fresh bcrypt hash and revoke that user's refresh sessions in the same
transaction. Trusted OIDC administration requires an explicitly configured
admin group, not an arbitrary email claim.

Refresh issue locks the account and verifies the authorized credential version.
Rotation locks account then token, returns the locked account snapshot, and
retains consumed tokens as replay evidence while the family remains live.
Replay commits family revocation before returning an error. Password changes
and refresh revocation are one transaction using the same lock ordering.

## Backup and recovery

Use full PostgreSQL native backups and the offline recovery procedure in
[Backup and recovery](BACKUP_RESTORE.md). Application JSON backup endpoints,
CLI commands, and their token scopes have been removed. Existing tokens with
other scopes keep those capabilities; removed scopes grant no capability.

Dumps include credential hashes, encrypted secrets, audit logs, and runtime
state. Preserve the original encryption key separately. Before starting a
recovered deployment, run the documented runtime cleanup to discard refresh
sessions, old coordination state, idempotency results, and delivery jobs.
Access JWTs and retained API tokens are not automatically revoked by recovery.
Review post-backup account changes and revocations before reopening traffic.

## Health-check egress

Health probes deny all destinations until both CIDRs and required ports are
explicitly configured. Ping needs an allowed CIDR but no transport port.

```dotenv
HEALTH_CHECK_ALLOWED_CIDRS=10.20.0.0/16,192.168.50.0/24
HEALTH_CHECK_ALLOWED_PORTS=80,443,8080,22
```

Use only the infrastructure subnets and ports the deployment actually monitors.
Do not use universal allow rules. A single `*` entry in
`HEALTH_CHECK_ALLOWED_PORTS` allows every TCP/UDP port; it is intended for
trusted lab setups only. Loopback, link-local, unspecified, multicast,
cloud metadata addresses, and Azure's platform endpoint remain denied even
when inside an allowed CIDR. IPv4-mapped IPv6 addresses are
normalized before policy checks. Private infrastructure is supported through
explicit CIDRs, including IPv6 private networks except metadata targets.

HTTP accepts only HTTP/HTTPS, without URL credentials or fragments. Redirects
are rejected, not followed. DNS is checked again at dial time; every resolved
address must be allowed, and the validated IP is dialed directly. Proxies from
the environment are not used. TCP and ping also use validated addresses.

Timeout must be 1 to 30 seconds, retry count 0 to 3, and interval 10 to 86400
seconds. The timeout bounds the whole probe including DNS and all retries.
HTTP, TCP, and ping propagate their supplied context cancellation. Fiber does
not expose ordinary client disconnects as canceled request contexts; request
probes remain bounded by their overall timeout. Worker shutdown cancels probes.
