package server

import (
	"context"
	"io"
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

func TestUserReads_VerifiedBearerCurrentRoleAndSafeDTORealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	store := db.NewStore(pool)
	ctx := context.Background()
	id := uuid.New()
	password, err := bcrypt.GenerateFromPassword([]byte("SyntheticPass1!"), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.users (id,email,name,password,role) VALUES ($1,$2,'Admin Test',$3,'admin')", id, id.String()+"@example.invalid", string(password))
	require.NoError(t, err)
	tokens := services.NewAPITokenService(store)
	defaultToken, _, err := tokens.GenerateToken(ctx, id, "default", nil)
	require.NoError(t, err)
	scopedToken, _, err := tokens.GenerateToken(ctx, id, "explicit", nil, services.TokenOptions{Scopes: []string{"users:read"}})
	require.NoError(t, err)
	app := fiber.New()
	app.Use(BearerAuthentication(tokens, store, nil), CookieAuthentication(store, nil), AuthorizationMiddleware(store))
	handler := handlers.NewUserHandler(services.NewUserService(store), nil, nil, nil, nil)
	app.Get("/api/users", handler.GetUsers)
	app.Get("/api/users/find", handler.GetUser)
	for _, tc := range []struct {
		name, raw, role string
		status          int
	}{
		{"default denied", defaultToken, "admin", 403},
		{"explicit allowed", scopedToken, "admin", 200},
		{"current demotion denied", scopedToken, "user", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role=$2 WHERE id=$1", id, tc.role)
			require.NoError(t, err)
			for _, path := range []string{"/api/users", "/api/users/find?email=" + id.String() + "@example.invalid"} {
				req := httptest.NewRequest("GET", path, nil)
				req.Header.Set("Authorization", "Bearer "+tc.raw)
				resp, err := app.Test(req)
				require.NoError(t, err)
				raw, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				require.NoError(t, err)
				require.Equal(t, tc.status, resp.StatusCode)
				require.NotContains(t, string(raw), `"password"`)
				require.NotContains(t, string(raw), string(password))
				if tc.status == 200 {
					require.Contains(t, string(raw), `"authProvider":"local"`)
				}
			}
		})
	}
}
