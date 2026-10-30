package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestClaimsScopedHTTPRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	store := db.NewStore(pool)
	ctx := context.Background()
	user := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Claim Test','synthetic','admin')", user, user.String()+"@invalid.test")
	require.NoError(t, err)
	var allowed uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('claims') RETURNING id").Scan(&allowed))
	foreign := uuid.New()
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.spaces(id,name) VALUES($1,'foreign')", foreign)
	require.NoError(t, err)
	visible, hidden := uuid.New(), uuid.New()
	for _, v := range []struct {
		id, space uuid.UUID
		name      string
	}{{visible, allowed, "visible"}, {hidden, foreign, "hidden"}} {
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'http',$3)", v.id, v.name, v.space)
		require.NoError(t, err)
	}
	tokens := services.NewAPITokenService(store)
	raw, _, err := tokens.GenerateToken(ctx, user, "scoped", nil, services.TokenOptions{Scopes: []string{"read", "reserve"}, SpaceIDs: []uuid.UUID{allowed}})
	require.NoError(t, err)
	app := fiber.New()
	app.Use(BearerAuthentication(tokens, store, nil), CookieAuthentication(store, nil), AuthorizationMiddleware(store))
	realtime := &testutils.MockRealtimeService{}
	realtime.On("Broadcast", mock.Anything).Return()
	webhook := services.NewWebhookService(store, nil, "")
	notification := services.NewNotificationService(store, nil, nil)
	h := handlers.NewReservationHandler(services.NewReservationService(store, nil, webhook, realtime, notification))
	app.Get("/api/reservations/policy", h.Policy)
	app.Post("/api/reservations/claim", h.Claim)
	app.Post("/api/reservations", h.CreateReservation)
	app.Post("/api/reservations/timed", h.CreateTimedReservation)
	app.Patch("/api/reservations/:id/complete", h.CompleteReservation)
	app.Patch("/api/reservations/:id/cancel", h.CancelReservation)
	call := func(path, body, key string, expected int) []byte {
		method := "POST"
		if strings.HasSuffix(path, "/complete") || strings.HasSuffix(path, "/cancel") {
			method = "PATCH"
		}
		if path == "/api/reservations/policy" {
			method = "GET"
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+raw)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		resp, e := app.Test(req)
		require.NoError(t, e)
		defer resp.Body.Close()
		payload, e := io.ReadAll(resp.Body)
		require.NoError(t, e)
		require.Equal(t, expected, resp.StatusCode, string(payload))
		return payload
	}
	require.Contains(t, string(call("/api/reservations/policy", "", "", 200)), `"defaultDuration":"1h"`)
	call("/api/reservations/claim", `{"resourceId":"`+hidden.String()+`"}`, "", 403)
	for _, duration := range []string{"", "0s", "-1h", "25h", "999999999999999999999d", "garbage"} {
		call("/api/reservations/timed", `{"resourceId":"`+visible.String()+`","duration":"`+duration+`"}`, "", 400)
	}
	call("/api/reservations/claim", `{"type":"http","indefinite":true}`, "", 400)
	call("/api/reservations/claim", `{"type":"http","requireHealthy":true,"noQueue":true}`, "", 409)
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.resource_health_status(resource_id,status,checked_at) VALUES($1,'healthy',1)", visible)
	require.NoError(t, err)
	body := `{"type":"http","noQueue":true,"requireHealthy":true}`
	first := call("/api/reservations/claim", body, "http-key", 201)
	second := call("/api/reservations/claim", body, "http-key", 201)
	require.JSONEq(t, string(first), string(second))
	var r db.ClaimctlReservation
	require.NoError(t, json.Unmarshal(first, &r))
	require.Equal(t, visible, r.ResourceID)
	require.True(t, r.Duration.Valid)
	call("/api/reservations/claim", `{"resourceId":"`+visible.String()+`","noQueue":true}`, "", 409)
	call("/api/reservations/claim", `{"type":"http","duration":"2h"}`, "http-key", 409)
	call("/api/reservations", `{"resourceId":"`+hidden.String()+`"}`, "", 403)
	call("/api/reservations/"+r.ID.String()+"/complete", "", "", 200)
	for i := 0; i < 10; i++ {
		payload := call("/api/reservations/claim", body, "", 201)
		require.NoError(t, json.Unmarshal(payload, &r))
		call("/api/reservations/"+r.ID.String()+"/cancel", "", "", 200)
	}
	call("/api/reservations/claim", body, "http-key", 201)
}
