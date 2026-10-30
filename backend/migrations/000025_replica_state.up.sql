CREATE TABLE claimctl.realtime_events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    resource_id UUID,
    space_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX realtime_events_created_at ON claimctl.realtime_events(created_at);
CREATE TABLE claimctl.shared_storage (
    key TEXT PRIMARY KEY,
    value BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE claimctl.rate_limits (
    key TEXT PRIMARY KEY,
    count BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE FUNCTION claimctl.lock_realtime_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM set_config('lock_timeout', '5s', true);
    PERFORM pg_advisory_xact_lock(725025);
    RETURN NULL;
END;
$$;

CREATE FUNCTION claimctl.capture_realtime_event() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    rid UUID;
    sid UUID;
    kind TEXT := 'resync';
BEGIN
    IF current_setting('claimctl.restoring', true) = 'on' THEN
        RETURN NULL;
    END IF;
    PERFORM pg_advisory_xact_lock(725025);
    IF TG_TABLE_NAME = 'api_tokens' AND TG_OP = 'UPDATE' THEN
        IF ROW(OLD.id, OLD.user_id, OLD.token_hash, OLD.scopes,
               OLD.space_ids, OLD.expires_at) IS NOT DISTINCT FROM
           ROW(NEW.id, NEW.user_id, NEW.token_hash, NEW.scopes,
               NEW.space_ids, NEW.expires_at) THEN
            RETURN NULL;
        END IF;
    ELSIF TG_TABLE_NAME = 'users' AND TG_OP = 'UPDATE' THEN
        IF ROW(OLD.id, OLD.password, OLD.status, OLD.role) IS NOT DISTINCT FROM
           ROW(NEW.id, NEW.password, NEW.status, NEW.role) THEN
            RETURN NULL;
        END IF;
    END IF;
    IF TG_TABLE_NAME = 'resources' THEN
        IF TG_OP = 'DELETE' THEN
            rid := OLD.id; sid := OLD.space_id;
        ELSE
            rid := NEW.id; sid := NEW.space_id;
        END IF;
        kind := 'maintenance_change';
        IF TG_OP = 'UPDATE' AND OLD.space_id IS DISTINCT FROM NEW.space_id THEN
            rid := NULL; sid := NULL; kind := 'resync';
        END IF;
    ELSIF TG_TABLE_NAME = 'reservations' THEN
        IF TG_OP = 'DELETE' THEN rid := OLD.resource_id;
        ELSE rid := NEW.resource_id;
        END IF;
        SELECT space_id INTO sid FROM claimctl.resources WHERE id = rid;
        kind := 'reservation_update';
        IF sid IS NULL THEN rid := NULL; kind := 'resync'; END IF;
    END IF;
    INSERT INTO claimctl.realtime_events(event_type, resource_id, space_id)
    VALUES (kind, rid, sid);
    RETURN NULL;
END;
$$;

DO $$
DECLARE
    tbl TEXT;
    events TEXT;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['resources', 'reservations', 'users',
        'spaces', 'groups', 'group_members', 'space_permissions', 'api_tokens']
    LOOP
        events := 'INSERT OR UPDATE OR DELETE';
        IF tbl = 'api_tokens' THEN
            events := 'INSERT OR DELETE OR UPDATE OF id, user_id, token_hash, scopes, space_ids, expires_at';
        ELSIF tbl = 'users' THEN
            events := 'INSERT OR DELETE OR UPDATE OF id, password, status, role';
        END IF;
        EXECUTE format('CREATE TRIGGER realtime_lock BEFORE %s ON claimctl.%I FOR EACH STATEMENT EXECUTE FUNCTION claimctl.lock_realtime_mutation()', events, tbl);
        EXECUTE format('CREATE TRIGGER realtime_capture AFTER %s ON claimctl.%I FOR EACH ROW EXECUTE FUNCTION claimctl.capture_realtime_event()', events, tbl);
    END LOOP;
END;
$$;

CREATE FUNCTION claimctl.lock_realtime_insert() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM set_config('lock_timeout', '5s', true);
    PERFORM pg_advisory_xact_lock(725025);
    RETURN NULL;
END;
$$;
CREATE TRIGGER realtime_insert_lock BEFORE INSERT ON claimctl.realtime_events
FOR EACH STATEMENT EXECUTE FUNCTION claimctl.lock_realtime_insert();

CREATE FUNCTION claimctl.prune_realtime_events() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.id % 100 = 0 THEN
        DELETE FROM claimctl.realtime_events WHERE id < NEW.id - 10000;
    END IF;
    RETURN NULL;
END;
$$;
CREATE TRIGGER realtime_prune AFTER INSERT ON claimctl.realtime_events
FOR EACH ROW EXECUTE FUNCTION claimctl.prune_realtime_events();
