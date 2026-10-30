# Reservation leases, bundles and metrics

All endpoints are under `/api`. Calendar functionality is unchanged.

## Leases

Claims, ordinary creation and timed creation accept optional `leaseDuration`,
a positive duration no greater than policy `maxDuration`. Omission preserves
non-heartbeat behavior. All new claims have an absolute `hardEndTime` at
`createdAt + maxDuration`, including pending claims. Activation starts the
requested hold duration but clips it to that absolute deadline.

`POST /reservations/:id/renew` accepts `{ "leaseDuration": "10m" }` or `{}`.
Only the owner can renew, with `reserve` scope and current resource access.
Only existing, live leases can be renewed. Omitted duration reuses the stored
interval. Renewal updates the heartbeat deadline, not FIFO order, creation
time or the requested active hold. The deadline is the minimum of database
statement time plus the lease interval, the absolute hard deadline, and the
active scheduled hold end. Expired leases return409 even before reclamation.
Completed or cancelled claims cannot be resurrected. Invalid durations
return400 and denied access403.

Clients running commands should request both a sufficient finite hold and a
finite lease, heartbeat the lease, and stop work on irrecoverable renewal
failure or the hold/hard deadline. Renewal cannot extend the requested hold.

The expiry worker reclaims expired active and pending claims under the same
resource lock used for renewal and acquisition. Pending claims are cancelled;
active claims are completed. End times are clipped to the elapsed deadline.
Expired queue heads are skipped, the first valid pending claim is promoted,
and remaining queue positions are normalized transactionally. Claim and bundle
availability checks also reclaim stale leases while holding resource locks.
Historical claims predating this migration retain their existing deadlines.

## Bundles

`POST /reservations/bundle` accepts `resourceIds` (nonempty, unique UUIDs),
optional `duration` and optional `leaseDuration`. Duration defaults to policy.
Bundles cannot exceed the actor's live-claim quota. All resources must be
accessible, outside maintenance, and available after stale claim reclamation.
Resources are locked in UUID order after locking the actor. Busy or quota
conflicts return409: no partial claims, holds or new queue entries commit.

`Idempotency-Key` is optional and bounded to200 characters. Actor and operation
scope isolate bundle keys from single-claim keys. Resource order is
canonicalized before fingerprinting. Same-key retries return the full original
snapshot, after verifying current access to every resource. A different
fingerprint conflicts. Result snapshots commit with the claims and survive
individual claim deletion until the actor is deleted. Full database backups
include these keys; the offline [recovery procedure](BACKUP_RESTORE.md) clears
them before a fresh deployment starts. Do not retry old keys across recovery.

Native database backups preserve `lease_duration`, `lease_expires_at` and
`hard_end_time`. Restored elapsed deadlines remain elapsed and are reclaimed
normally.

Migration23 also repairs queue history: its allowed `queue_position_changed`
action needs32-character storage instead of the old20-character column, and
pending queue moves must emit that action. Down migration deliberately retains
this compatible widening and corrected trigger to preserve populated history.
Only lease and bundle additions are removed; history is never relabeled or
removed to force a destructive shrink.

## Metrics

`GET /resources/:id/metrics` requires authenticated resource read access.
`GET /admin/metrics` returns per-resource metrics and requires a current admin.
API tokens must explicitly opt into `metrics:read` for the admin endpoint;
default tokens and space-restricted tokens cannot use it.

Both endpoints accept integer epoch-second `from` and `to`. Windows are
half-open `[from,to)`, positive, at most90 days and not in the future. Defaults
are the previous24 hours. If only `to` is supplied, `from` is24 hours earlier.
Invalid windows return400. Metrics use seconds and a zero-safe denominator.

* `windowSeconds`: requested duration, the utilization denominator.
* `occupiedSeconds`: union of clipped intervals with a real start time;
  pending claims never occupy resources. Open active intervals end at `to` or
  the earliest stored hold, lease or hard deadline. Overlap is counted once.
* `utilization`: occupied seconds divided by window seconds, in `[0,1]`.
* `claimCount`: claims created inside the window, including cancelled claims.
* `openClaims`: claims created before `to` and still live at that boundary,
  using stored end times and deadlines, not just their current status.
* `queueWaitSamples` and `meanQueueWaitSeconds`: activations starting inside
  the window; full creation-to-start wait, including zero-wait acquisitions.
* `openPendingSamples` and `openPendingWaitSeconds`: claims still pending at
  `to`, with censored waits clipped to the requested window. These waits are
  separate from completed activation samples.

Metrics reflect retained reservation records. Deleting historical records
removes their contribution. Aggregate output is per-resource, not a sum that
could obscure different utilization denominators.
