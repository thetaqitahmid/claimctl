package server

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func signedJWT(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	require.NoError(t, err)
	return raw
}

func TestAuthentication_CurrentAccountAndMalformedClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	id := testutils.TestUUID(1)
	for _, tc := range []struct {
		name, status, role, version string
		malformed                   bool
		expected                    int
	}{
		{name: "current admin", status: "active", role: "admin", version: "password", expected: 200},
		{name: "demoted admin", status: "active", role: "user", version: "password", expected: 403},
		{name: "inactive account", status: "inactive", role: "admin", version: "password", expected: 401},
		{name: "password revoked", status: "active", role: "admin", version: "old", expected: 401},
		{name: "old token no version", status: "active", role: "admin", expected: 401},
		{name: "malformed id", malformed: true, expected: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			q.On("FindUserById", mock.Anything, id).Return(db.ClaimctlUser{ID: id, Status: tc.status, Role: tc.role, Password: "password"}, nil).Maybe()
			app := fiber.New()
			app.Use(BearerAuthentication(services.NewAPITokenService(q), q, &key.PublicKey))
			app.Use(CookieAuthentication(q, &key.PublicKey))
			app.Get("/admin", AdminMiddleware(), func(c *fiber.Ctx) error { return c.SendStatus(200) })
			claims := jwt.MapClaims{"id": id.String(), "role": "admin", "exp": time.Now().Add(time.Hour).Unix(), "iss": "claimctl"}
			if tc.version != "" {
				claims["credential_version"] = services.CredentialVersion(tc.version)
			}
			if tc.malformed {
				claims["id"] = []string{"bad"}
			}
			req := httptest.NewRequest("GET", "/admin", nil)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: signedJWT(t, key, claims)})
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.expected, resp.StatusCode)
		})
	}
}

func TestCSRF_BrowserAndVerifiedBearer(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://frontend.example")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := &testutils.MockQuerier{}
	id := testutils.TestUUID(1)
	q.On("FindUserById", mock.Anything, id).Return(db.ClaimctlUser{ID: id, Status: "active", Role: "user", Password: "password"}, nil)
	apiToken := "res_" + strings.Repeat("a", 64)
	q.On("GetAPITokenByHash", mock.Anything, mock.Anything).Return(db.ClaimctlApiToken{ID: testutils.TestUUID(5), UserID: id, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, Scopes: []string{"read", "reserve"}}, nil)
	q.On("UpdateAPITokenLastUsed", mock.Anything, mock.Anything).Return(nil)
	app := fiber.New()
	app.Use(BearerAuthentication(services.NewAPITokenService(q), q, &key.PublicKey))
	app.Use(CSRFMiddleware(&csrfTestStorage{values: map[string][]byte{}}))
	app.Get("/api/auth/csrf", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"csrfToken": c.Locals("csrf")}) })
	for _, path := range []string{"/api/login", "/api/auth/refresh", "/api/logout", "/api/reservations"} {
		app.Post(path, func(c *fiber.Ctx) error {
			if c.Locals("verified_bearer") == true {
				assert.Empty(t, c.Cookies("access_token"))
				assert.Empty(t, c.Cookies("refresh_token"))
			}
			return c.SendStatus(200)
		})
	}
	challenge, err := app.Test(httptest.NewRequest("GET", "/api/auth/csrf", nil))
	require.NoError(t, err)
	var data struct {
		Token string `json:"csrfToken"`
	}
	require.NoError(t, json.NewDecoder(challenge.Body).Decode(&data))
	challenge.Body.Close()
	require.Len(t, data.Token, 64)
	cookie := challenge.Cookies()[0]
	require.False(t, cookie.HttpOnly)
	bearer := signedJWT(t, key, jwt.MapClaims{"id": id.String(), "exp": time.Now().Add(time.Hour).Unix(), "iss": "claimctl", "credential_version": services.CredentialVersion("password")})
	for _, path := range []string{"/api/login", "/api/auth/refresh", "/api/logout", "/api/reservations"} {
		for _, tc := range []struct {
			name, header, auth, origin string
			cookie                     bool
			expected                   int
		}{
			{name: "missing challenge", expected: 403},
			{name: "cookie only", cookie: true, expected: 403},
			{name: "matching header", cookie: true, header: data.Token, expected: 200},
			{name: "wrong header", cookie: true, header: "bad", expected: 403},
			{name: "trusted cross origin", cookie: true, header: data.Token, origin: "https://frontend.example", expected: 200},
			{name: "untrusted origin", cookie: true, header: data.Token, origin: "https://evil.example", expected: 403},
			{name: "verified bearer", auth: "Bearer " + bearer, expected: 200},
			{name: "verified API token with cookies discarded", auth: "Bearer " + apiToken, cookie: true, expected: 200},
			{name: "malformed API token no fallback", auth: "Bearer res_invalid", cookie: true, header: data.Token, expected: 401},
			{name: "invalid bearer no cookie fallback", auth: "Bearer broken", cookie: true, header: data.Token, expected: 401},
			{name: "malformed bearer", auth: "Basic bad", cookie: true, header: data.Token, expected: 401},
		} {
			t.Run(path+tc.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", path, nil)
				if tc.cookie {
					req.AddCookie(cookie)
					req.AddCookie(&http.Cookie{Name: "access_token", Value: bearer})
					req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-cookie"})
				}
				req.Header.Set("X-CSRF-Token", tc.header)
				req.Header.Set("Authorization", tc.auth)
				req.Header.Set("Origin", tc.origin)
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, tc.expected, resp.StatusCode)
				if resp.StatusCode == 403 {
					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err)
					require.Contains(t, string(body), "CSRF_INVALID")
				}
			})
		}
	}
}
