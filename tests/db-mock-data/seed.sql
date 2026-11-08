\set ON_ERROR_STOP on

BEGIN;

DO $$
BEGIN
    IF to_regclass('claimctl.spaces') IS NULL
       OR to_regclass('claimctl.resources') IS NULL THEN
        RAISE EXCEPTION 'Start make dev_up first to apply database migrations';
    END IF;
END;
$$;

SELECT pg_advisory_xact_lock(hashtext('claimctl.development_seed'));

INSERT INTO claimctl.spaces (name, description) VALUES
    ('Default Space', 'The default space for resources.'),
    ('Development Lab', 'Resources for development purposes'),
    ('QA Environment', 'Dedicated for QA testing'),
    ('Production Staging', 'Pre-production environment')
ON CONFLICT (name) DO NOTHING;

INSERT INTO claimctl.resources (name, type, labels, space_id)
SELECT sample.name, sample.type, sample.labels::jsonb, space.id
FROM (VALUES
    ('matlab license 14', 'license', '["label 1", "label 2"]', 'Development Lab'),
    ('development tool 69', 'server', '["high priority", "low priority"]', 'Development Lab'),
    ('development tool 56', 'security', '["label 1", "label 2"]', 'Development Lab'),
    ('lab server 79', 'license', '["label 1", "label 2"]', 'QA Environment'),
    ('lab server 35', 'cloud', '["alpha", "beta", "gamma"]', 'QA Environment'),
    ('database instance 19', 'database', '["very long label", "short label", "tiny lbl"]', 'QA Environment'),
    ('lab server 47', 'environment', '["very long label", "short label", "tiny lbl"]', 'Production Staging'),
    ('virtual machine 3', 'security', '["very long label", "short label", "tiny lbl"]', 'Production Staging'),
    ('network ip 13', 'tool', '["alpha", "beta", "gamma"]', 'Default Space'),
    ('testing environment 7', 'cloud', '["test", "dev", "prod"]', 'Default Space'),
    ('virtual machine 78', 'virtual machine', '["high priority", "low priority"]', 'Development Lab'),
    ('testing environment 21', 'virtual machine', '["label 1", "label 2"]', 'QA Environment'),
    ('matlab license 68', 'environment', '["test", "dev", "prod"]', 'Production Staging'),
    ('Google DNS', 'server', '["health-check", "external"]', 'Default Space'),
    ('Cloudflare DNS', 'server', '["health-check", "external"]', 'Default Space'),
    ('Example HTTP', 'website', '["health-check", "external"]', 'Default Space')
) AS sample(name, type, labels, space_name)
JOIN claimctl.spaces AS space ON space.name = sample.space_name
ON CONFLICT (name) DO NOTHING;

COMMIT;
