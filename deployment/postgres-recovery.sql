BEGIN;
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(725025);
TRUNCATE TABLE
    claimctl.refresh_tokens,
    claimctl.claim_idempotency,
    claimctl.bundle_idempotency,
    claimctl.webhook_outbox,
    claimctl.worker_leases,
    claimctl.realtime_events,
    claimctl.shared_storage,
    claimctl.rate_limits;
COMMIT;
