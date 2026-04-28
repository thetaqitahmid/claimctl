# Todo

## Features

### Refresh Token + Access Token Authentication

Replace the current single long-lived JWT (24h) with a short-lived access token
(15min) and a long-lived refresh token (7d). Both stored as HTTP-only cookies.

**Strategy**

- Access token: JWT RS256, 15min expiry, `access_token` HTTP-only cookie
- Refresh token: opaque random token, 7d expiry, `refresh_token` HTTP-only
  cookie, stored as SHA-256 hash in DB with family-based theft detection

**Tasks**

- [ ] **Task 1 — DB migration**
  - Create `migrations/000018_refresh_tokens.up.sql` with table
    `claimctl.refresh_tokens(id, user_id, token_hash UNIQUE, family_id,
    expires_at, revoked, created_at)`
  - Add sqlc queries: `CreateRefreshToken`, `GetRefreshTokenByHash`,
    `RevokeRefreshTokenFamily`, `DeleteRefreshToken`,
    `DeleteExpiredRefreshTokens`
  - Run `sqlc generate`

- [ ] **Task 2 — RefreshTokenService**
  - Create `internal/services/refresh_token_service.go`
  - Interface: `Issue(ctx, userID)`, `Rotate(ctx, rawToken)`, `Revoke(ctx,
    rawToken)`
  - Issue: 32 random bytes, `"rt_"` prefix, SHA-256 hash stored, new
    `family_id`, 7d expiry
  - Rotate: lookup hash → if revoked, revoke entire family + return error
    (theft detection); else delete old row, insert new row with same `family_id`
  - Add unit tests in `refresh_token_service_test.go`

- [ ] **Task 3 — Shorten access token + cookie helper**
  - Change `generateJWT` expiry from 24h to 15min in `user_handlers.go`
  - Add `setAuthCookies(c, jwtToken, refreshToken string)` helper (HTTPOnly,
    Secure, SameSite=Strict)
  - Inject `RefreshTokenService` into `UserHandler`; update `NewUserHandler`
    and its call site in `routes.go`

- [ ] **Task 4 — Update login/logout handlers**
  - Login, LoginLDAP, CallbackOIDC: call `Issue()` + `setAuthCookies()`
  - Logout: read `refresh_token` cookie → `Revoke()` → clear both cookies

- [ ] **Task 5 — `/auth/refresh` endpoint**
  - Add `RefreshToken` handler: read cookie → `Rotate()` → fetch user →
    `generateJWT` → `setAuthCookies` → 200
  - Register `POST /auth/refresh` in `routes.go` before jwtware middleware

- [ ] **Task 6 — Cleanup worker**
  - Add `db.DeleteExpiredRefreshTokens(ctx)` to the existing cleanup loop in
    `cleanup_worker.go`

- [ ] **Task 7 — Frontend silent refresh**
  - Create `frontend/src/store/api/baseQuery.ts` with `baseQueryWithReauth`:
    on 401 → `POST /auth/refresh`; if refresh fails dispatch
    `clearCredentials()`; else retry original request once
  - Swap all RTK Query API slices from `fetchBaseQuery` to
    `baseQueryWithReauth`

- [ ] **Task 8 — Verify end-to-end**
  - Confirm `AuthProvider.tsx` and `ProtectedRoute` need no structural changes
  - Smoke test: login → wait 15min → verify silent refresh → logout → verify
    refresh token is revoked

**Files**

| Action | Path |
|--------|------|
| CREATE | `migrations/000018_refresh_tokens.{up,down}.sql` |
| CREATE | `sql/queries/refresh_tokens.sql` |
| CREATE | `internal/services/refresh_token_service.go` + `_test.go` |
| CREATE | `frontend/src/store/api/baseQuery.ts` |
| MODIFY | `internal/handlers/user_handlers.go` |
| MODIFY | `internal/server/routes.go` |
| MODIFY | `internal/workers/cleanup_worker.go` |
| MODIFY | `frontend/src/store/api/*.ts` (all API slices) |

**Notes**

- Backward-compatible: existing 24h JWTs expire naturally within 24h of deploy,
  no forced logout required
- Theft detection via token family: reuse of a rotated refresh token revokes
  the entire family
