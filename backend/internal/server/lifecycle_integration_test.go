package server

import (
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLifecycleScopedHTTPRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	store := db.NewStore(pool)
	ctx := context.Background()
	actor := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Lifecycle','synthetic','admin')", actor, actor.String()+"@invalid.test")
	require.NoError(t, err)
	var allowed, foreign uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('allowed') RETURNING id").Scan(&allowed))
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('foreign') RETURNING id").Scan(&foreign))
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for i, id := range ids {
		space := allowed
		if i == 2 {
			space = foreign
		}
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'http',$3)", id, id.String(), space)
		require.NoError(t, err)
	}
	tokens := services.NewAPITokenService(store)
	raw, _, err := tokens.GenerateToken(ctx, actor, "scoped", nil, services.TokenOptions{Scopes: []string{"read", "reserve"}, SpaceIDs: []uuid.UUID{allowed}})
	require.NoError(t, err)
	adminRaw, _, err := tokens.GenerateToken(ctx, actor, "metrics", nil, services.TokenOptions{Scopes: []string{"metrics:read"}})
	require.NoError(t, err)
	app := fiber.New()
	app.Use(BearerAuthentication(tokens, store, nil), CookieAuthentication(store, nil), AuthorizationMiddleware(store))
	h := handlers.NewReservationHandler(services.NewReservationService(store, nil, nil, nil, nil))
	app.Post("/api/reservations/bundle", h.Bundle)
	app.Post("/api/reservations/:id/renew", h.Renew)
	app.Get("/api/resources/:id/metrics", h.Metrics)
	app.Get("/api/admin/metrics", AdminMiddleware(), h.AggregateMetrics)
	call := func(method, path, body, key, token string, status int) []byte {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		resp, e := app.Test(req)
		require.NoError(t, e)
		defer resp.Body.Close()
		payload, e := io.ReadAll(resp.Body)
		require.NoError(t, e)
		require.Equal(t, status, resp.StatusCode, string(payload))
		return payload
	}
	call("POST", "/api/reservations/bundle", `{"resourceIds":["`+ids[0].String()+`","`+ids[2].String()+`"]}`, "denied", raw, 403)
	body := `{"resourceIds":["` + ids[1].String() + `","` + ids[0].String() + `"],"leaseDuration":"10m"}`
	first := call("POST", "/api/reservations/bundle", body, "bundle", raw, 201)
	require.JSONEq(t, string(first), string(call("POST", "/api/reservations/bundle", body, "bundle", raw, 201)))
	var reservations []db.ClaimctlReservation
	require.NoError(t, json.Unmarshal(first, &reservations))
	call("POST", "/api/reservations/"+reservations[0].ID.String()+"/renew", `{}`, "", raw, 200)
	_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET lease_expires_at=statement_timestamp()-interval '1 second' WHERE id=$1", reservations[0].ID)
	require.NoError(t, err)
	call("POST", "/api/reservations/"+reservations[0].ID.String()+"/renew", `{}`, "", raw, 409)
	call("GET", "/api/resources/"+ids[2].String()+"/metrics", "", "", raw, 403)
	call("GET", "/api/resources/"+ids[0].String()+"/metrics?from=100&to=200", "", "", raw, 200)
	call("GET", "/api/admin/metrics", "", "", raw, 403)
	call("GET", "/api/admin/metrics", "", "", adminRaw, 200)
	_, err = pool.Exec(ctx, "UPDATE claimctl.users SET role='user' WHERE id=$1", actor)
	require.NoError(t, err)
	call("GET", "/api/admin/metrics", "", "", adminRaw, 403)
}
