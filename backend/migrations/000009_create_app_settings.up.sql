CREATE TABLE IF NOT EXISTS app_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    description TEXT,
    is_secret BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO app_settings (key, value, category, description, is_secret) VALUES
('slack_bot_token', '', 'notification', 'Bot User OAuth Token for Slack', TRUE),
('smtp_host', '', 'notification', 'SMTP Server Host', FALSE),
('smtp_port', '587', 'notification', 'SMTP Server Port', FALSE),
('smtp_user', '', 'notification', 'SMTP Username', FALSE),
('smtp_pass', '', 'notification', 'SMTP Password', TRUE),
('smtp_from', 'noreply@github.com/thetaqitahmid/claimctl', 'notification', 'Email Sender Address', FALSE),
('ldap_url', '', 'auth', 'LDAP Server URL (ldap://...)', FALSE),
('ldap_bind_dn', '', 'auth', 'LDAP Bind DN', FALSE),
('ldap_bind_password', '', 'auth', 'LDAP Bind Password', TRUE),
('ldap_base_dn', '', 'auth', 'LDAP Base DN', FALSE),
('oidc_issuer', '', 'auth', 'OIDC Issuer URL', FALSE),
('oidc_client_id', '', 'auth', 'OIDC Client ID', FALSE),
('oidc_client_secret', '', 'auth', 'OIDC Client Secret', TRUE),
('oidc_scopes', 'openid profile email', 'auth', 'OIDC Scopes (space separated)', FALSE),
('jwt_private_key', '', 'auth', 'JWT Private Key', TRUE),
('jwt_public_key', '', 'auth', 'JWT Public Key', TRUE)
ON CONFLICT (key) DO NOTHING;
