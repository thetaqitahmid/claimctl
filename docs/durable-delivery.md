# Durable webhook delivery and worker coordination

Migration 24 installs a deferred reservation trigger and two runtime tables:
`claimctl.webhook_outbox` and `claimctl.worker_leases`. Reservation creation,
status transitions, and renewal capture delivery intent in the mutation's
transaction. Rollback discards intent. Creation snapshots include the final
lease and hard end, including selector and bundle claims. Initial lease setup
is not a separate renewal. Replay of an idempotency result emits no new intent.
Deleting a hook or subscription does not cancel already committed intent.
No webhook network request runs in a reservation transaction.

Subscriptions retain their existing event filtering. Events include
`reservation.created`, `reservation.activated`, `reservation.completed`,
`reservation.cancelled`, `reservation.expired`, and `reservation.renewed`.
An expired pending lease emits `reservation.expired`. Queue renumbering does
not emit webhook intent. Hook configuration and reservation fields are
snapshotted; the existing `resource_id`, `event`, `data`, and `reservation`
payload and typed template fields remain supported. Named secrets referenced
by templates, URLs, or headers resolve at delivery time. New signing secrets
remain encrypted in the stored hook snapshot with the application key.

Each server runs four delivery workers with no in-memory job backlog. Claims
use `FOR UPDATE SKIP LOCKED`, a fresh UUID fence, and a 30 second lease. Network
requests have a total 10 second deadline; DB operations have five second
budgets. An expired lease becomes claimable after a crash or restart. Ack and
retry updates require the current, unexpired fence. A stale worker cannot
acknowledge, reschedule, or mark another worker's delivery dead.

`X-claimctl-Delivery-ID` is the stable outbox UUID on every attempt. Existing
HMAC body signatures remain supported. Delivery is **at least once**, not
exactly once: an accepted request followed by a crash or lost acknowledgement
can be sent again. Consumers must deduplicate by delivery ID. Concurrent
workers do not promise event ordering. Fencing prevents stale DB writes, not
an already transmitted external request.

Only network failures, 429, and 5xx responses retry. Backoff is persisted as
2, 4, 8, 16, 32, 64, and 128 seconds. Eight claimed attempts exhaust the job;
crashes can consume attempts even if no request was sent. Other non-2xx
responses and invalid payload/configuration become dead immediately. Recovery
of an exhausted lease marks it dead without a ninth network request. There is
no automatic dead-letter replay. Response bodies are bounded and discarded;
URLs, payloads, resolved headers, and secrets are not in delivery diagnostics.

`GET /api/webhooks/:id/deliveries` returns the newest 100 delivery summaries:
ID, webhook/resource IDs, event, state, attempts, timestamps, last HTTP status,
and a sanitized error category. `GET /api/webhooks/deliveries/dead` lists the
newest 100 dead letters globally, including deliveries for deleted hooks.
Current admin authority is required; bearer
tokens additionally require `webhooks:write`. Space-restricted tokens cannot
read this global endpoint. Hook snapshots and signing secrets are never part
of this response. Delivered rows older than `DB_CLEANUP_DAYS` are cleaned up;
pending and dead rows remain available for operator investigation.

Expiry takes a per-resource transaction advisory lock before reclamation and
promotion. Cleanup uses a transaction advisory lock and a 30 second budget.
Connection loss releases these locks; another replica can recover. Health
workers use per-resource 60 second UUID-fenced leases, a total 45 second
budget, and transactional fence validation before publishing a health result.
Completion holds off the next check for the configured interval. Stale probes
cannot publish health status. The health queue and worker settings are clamped
to 1..1000 and 1..64 respectively. Scheduling/expiry scans are capped at 1000.
Existing constrained health egress and authorization are unchanged.

Reservation notifications run synchronously **after commit**, with a detached
real context and an aggregate 10 second budget. Dispatch no longer creates
unbounded background goroutines or retains Fiber request contexts. SMTP dial,
connection operations, and cancellation are bounded; STARTTLS still verifies
the configured server name. Notifications remain best effort, not durable.
They can add response latency; failure does not fail a committed reservation
or trigger duplicate webhook intent.

## Backup and restore

Full PostgreSQL backups include outbox and worker lease tables. The offline
[recovery procedure](BACKUP_RESTORE.md) clears them explicitly after a full
restore, before a fresh deployment starts. Native full restore installs
triggers after loading data, so historical reservations do not generate new
webhook jobs during import. Ordinary startup never clears durable jobs.

Recovery cleanup intentionally loses undelivered events to avoid replaying
side effects completed after the backup. Stop all replicas and workers before
restore: fencing cannot recall a network request already in flight. Do not
reuse pre-recovery claim or bundle idempotency keys.
Migration 24 down removes runtime jobs and coordination, preserving reservation
and history data and the corrected history behavior from migration 23. Up does
not backfill historical webhook events. Back up business data before downgrade
and expect loss of undelivered runtime jobs.
