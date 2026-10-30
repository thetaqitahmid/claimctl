package services

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestDeliveryLifecycleSnapshotsRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, hook := seedDelivery(t, pool, "https://example.com")
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
	require.NoError(t, err)
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	lease := "1m"
	r, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource, LeaseDuration: &lease}, "create")
	require.NoError(t, err)
	job, err := store.ClaimWebhookDelivery(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
	require.NoError(t, err)
	_, payload, err := decodeDelivery(job)
	require.NoError(t, err)
	require.Equal(t, *r, payload.Data, "created snapshot includes final lease and hard end")
	rows, err := store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	_, err = svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource, LeaseDuration: &lease}, "create")
	require.NoError(t, err)
	rows, err = store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	longer := "2m"
	_, err = svc.Renew(ctx, actor, r.ID, &longer)
	require.NoError(t, err)
	rows, err = store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "reservation.renewed", rows[0].Event)
	queued, _ := seedRefreshAccount(t, pool)
	_, err = insertDeliveryReservation(ctx, store, queued, resource)
	require.NoError(t, err)
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, "SET LOCAL claimctl.restoring='on'")
	require.NoError(t, err)
	_, err = tx.Exec(ctx, "UPDATE claimctl.reservations SET lease_expires_at=clock_timestamp()-INTERVAL '1 second' WHERE id=$1", r.ID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- svc.ExpireReservations(ctx) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	rows, err = store.ListWebhookDeliveries(ctx, hook)
	require.NoError(t, err)
	events := map[string]int{}
	for _, row := range rows {
		events[row.Event]++
	}
	require.Equal(t, map[string]int{"reservation.created": 2, "reservation.renewed": 1, "reservation.expired": 1, "reservation.activated": 1}, events)
}

func TestDeliveryBundleRollbackRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, second, _ := seedDelivery(t, pool, "https://example.com")
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
	require.NoError(t, err)
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	lease := "1m"
	_, err = pool.Exec(ctx, `CREATE FUNCTION claimctl.reject_bundle_delivery() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$; CREATE TRIGGER reject_bundle_delivery BEFORE INSERT ON claimctl.bundle_idempotency FOR EACH ROW EXECUTE FUNCTION claimctl.reject_bundle_delivery()`)
	require.NoError(t, err)
	_, err = svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: []uuid.UUID{resource, second}, LeaseDuration: &lease}, "rollback")
	require.Error(t, err)
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations").Scan(&count))
	require.Zero(t, count)
	_, err = pool.Exec(ctx, "DROP TRIGGER reject_bundle_delivery ON claimctl.bundle_idempotency; DROP FUNCTION claimctl.reject_bundle_delivery()")
	require.NoError(t, err)
	reservations, err := svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: []uuid.UUID{resource, second}, LeaseDuration: &lease}, "bundle")
	require.NoError(t, err)
	require.Len(t, reservations, 2)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Equal(t, 2, count)
	for i := 0; i < 2; i++ {
		job, err := store.ClaimWebhookDelivery(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
		require.NoError(t, err)
		_, payload, err := decodeDelivery(job)
		require.NoError(t, err)
		row := payload.Data.(db.ClaimctlReservation)
		require.True(t, row.LeaseExpiresAt.Valid)
		require.True(t, row.HardEndTime.Valid)
	}
	_, err = svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: []uuid.UUID{resource, second}, LeaseDuration: &lease}, "bundle")
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Equal(t, 2, count)
}
