package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"sync"
	"testing"
)

func TestLifecycleRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, _ := seedRefreshAccount(t, pool)
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
	require.NoError(t, err)
	var space uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('lifecycle') RETURNING id").Scan(&space))
	resources := make([]uuid.UUID, 6)
	for i := range resources {
		resources[i] = uuid.New()
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'lifecycle',$3)", resources[i], fmt.Sprint(i), space)
		require.NoError(t, err)
	}
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	lease := "10m"
	t.Run("concurrent bundle replay and rollback", func(t *testing.T) {
		req := BundleRequest{ResourceIDs: resources[:2], LeaseDuration: &lease}
		var wg sync.WaitGroup
		results := make(chan []db.ClaimctlReservation, 8)
		errs := make(chan error, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); r, e := svc.Bundle(ctx, actor, req, "bundle"); results <- r; errs <- e }()
		}
		wg.Wait()
		close(results)
		close(errs)
		for e := range errs {
			require.NoError(t, e)
		}
		var first []db.ClaimctlReservation
		for r := range results {
			if first == nil {
				first = r
			}
			require.Equal(t, first, r)
		}
		_, err = svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: []uuid.UUID{resources[2], resources[0]}}, "busy")
		require.ErrorIs(t, err, ErrClaimConflict)
		var count int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE resource_id=$1", resources[2]).Scan(&count))
		require.Zero(t, count)
		_, err = pool.Exec(ctx, `CREATE FUNCTION claimctl.reject_bundle() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'rollback'; END $$; CREATE TRIGGER reject_bundle BEFORE INSERT ON claimctl.bundle_idempotency FOR EACH ROW EXECUTE FUNCTION claimctl.reject_bundle()`)
		require.NoError(t, err)
		_, err = svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: resources[2:4]}, "rollback")
		require.Error(t, err)
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE resource_id=ANY($1)", resources[2:4]).Scan(&count))
		require.Zero(t, count)
		_, err = pool.Exec(ctx, "DROP TRIGGER reject_bundle ON claimctl.bundle_idempotency; DROP FUNCTION claimctl.reject_bundle()")
		require.NoError(t, err)
	})
	t.Run("expired lease cannot resurrect and promotes valid head", func(t *testing.T) {
		active, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resources[4], LeaseDuration: &lease}, "")
		require.NoError(t, err)
		other, _ := seedRefreshAccount(t, pool)
		_, err = pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", other)
		require.NoError(t, err)
		pending, err := svc.Claim(ctx, other, ClaimRequest{ResourceID: resources[4], LeaseDuration: &lease}, "")
		require.NoError(t, err)
		_, err = svc.Renew(ctx, other, pending.ID, nil)
		require.NoError(t, err)
		third, _ := seedRefreshAccount(t, pool)
		_, err = pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", third)
		require.NoError(t, err)
		valid, err := svc.Claim(ctx, third, ClaimRequest{ResourceID: resources[4], LeaseDuration: &lease}, "")
		require.NoError(t, err)
		fourth, _ := seedRefreshAccount(t, pool)
		_, err = pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", fourth)
		require.NoError(t, err)
		remaining, err := svc.Claim(ctx, fourth, ClaimRequest{ResourceID: resources[4], LeaseDuration: &lease}, "")
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET lease_expires_at=statement_timestamp()-interval '1 second' WHERE id=$1", pending.ID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET lease_expires_at=statement_timestamp()-interval '1 second' WHERE id=$1", active.ID)
		require.NoError(t, err)
		var wg sync.WaitGroup
		wg.Add(2)
		var renewErr, expireErr error
		go func() { defer wg.Done(); _, renewErr = svc.Renew(ctx, actor, active.ID, nil) }()
		go func() { defer wg.Done(); expireErr = svc.ExpireReservations(ctx) }()
		wg.Wait()
		require.ErrorIs(t, renewErr, ErrClaimConflict)
		require.NoError(t, expireErr)
		r, err := store.FindReservationById(ctx, active.ID)
		require.NoError(t, err)
		require.Equal(t, "completed", r.Status.String)
		abandoned, err := store.FindReservationById(ctx, pending.ID)
		require.NoError(t, err)
		require.Equal(t, "cancelled", abandoned.Status.String)
		p, err := store.FindReservationById(ctx, valid.ID)
		require.NoError(t, err)
		require.Equal(t, "active", p.Status.String)
		require.LessOrEqual(t, p.ScheduledEndTime.Time.Unix(), p.HardEndTime.Time.Unix())
		queued, err := store.FindReservationById(ctx, remaining.ID)
		require.NoError(t, err)
		require.Equal(t, int32(1), queued.QueuePosition.Int32)
	})
	t.Run("renew wins before stale expiry scan", func(t *testing.T) {
		r, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resources[5], LeaseDuration: &lease}, "")
		require.NoError(t, err)
		before := r.ScheduledEndTime.Time
		renewed, err := svc.Renew(ctx, actor, r.ID, nil)
		require.NoError(t, err)
		require.Equal(t, before, renewed.ScheduledEndTime.Time)
		require.NoError(t, svc.ExpireReservations(ctx))
		current, err := store.FindReservationById(ctx, r.ID)
		require.NoError(t, err)
		require.Equal(t, "active", current.Status.String)
		_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET hard_end_time=statement_timestamp()-interval '1 second' WHERE id=$1", r.ID)
		require.NoError(t, err)
		_, err = svc.Renew(ctx, actor, r.ID, nil)
		require.ErrorIs(t, err, ErrClaimConflict)
	})
	t.Run("metrics exact overlapping clipped intervals", func(t *testing.T) {
		id := resources[3]
		for _, v := range [][3]int64{{80, 120, 70}, {110, 150, 100}, {190, 0, 180}} {
			status := "completed"
			var end interface{} = v[1]
			if v[1] == 0 {
				end = nil
				status = "active"
			}
			_, err = pool.Exec(ctx, "INSERT INTO claimctl.reservations(resource_id,user_id,status,start_time,end_time,created_at) VALUES($1,$2,$3,$4,$5,$6)", id, actor, status, v[0], end, v[2])
			require.NoError(t, err)
		}
		pendingActor, _ := seedRefreshAccount(t, pool)
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.reservations(resource_id,user_id,status,created_at,queue_position) VALUES($1,$2,'pending',170,1)", id, pendingActor)
		require.NoError(t, err)
		m, err := svc.Metrics(ctx, id, 100, 200)
		require.NoError(t, err)
		require.Equal(t, int64(60), m.OccupiedSeconds)
		require.Equal(t, 0.6, m.Utilization)
		require.Equal(t, int64(3), m.ClaimCount)
		require.Equal(t, int64(2), m.QueueWaitSamples)
		require.Equal(t, float64(10), m.MeanQueueWaitSeconds)
		require.Equal(t, int64(30), m.OpenPendingWaitSeconds)
		_, err = svc.Metrics(ctx, id, 200, 100)
		require.ErrorIs(t, err, ErrInvalidClaim)
	})

}
