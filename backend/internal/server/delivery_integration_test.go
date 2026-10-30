package server

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestDeliveryObservabilityScopedHTTPRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	user, hook, resource := uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'delivery','synthetic','admin')", user, user.String()+"@invalid.test")
	require.NoError(t, err)
	var space uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('delivery') RETURNING id").Scan(&space))
	_, err = pool.Exec(ctx, `INSERT INTO claimctl.webhook_outbox(webhook_id,resource_id,event,hook,reservation,state,attempts) VALUES($1,$2,'reservation.created','{"signing_secret":"ENC:never-expose"}','{}','dead',8)`, hook, resource)
	require.NoError(t, err)
	tokens := services.NewAPITokenService(store)
	privileged, _, err := tokens.GenerateToken(ctx, user, "delivery", nil, services.TokenOptions{Scopes: []string{"webhooks:write"}})
	require.NoError(t, err)
	ordinary, _, err := tokens.GenerateToken(ctx, user, "ordinary", nil)
	require.NoError(t, err)
	restricted, _, err := tokens.GenerateToken(ctx, user, "restricted", nil, services.TokenOptions{Scopes: []string{"webhooks:write"}, SpaceIDs: []uuid.UUID{space}})
	require.NoError(t, err)
	app := fiber.New()
	app.Use(BearerAuthentication(tokens, store, nil), CookieAuthentication(store, nil), AuthorizationMiddleware(store))
	h := handlers.NewWebhookHandler(services.NewWebhookService(store, nil, ""))
	app.Get("/api/webhooks/:id/deliveries", AdminMiddleware(), h.ListDeliveries)
	app.Get("/api/webhooks/deliveries/dead", AdminMiddleware(), h.ListDeadDeliveries)
	path := "/api/webhooks/" + hook.String() + "/deliveries"
	call := func(raw string, want int) string {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, want, resp.StatusCode, string(body))
		return string(body)
	}
	body := call(privileged, 200)
	require.Contains(t, body, `"state":"dead"`)
	require.Contains(t, body, `"attempts":8`)
	require.NotContains(t, body, "never-expose")
	require.NotContains(t, body, "signing_secret")
	call(ordinary, 403)
	call(restricted, 403)
	path = "/api/webhooks/deliveries/dead"
	require.Contains(t, call(privileged, 200), hook.String())
	call(ordinary, 403)
	call(restricted, 403)
	_, err = pool.Exec(ctx, "UPDATE claimctl.users SET role='user' WHERE id=$1", user)
	require.NoError(t, err)
	call(privileged, 403)
}
