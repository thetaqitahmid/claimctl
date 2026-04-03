-- Insert sample users
INSERT INTO claimctl.users (email, name, password, role) VALUES
('alice.johnson@company.com', 'Alice Johnson', '$2a$10$placeholder_hash', 'user'),
('bob.smith@company.com', 'Bob Smith', '$2a$10$placeholder_hash', 'user'),
('charlie.brown@company.com', 'Charlie Brown', '$2a$10$placeholder_hash', 'user'),
('david.wilson@company.com', 'David Wilson', '$2a$10$placeholder_hash', 'user'),
('eva.davis@company.com', 'Eva Davis', '$2a$10$placeholder_hash', 'user'),
('frank.miller@company.com', 'Frank Miller', '$2a$10$placeholder_hash', 'user'),
('grace.lee@company.com', 'Grace Lee', '$2a$10$placeholder_hash', 'user'),
('hannah.clark@company.com', 'Hannah Clark', '$2a$10$placeholder_hash', 'user'),
('isaac.martinez@company.com', 'Isaac Martinez', '$2a$10$placeholder_hash', 'user'),
('jamie.anderson@company.com', 'Jamie Anderson', '$2a$10$placeholder_hash', 'user');

-- Insert sample spaces
-- Note: We assume clean slate.

-- We will insert more spaces.
INSERT INTO claimctl.spaces (name, description) VALUES
('Default Space', 'The default space for resources.'),
('Development Lab', 'Resources for development purposes'),
('QA Environment', 'Dedicated for QA testing'),
('Production Staging', 'Pre-production environment');

-- Insert sample resources
-- We assign them to spaces using subqueries for UUID lookups.

INSERT INTO claimctl.resources (name, type, labels, space_id) VALUES
('matlab license 14', 'license', '["label 1", "label 2"]', (SELECT id FROM claimctl.spaces WHERE name = 'Development Lab')),
('development tool 69', 'server', '["high priority", "low priority"]', (SELECT id FROM claimctl.spaces WHERE name = 'Development Lab')),
('development tool 56', 'security', '["label 1", "label 2"]', (SELECT id FROM claimctl.spaces WHERE name = 'Development Lab')),
('lab server 79', 'license', '["label 1", "label 2"]', (SELECT id FROM claimctl.spaces WHERE name = 'QA Environment')),
('lab server 35', 'cloud', '["alpha", "beta", "gamma"]', (SELECT id FROM claimctl.spaces WHERE name = 'QA Environment')),
('database instance 19', 'database', '["very long label", "short label", "tiny lbl"]', (SELECT id FROM claimctl.spaces WHERE name = 'QA Environment')),
('lab server 47', 'environment', '["very long label", "short label", "tiny lbl"]', (SELECT id FROM claimctl.spaces WHERE name = 'Production Staging')),
('virtual machine 3', 'security', '["very long label", "short label", "tiny lbl"]', (SELECT id FROM claimctl.spaces WHERE name = 'Production Staging')),
('network ip 13', 'tool', '["alpha", "beta", "gamma"]', (SELECT id FROM claimctl.spaces WHERE name = 'Default Space')),
('testing environment 7', 'cloud', '["test", "dev", "prod"]', (SELECT id FROM claimctl.spaces WHERE name = 'Default Space')),
('virtual machine 78', 'virtual machine', '["high priority", "low priority"]', (SELECT id FROM claimctl.spaces WHERE name = 'Development Lab')),
('testing environment 21', 'virtual machine', '["label 1", "label 2"]', (SELECT id FROM claimctl.spaces WHERE name = 'QA Environment')),
('matlab license 68', 'environment', '["test", "dev", "prod"]', (SELECT id FROM claimctl.spaces WHERE name = 'Production Staging'));

