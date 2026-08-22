DELETE FROM app_settings WHERE key IN (
    'oidc_redirect_url',
    'oidc_admin_group',
    'oidc_groups_claim'
);

DROP INDEX IF EXISTS claimctl.users_oidc_subject_uidx;

ALTER TABLE claimctl.users DROP COLUMN IF EXISTS oidc_subject;

-- Cannot reliably restore which users were previously ldap; settings only.
INSERT INTO app_settings (key, value, category, description, is_secret) VALUES
    ('ldap_url', '', 'auth', 'LDAP Server URL (ldap://...)', FALSE),
    ('ldap_bind_dn', '', 'auth', 'LDAP Bind DN', FALSE),
    ('ldap_bind_password', '', 'auth', 'LDAP Bind Password', TRUE),
    ('ldap_base_dn', '', 'auth', 'LDAP Base DN', FALSE)
ON CONFLICT (key) DO NOTHING;
