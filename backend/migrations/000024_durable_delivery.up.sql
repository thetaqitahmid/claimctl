CREATE TABLE claimctl.webhook_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL,
    resource_id UUID NOT NULL,
    event TEXT NOT NULL,
    hook JSONB NOT NULL,
    reservation JSONB NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','delivering','delivered','dead')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 8),
    available_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    lease_until TIMESTAMPTZ,
    fence UUID,
    last_status INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK ((state = 'delivering' AND fence IS NOT NULL AND lease_until IS NOT NULL)
        OR (state <> 'delivering' AND fence IS NULL AND lease_until IS NULL))
);
CREATE INDEX webhook_outbox_ready ON claimctl.webhook_outbox (available_at, created_at)
    WHERE state IN ('pending','delivering');
CREATE INDEX webhook_outbox_hook ON claimctl.webhook_outbox (webhook_id, created_at DESC);
CREATE INDEX webhook_outbox_dead ON claimctl.webhook_outbox (created_at DESC)
    WHERE state = 'dead';
CREATE TABLE claimctl.worker_leases (
    name TEXT PRIMARY KEY,
    fence UUID NOT NULL,
    lease_until TIMESTAMPTZ NOT NULL
);
CREATE FUNCTION claimctl.enqueue_reservation_webhooks() RETURNS TRIGGER AS $$
DECLARE
    event_name TEXT;
    snapshot claimctl.reservations;
BEGIN
    snapshot := NEW;
    IF TG_OP = 'INSERT' THEN
        SELECT * INTO snapshot FROM claimctl.reservations WHERE id = NEW.id;
        IF NOT FOUND THEN RETURN NULL; END IF;
        event_name := 'reservation.created';
    ELSIF NEW.status IS DISTINCT FROM OLD.status THEN
        event_name := 'reservation.' || CASE
            WHEN NEW.status IN ('completed', 'cancelled') AND (
                OLD.lease_expires_at <= clock_timestamp() OR
                OLD.scheduled_end_time <= clock_timestamp() OR
                OLD.hard_end_time <= clock_timestamp()) THEN 'expired'
            WHEN NEW.status = 'active' THEN 'activated'
            ELSE NEW.status END;
    ELSIF OLD.lease_expires_at IS NOT NULL AND
        NEW.lease_expires_at IS DISTINCT FROM OLD.lease_expires_at THEN
        event_name := 'reservation.renewed';
    ELSE RETURN NEW;
    END IF;
    INSERT INTO claimctl.webhook_outbox (webhook_id, resource_id, event, hook, reservation)
    SELECT w.id, NEW.resource_id, event_name, to_jsonb(w), to_jsonb(snapshot)
    FROM claimctl.webhooks w JOIN claimctl.resource_webhooks rw ON rw.webhook_id = w.id
    WHERE rw.resource_id = NEW.resource_id AND event_name = ANY(rw.events);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE CONSTRAINT TRIGGER reservation_webhook_intent AFTER INSERT OR UPDATE ON claimctl.reservations
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION claimctl.enqueue_reservation_webhooks();
