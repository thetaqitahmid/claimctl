# Authentication Guide

claimctl supports three modes of authentication:

1. **Local Authentication** (Database-backed users)
2. **LDAP Authentication** (Enterprise directory integration)
3. **OpenID Connect (OIDC)** (Single Sign-On)

## Authentication & Authorization

claimctl supports three methods of authentication: Standard, LDAP, and
OpenID Connect (OIDC).

### 1. Standard Email/Password

User credentials are stored locally in the PostgreSQL database with bcrypt
hashing.

- **Registration**: Users can be created by Admins via the User Management
  panel.
- **Login**: `/api/login` verifies email/password against the `users` table.
- **Session**: Returns a JWT token set as an HTTP-only cookie.

### 2. LDAP (Lightweight Directory Access Protocol)

Integration with enterprise directory services (e.g., Active Directory,
OpenLDAP).

- **Configuration**: Set `LDAP_URL`, `LDAP_BIND_DN`, `LDAP_BIND_PASSWORD`,
  `LDAP_BASE_DN` in environment variables.
- **Flow**:
  1. User enters LDAP credentials in the login form.
  2. Backend binds to the LDAP server with service credentials.
  3. Backend searches for the user by `mail` or `uid`.
  4. Backend attempts to bind as the user to verify password.
  5. On success, user profile is synced to local `users` table and a session is
     created.
- **Role Mapping**: Admins can be mapped via `LDAP_ADMIN_GROUP_DN`.

### 3. OpenID Connect (OIDC)

Single Sign-On (SSO) using modern providers (e.g., Google, GitLab, Keycloak).

- **Configuration**:
  - `OIDC_ISSUER`: The URL of the specific provider.
  - `OIDC_CLIENT_ID`: Public identifier for the app.
  - `OIDC_CLIENT_SECRET`: Secret known only to the app and provider.
  - `OIDC_REDIRECT_URL`: (Optional) The callback URL. If not provided, it is
    automatically derived as `{BASE_URL}/api/auth/oidc/callback`.
  - `FRONTEND_URL`: URL to redirect the user after successful login (useful for
    dev vs prod).
- **Flow (with PKCE)**:
  1. User clicks "Sign in with SSO".
  2. Backend generates a `code_verifier` and `code_challenge`. Stores verifier
     in HTTP-only cookie.
  3. Redirects to Provider with `code_challenge`.
  4. Provider redirects back to `/api/auth/oidc/callback` with an auth code.
  5. Backend sends auth code + `code_verifier` to Provider.
  6. Provider validates `SHA256(verifier) == challenge` and returns tokens.
  7. Backend syncs user and redirects to `FRONTEND_URL`.
