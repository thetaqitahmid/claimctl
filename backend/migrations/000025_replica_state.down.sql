DO $$
DECLARE tbl TEXT;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['resources', 'reservations', 'users',
        'spaces', 'groups', 'group_members', 'space_permissions', 'api_tokens']
    LOOP
        EXECUTE format('DROP TRIGGER realtime_capture ON claimctl.%I', tbl);
        EXECUTE format('DROP TRIGGER realtime_lock ON claimctl.%I', tbl);
    END LOOP;
END;
$$;
DROP TABLE claimctl.realtime_events;
DROP TABLE claimctl.shared_storage;
DROP TABLE claimctl.rate_limits;
DROP FUNCTION claimctl.prune_realtime_events();
DROP FUNCTION claimctl.lock_realtime_insert();
DROP FUNCTION claimctl.capture_realtime_event();
DROP FUNCTION claimctl.lock_realtime_mutation();
