package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

type passwordChangingRotation struct {
	services.RefreshTokenService
	change func() error
}

func (s *passwordChangingRotation) Rotate(ctx context.Context, raw string) (string, *db.ClaimctlUser, error) {
	next, user, err := s.RefreshTokenService.Rotate(ctx, raw)
	if err != nil {
		return "", nil, err
	}
	if err := s.change(); err != nil {
		return "", nil, err
	}
	return next, user, nil
}

func TestRefreshHandler_PasswordChangeAfterRotateBeforeSigningRealDatabase(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	id := uuid.New()
	password, err := bcrypt.GenerateFromPassword([]byte("OldPass1!"), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.users (id,email,name,password) VALUES ($1,$2,'Snapshot Test',$3)", id, id.String()+"@example.invalid", string(password))
	require.NoError(t, err)
	tokens := services.NewRefreshTokenService(store)
	original, err := tokens.Issue(ctx, id, services.CredentialVersion(string(password)))
	require.NoError(t, err)
	users := services.NewUserService(store)
	interleaved := &passwordChangingRotation{RefreshTokenService: tokens, change: func() error { return users.UpdatePassword(ctx, id, "OldPass1!", "NewPass2!") }}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	handler := handlers.NewUserHandler(users, nil, key, nil, interleaved)
	app := fiber.New()
	app.Post("/auth/refresh", handler.RefreshToken)
	app.Get("/protected", CookieAuthentication(store, &key.PublicKey), func(c *fiber.Ctx) error { return c.SendStatus(200) })
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: original})
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	var access, refresh *http.Cookie
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case "access_token":
			access = cookie
		case "refresh_token":
			refresh = cookie
		}
	}
	require.NotNil(t, access)
	require.NotNil(t, refresh)
	digest := sha256.Sum256([]byte(refresh.Value))
	record, err := store.GetRefreshTokenByHash(ctx, hex.EncodeToString(digest[:]))
	require.NoError(t, err)
	require.True(t, record.Revoked)
	req = httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(access)
	resp, err = app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 401, resp.StatusCode, "signing must use the locked old snapshot, never the newer password hash")
}