-- Insert sample reservations
INSERT INTO claimctl.reservations (resource_id, user_id, status, queue_position, start_time, end_time) VALUES
((SELECT id FROM claimctl.resources WHERE name = 'matlab license 14'), (SELECT id FROM claimctl.users WHERE email = 'alice.johnson@company.com'), 'active', NULL, EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT, NULL),
((SELECT id FROM claimctl.resources WHERE name = 'matlab license 14'), (SELECT id FROM claimctl.users WHERE email = 'bob.smith@company.com'), 'pending', 1, NULL, NULL),
((SELECT id FROM claimctl.resources WHERE name = 'matlab license 14'), (SELECT id FROM claimctl.users WHERE email = 'charlie.brown@company.com'), 'pending', 2, NULL, NULL),
((SELECT id FROM claimctl.resources WHERE name = 'lab server 79'), (SELECT id FROM claimctl.users WHERE email = 'david.wilson@company.com'), 'active', NULL, EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - INTERVAL '1 hour')::BIGINT, NULL),
((SELECT id FROM claimctl.resources WHERE name = 'lab server 79'), (SELECT id FROM claimctl.users WHERE email = 'eva.davis@company.com'), 'pending', 1, NULL, NULL),
((SELECT id FROM claimctl.resources WHERE name = 'database instance 19'), (SELECT id FROM claimctl.users WHERE email = 'frank.miller@company.com'), 'completed', NULL, EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - INTERVAL '2 days')::BIGINT, EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - INTERVAL '1 day')::BIGINT),
((SELECT id FROM claimctl.resources WHERE name = 'lab server 47'), (SELECT id FROM claimctl.users WHERE email = 'grace.lee@company.com'), 'active', NULL, EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - INTERVAL '30 minutes')::BIGINT, NULL);

-- Insert sample secrets
INSERT INTO claimctl.secrets (key, value, description) VALUES
('SLACK_WEBHOOK_URL', 'https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX', 'Webhook URL for Slack notifications'),
('CI_PIPELINE_TOKEN', 'glpat-xxxxxxxxxxxxxxxxxxxx', 'GitLab CI Pipeline Token');

-- Insert sample webhooks
INSERT INTO claimctl.webhooks (name, url, method, headers, template, description) VALUES
('Slack Notification', '{{SLACK_WEBHOOK_URL}}', 'POST', '{"Content-Type": "application/json"}', '{"text": "Resource {{resource.name}} status changed to {{status}}"}', 'Notify Slack channel on resource events'),
('CI Pipeline Trigger', 'https://gitlab.com/api/v4/projects/1/trigger/pipeline', 'POST', '{"PRIVATE-TOKEN": "{{CI_PIPELINE_TOKEN}}"}', '{"ref": "master", "variables": {"RESOURCE_ID": "{{resource.id}}"}}', 'Trigger CI pipeline on resource updates');

-- Insert sample resource webhooks
INSERT INTO claimctl.resource_webhooks (resource_id, webhook_id, events) VALUES
((SELECT id FROM claimctl.resources WHERE name = 'matlab license 14'), (SELECT id FROM claimctl.webhooks WHERE name = 'Slack Notification'), ARRAY['reservation.created', 'reservation.cancelled']),
((SELECT id FROM claimctl.resources WHERE name = 'development tool 69'), (SELECT id FROM claimctl.webhooks WHERE name = 'CI Pipeline Trigger'), ARRAY['reservation.created']);

-- Insert sample health check resources
INSERT INTO claimctl.resources (name, type, labels, space_id) VALUES
('Google DNS', 'server', '["health-check", "external"]', (SELECT id FROM claimctl.spaces WHERE name = 'Default Space')),
('Cloudflare DNS', 'server', '["health-check", "external"]', (SELECT id FROM claimctl.spaces WHERE name = 'Default Space')),
('Example HTTP', 'website', '["health-check", "external"]', (SELECT id FROM claimctl.spaces WHERE name = 'Default Space'));

-- Insert sample health check configurations
INSERT INTO claimctl.resource_health_configs (resource_id, enabled, check_type, target, interval_seconds, timeout_seconds, retry_count) VALUES
((SELECT id FROM claimctl.resources WHERE name = 'Google DNS'), true, 'ping', '8.8.8.8', 60, 5, 3),
((SELECT id FROM claimctl.resources WHERE name = 'Cloudflare DNS'), true, 'ping', '1.1.1.1', 60, 5, 3),
((SELECT id FROM claimctl.resources WHERE name = 'Example HTTP'), true, 'http', 'https://example.com', 300, 10, 3);