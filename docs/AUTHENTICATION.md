# Authentication Guide

claimctl supports two modes of authentication:

1. **Local Authentication** (Database-backed users)
2. **OpenID Connect (OIDC)** (Single Sign-On)

## Authentication & Authorization

### 1. Standard Email/Password

User credentials are stored locally in the PostgreSQL database with bcrypt
hashing.

- **Registration**: Users can be created by Admins via the User Management
  panel.
- **Login**: `/api/login` verifies email/password against the `users` table.
  Only users with `auth_provider=local` may use password login.
- **Session**: Returns a JWT access token and rotating refresh token as
  HTTP-only cookies.
- **Lockout**: After 3 failed attempts the account is locked for 30 minutes.

### 2. OpenID Connect (OIDC)

Single Sign-On (SSO) using modern providers (e.g. Google, GitLab, Keycloak,
Entra ID).

- **Configuration** (environment variables sync into `app_settings` on
  startup when DB values are empty; also editable in Admin → Auth settings):
  - `OIDC_ISSUER`: Issuer URL of the provider.
  - `OIDC_CLIENT_ID`: Public client identifier.
  - `OIDC_CLIENT_SECRET`: Client secret (stored encrypted when marked secret).
  - `OIDC_REDIRECT_URL`: (Optional) Callback URL. If empty, derived as
    `{request base}/api/auth/oidc/callback`.
  - `OIDC_SCOPES`: (Optional) Space-separated scopes; default
    `openid profile email`.
  - `OIDC_ADMIN_GROUP`: (Optional) Group name that grants the `admin` role
    (promote-only; existing admins are not demoted when the claim is absent).
  - `OIDC_GROUPS_CLAIM`: (Optional) Claim name for groups; default `groups`.
  - `FRONTEND_URL`: Redirect target after successful login.

- **Discovery**: `GET /api/auth/methods` returns
  `{ "local": true, "oidc": true|false }` so the login UI can hide SSO when
  OIDC is not configured.

- **Flow (with PKCE)**:
  1. User clicks "Sign in with SSO".
  2. Backend generates a `code_verifier` and `code_challenge`. Stores verifier
     and CSRF `state` in HTTP-only cookies (`SameSite=Lax`).
  3. Redirects to the provider with `code_challenge`.
  4. Provider redirects to `/api/auth/oidc/callback` with an auth code.
  5. Backend exchanges the code + verifier, then **verifies the ID token**
     (signature, audience, issuer).
  6. Requires an `email` claim. If `email_verified` is present it must be
     `true`.
  7. Links the user by stable OIDC `sub` (`oidc_subject`). First login may
     bind an existing `auth_provider=oidc` row by email; a local account with
     the same email is **not** taken over.
  8. Issues session cookies and redirects to `FRONTEND_URL`.

- **Password changes**: Not available for OIDC users (managed at the IdP).

- **Admin role mapping**: Matching `OIDC_ADMIN_GROUP` **promotes** a user to
  `admin` only. Removing the group membership at the IdP does **not** demote
  them; a local admin must change the role in User Management.

### Migrating former LDAP users

LDAP authentication has been removed. On upgrade, migration `000020` sets
`auth_provider='local'` for any remaining LDAP users. Their old dummy passwords
are not usable. An admin must reset each user's password (or delete and
re-create the account, or have them use OIDC with a different email if linking
is required).
