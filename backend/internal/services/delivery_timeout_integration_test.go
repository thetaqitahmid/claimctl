package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestDeliveryTimeoutRecoveryRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	gate := make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			select {
			case <-gate:
			case <-r.Context().Done():
			}
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	defer close(gate)
	actor, resource, hook := seedDelivery(t, pool, server.URL)
	_, err := insertDeliveryReservation(ctx, store, actor, resource)
	require.NoError(t, err)
	svc := NewWebhookService(store, nil, "")
	svc.httpClient = server.Client()
	svc.httpClient.Timeout = 50 * time.Millisecond
	start := time.Now()
	worked, err := svc.DeliverOne(ctx)
	require.NoError(t, err)
	require.True(t, worked)
	require.Less(t, time.Since(start), time.Second)
	rows, err := store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	require.Equal(t, "pending", rows[0].State)
	require.Equal(t, "network_error", rows[0].LastError)
	_, err = pool.Exec(ctx, "UPDATE claimctl.webhook_outbox SET available_at=clock_timestamp()-INTERVAL '1 second'")
	require.NoError(t, err)
	restarted := NewWebhookService(store, nil, "")
	restarted.httpClient = server.Client()
	worked, err = restarted.DeliverOne(ctx)
	require.NoError(t, err)
	require.True(t, worked)
	rows, err = store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	require.Equal(t, "delivered", rows[0].State)
	require.EqualValues(t, 2, rows[0].Attempts)
}

func TestDeliveryExhaustedLeaseRecoveryRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	actor, resource, hook := seedDelivery(t, pool, server.URL)
	_, err := insertDeliveryReservation(ctx, store, actor, resource)
	require.NoError(t, err)
	_, err = store.ClaimWebhookDelivery(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE claimctl.webhook_outbox SET attempts=8,lease_until=clock_timestamp()-INTERVAL '1 second'")
	require.NoError(t, err)
	svc := NewWebhookService(store, nil, "")
	svc.httpClient = server.Client()
	worked, err := svc.DeliverOne(ctx)
	require.NoError(t, err)
	require.True(t, worked)
	require.Zero(t, calls.Load())
	rows, err := store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	require.Equal(t, "dead", rows[0].State)
	require.EqualValues(t, 8, rows[0].Attempts)
	require.Equal(t, "attempt_limit", rows[0].LastError)
}
