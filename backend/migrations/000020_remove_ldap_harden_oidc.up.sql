-- Remove LDAP settings (feature removed)
DELETE FROM app_settings WHERE key LIKE 'ldap_%';

-- Former LDAP users keep their rows but can no longer bind via LDAP.
-- Convert to local so an admin can set a password; dummy LDAP hashes remain
-- until an admin resets the password via User Management.
UPDATE claimctl.users
SET auth_provider = 'local',
    updated_at = CURRENT_TIMESTAMP
WHERE auth_provider = 'ldap';

-- OIDC subject for stable identity linking
ALTER TABLE claimctl.users
    ADD COLUMN IF NOT EXISTS oidc_subject VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS users_oidc_subject_uidx
    ON claimctl.users (oidc_subject)
    WHERE oidc_subject IS NOT NULL;

-- Additional OIDC settings
INSERT INTO app_settings (key, value, category, description, is_secret) VALUES
    ('oidc_redirect_url', '', 'auth', 'OIDC Redirect URL (optional; defaults to {BASE}/api/auth/oidc/callback)', FALSE),
    ('oidc_admin_group', '', 'auth', 'OIDC group name that grants admin role (empty = disabled)', FALSE),
    ('oidc_groups_claim', 'groups', 'auth', 'OIDC claim name containing group memberships', FALSE)
ON CONFLICT (key) DO NOTHING;
