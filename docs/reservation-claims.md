# Reservation policy and atomic claims

`GET /api/reservations/policy` requires authenticated `read` access. Response:

```json
{
  "defaultDuration": "1h",
  "maxDuration": "24h",
  "maxLiveClaims": 8,
  "allowIndefinite": false
}
```

Configure all replicas identically with these environment variables:

* `RESERVATION_DEFAULT_DURATION`, default `1h`
* `RESERVATION_MAX_DURATION`, default `24h`
* `RESERVATION_MAX_LIVE_CLAIMS`, default `8`
* `RESERVATION_ALLOW_INDEFINITE`, default `false`

Invalid configuration stops service initialization. Default duration must be
positive, at least one second, and no larger than maximum duration. Quota must
be a positive int64. Boolean configuration uses Go ParseBool syntax.

## Atomic selector claim

`POST /api/reservations/claim` requires authenticated `reserve` access.
Cookie callers also require the existing CSRF challenge/header. Bearer token
space restrictions apply even to administrators.

```json
{
  "resourceId": "optional UUID",
  "name": "optional exact name",
  "type": "optional exact type",
  "labelExpr": "gpu AND linux",
  "requireHealthy": true,
  "noQueue": true,
  "duration": "1h",
  "indefinite": false
}
```

At least one selector is required. Selectors combine with AND. `labelExpr`
uses the existing label filter grammar. Selection only considers currently
accessible resources. UUID ordering is deterministic; an available matching
resource is preferred over a busy one. Maintenance resources are excluded.
`requireHealthy` requires the latest persisted health status to be `healthy`;
missing or nonhealthy status does not match. It does not execute a health probe.

Omitted duration means policy default, never indefinite. Duration accepts Go
units and integer days (`1d` is `24h`). Invalid, zero, negative, overflowing,
submicrosecond, and overmaximum durations are rejected. Fractional days are
rejected; use hours instead. Explicit `indefinite:true` requires the policy
opt in and cannot be combined with duration.

The user row is locked before resource rows. Selection, availability checking,
quota counting, and insertion occur inside one transaction. Quota counts both
active and pending reservations across all resources and spaces. One actor
cannot hold duplicate live claims for the same resource. With `noQueue:true`,
busy resources never insert a pending reservation. Without it, an authorized
busy match may create a pending reservation if no available match exists.

Successful responses are HTTP201 with the existing reservation JSON shape.
Malformed input is HTTP400; denied explicit resource access is HTTP403;
no matching available resource, duplicate live claim, quota exhaustion, or
idempotency conflict is HTTP409. Storage failures return sanitized HTTP500.
An unrestricted selector that matches only
inaccessible resources returns409 without exposing those resources.

## Idempotency

Send `Idempotency-Key` on creation. Keys are actor scoped, at most200 bytes.
The parsed request fingerprint, reservation ID, and original response snapshot
are committed atomically in migration22 `claimctl.claim_idempotency`.
Concurrent same-key requests produce one claim and the same response. A key
with a different parsed request returns409. Omitted/default duration and an
explicit duration are different requests. Replay rechecks current resource
access and returns the original snapshot, even after release or cancellation;
it does not resurrect the reservation or emit another creation event.

Claim idempotency records are omitted from backup/restore, like refresh
sessions. Records live as long as their referenced reservation. Deletion
cascades deletion of its keys. Clients must use a fresh random key per logical
operation and must not reuse old keys beyond reservation retention.

Legacy `POST /api/reservations` and `/api/reservations/timed` use the same
policy, quota, atomic locking, scope checks, and idempotency enforcement.
The former accepts optional duration and defaults to the configured duration;
the timed endpoint requires a valid duration. Explicit indefinite and selector
options use `/claim`. Named `/claim` and `/policy` routes are not UUID paths.

## Timestamp and lifecycle integration

Reservation create, activation, release, cancellation, queue position changes,
and expiry use PostgreSQL statement timestamps, floored to epoch seconds.
Release/cancellation end time cannot precede an existing rounded start time.
The existing `check_time_order` constraint remains intact. No sleeps are
required before completing or cancelling a newly created reservation.

Later lease/bundle work should reuse `ClaimRequest`, `ReservationPolicy`, the
user-before-resource lock order, and the atomic idempotency result contract.
Creation events are emitted after commit by the existing delivery mechanism;
transactional delivery is assigned to the subsequent outbox component.
