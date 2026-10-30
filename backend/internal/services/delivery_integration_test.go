package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func seedDelivery(t *testing.T, pool *pgxpool.Pool, target string) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	actor, _ := seedRefreshAccount(t, pool)
	var space, resource, hook uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('delivery-' || gen_random_uuid()) RETURNING id").Scan(&space))
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.resources(name,type,space_id) VALUES('delivery-' || gen_random_uuid(),'test',$1) RETURNING id", space).Scan(&resource))
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.webhooks(name,url,method,headers,signing_secret) VALUES('delivery-' || gen_random_uuid(),$1,'POST','{}','synthetic') RETURNING id", target).Scan(&hook))
	_, err := pool.Exec(ctx, `INSERT INTO claimctl.resource_webhooks(resource_id,webhook_id,events) VALUES($1,$2,ARRAY['reservation.created','reservation.activated','reservation.completed','reservation.cancelled','reservation.expired','reservation.renewed'])`, resource, hook)
	require.NoError(t, err)
	return actor, resource, hook
}

func insertDeliveryReservation(ctx context.Context, q db.Querier, actor, resource uuid.UUID) (db.ClaimctlReservation, error) {
	return q.CreateReservation(ctx, db.CreateReservationParams{ResourceID: resource, UserID: actor, Duration: pgtype.Interval{Microseconds: 60000000, Valid: true}})
}

func TestDeliveryTransactionalIntentRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	injected := errors.New("rollback")
	require.ErrorIs(t, store.ExecTx(ctx, func(q db.Querier) error {
		_, err := insertDeliveryReservation(ctx, q, actor, resource)
		require.NoError(t, err)
		return injected
	}), injected)
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Zero(t, count)
	var reservation db.ClaimctlReservation
	require.NoError(t, store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		reservation, err = insertDeliveryReservation(ctx, q, actor, resource)
		return err
	}))
	job, err := store.ClaimWebhookDelivery(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
	require.NoError(t, err)
	hook, payload, err := decodeDelivery(job)
	require.NoError(t, err)
	require.Equal(t, resource, payload.ResourceID)
	require.Equal(t, reservation, payload.Data)
	require.Equal(t, "synthetic", hook.SigningSecret)
	require.NoError(t, store.CancelReservation(ctx, reservation.ID))
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Equal(t, 2, count)
	// Snapshot is unchanged after the source reservation is cancelled.
	_, payload, err = decodeDelivery(job)
	require.NoError(t, err)
	require.Equal(t, reservation, payload.Data)
}

func TestDeliveryConcurrentFencingRecoveryRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, err := insertDeliveryReservation(ctx, store, actor, resource)
	require.NoError(t, err)
	var winners atomic.Int32
	var first db.ClaimctlWebhookOutbox
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, err := store.ClaimWebhookDelivery(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
			if err == nil {
				winners.Add(1)
				mu.Lock()
				first = job
				mu.Unlock()
			} else if err != pgx.ErrNoRows {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, winners.Load())
	_, err = pool.Exec(ctx, "UPDATE claimctl.webhook_outbox SET lease_until=clock_timestamp()-INTERVAL '1 second'")
	require.NoError(t, err)
	n, err := store.FinishWebhookDelivery(ctx, db.FinishWebhookDeliveryParams{ID: first.ID, Fence: first.Fence, State: "delivered"})
	require.NoError(t, err)
	require.Zero(t, n)
	recovered, err := store.ClaimWebhookDelivery(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
	require.NoError(t, err)
	require.Equal(t, first.ID, recovered.ID)
	require.NotEqual(t, first.Fence, recovered.Fence)
	n, err = store.FinishWebhookDelivery(ctx, db.FinishWebhookDeliveryParams{ID: first.ID, Fence: first.Fence, State: "delivered"})
	require.NoError(t, err)
	require.Zero(t, n)
	n, err = store.FinishWebhookDelivery(ctx, db.FinishWebhookDeliveryParams{ID: recovered.ID, Fence: recovered.Fence, State: "delivered"})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func TestDeliveryRetriesRestartDeadLetterRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	var ids []string
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, r.Header.Get("X-claimctl-Delivery-ID"))
		calls.Add(1)
		require.NotEmpty(t, r.Header.Get("X-claimctl-Signature"))
		w.WriteHeader(503)
	}))
	defer server.Close()
	actor, resource, hook := seedDelivery(t, pool, server.URL)
	_, err := insertDeliveryReservation(ctx, store, actor, resource)
	require.NoError(t, err)
	for i := 0; i < 8; i++ {
		svc := NewWebhookService(store, nil, "")
		svc.httpClient = server.Client()
		worked, err := svc.DeliverOne(ctx)
		require.NoError(t, err)
		require.True(t, worked)
		rows, err := store.ListWebhookDeliveries(ctx, hook)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.EqualValues(t, i+1, rows[0].Attempts)
		if i < 7 {
			require.Equal(t, "pending", rows[0].State)
			worked, err = svc.DeliverOne(ctx)
			require.NoError(t, err)
			require.False(t, worked, "backoff must persist across restart")
			_, err = pool.Exec(ctx, "UPDATE claimctl.webhook_outbox SET available_at=clock_timestamp()-INTERVAL '1 second'")
			require.NoError(t, err)
		} else {
			require.Equal(t, "dead", rows[0].State)
		}
	}
	require.EqualValues(t, 8, calls.Load())
	for _, id := range ids {
		require.Equal(t, ids[0], id)
		require.NotEmpty(t, id)
	}
	rows, err := NewWebhookService(store, nil, "").ListDeliveries(ctx, hook)
	require.NoError(t, err)
	raw, err := json.Marshal(rows)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "synthetic")
	require.NotContains(t, string(raw), "signingSecret")
	require.NotContains(t, string(raw), `"hook":`)
}

func TestHealthWorkerFencingRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	_, resource, _ := seedDelivery(t, pool, "https://example.com")
	name := "health:" + resource.String()
	fence := uuid.New()
	_, err := store.AcquireWorkerLease(ctx, db.AcquireWorkerLeaseParams{Name: name, Fence: fence})
	require.NoError(t, err)
	_, err = store.AcquireWorkerLease(ctx, db.AcquireWorkerLeaseParams{Name: name, Fence: uuid.New()})
	require.ErrorIs(t, err, pgx.ErrNoRows)
	svc := NewHealthCheckService(store).(*healthCheckService)
	_, err = pool.Exec(ctx, "UPDATE claimctl.worker_leases SET lease_until=clock_timestamp()-INTERVAL '1 second'")
	require.NoError(t, err)
	fresh := uuid.New()
	_, err = store.AcquireWorkerLease(ctx, db.AcquireWorkerLeaseParams{Name: name, Fence: fresh})
	require.NoError(t, err)
	require.NoError(t, svc.recordHealthStatus(ctx, resource, HealthStatusDown, 0, "stale", name, fence, 60))
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.resource_health_status").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, svc.recordHealthStatus(ctx, resource, HealthStatusHealthy, 1, "", name, fresh, 60))
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.resource_health_status").Scan(&count))
	require.Equal(t, 1, count)
	_, err = store.AcquireWorkerLease(ctx, db.AcquireWorkerLeaseParams{Name: name, Fence: uuid.New()})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestDeliveryBoundedSaturationRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := db.NewStore(pool)
	var active, maximum, calls atomic.Int32
	gate := make(chan struct{})
	var release sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		for {
			old := maximum.Load()
			if n <= old || maximum.CompareAndSwap(old, n) {
				break
			}
		}
		calls.Add(1)
		select {
		case <-gate:
		case <-r.Context().Done():
		}
		w.WriteHeader(200)
	}))
	defer func() { cancel(); release.Do(func() { close(gate) }); server.Close() }()
	actor, resource, _ := seedDelivery(t, pool, server.URL)
	for i := 0; i < 20; i++ {
		if i > 0 {
			actor, _ = seedRefreshAccount(t, pool)
		}
		_, err := insertDeliveryReservation(ctx, store, actor, resource)
		require.NoError(t, err)
	}
	svc := NewWebhookService(store, nil, "")
	svc.httpClient = server.Client()
	svc.StartDelivery(ctx)
	svc.StartDelivery(ctx)
	defer svc.WaitDelivery()
	defer cancel()
	require.Eventually(t, func() bool { return calls.Load() == 4 }, 5*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 4, maximum.Load())
	release.Do(func() { close(gate) })
	require.Eventually(t, func() bool {
		var n int
		err := pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox WHERE state='delivered'").Scan(&n)
		return err == nil && n == 20
	}, 5*time.Second, 10*time.Millisecond)
	require.LessOrEqual(t, maximum.Load(), int32(4))
	cancel()
}
