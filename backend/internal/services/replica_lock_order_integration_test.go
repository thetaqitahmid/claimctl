package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestReplicaMixedMutationLockOrderRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := db.NewStore(pool)
	actor, r1, _ := seedDelivery(t, pool, "https://example.com")
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
	require.NoError(t, err)
	resource, err := store.FindResourceById(ctx, r1)
	require.NoError(t, err)
	r2 := uuid.New()
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,'second','http',$2)", r2, resource.SpaceID)
	require.NoError(t, err)
	ready, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	gate := make(chan error, 1)
	go func() {
		gate <- store.ExecTx(ctx, func(q db.Querier) error {
			if _, err := q.LockClaimUser(ctx, actor); err != nil {
				return err
			}
			if _, err := q.AcquireResourceLock(ctx, r1); err != nil {
				return err
			}
			close(ready)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var wg sync.WaitGroup
	results := make(chan error, 5)
	operations := []func() error{
		func() error {
			_, err := NewReservationService(store, nil, nil, nil, nil).(*reservationService).Bundle(ctx, actor, BundleRequest{ResourceIDs: []uuid.UUID{r2, r1}}, "replica-mixed")
			return err
		},
		func() error {
			_, err := pool.Exec(ctx, "UPDATE claimctl.resources SET name='mixed' WHERE id=$1", r1)
			return err
		},
		func() error {
			_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
			return err
		},
		func() error {
			_, err := pool.Exec(ctx, "INSERT INTO claimctl.space_permissions(space_id,user_id) VALUES($1,$2)", resource.SpaceID, actor)
			return err
		},
		func() error {
			_, err := pool.Exec(ctx, "INSERT INTO claimctl.realtime_events(event_type) VALUES('resync')")
			return err
		},
	}
	for _, operation := range operations {
		wg.Add(1)
		go func(f func() error) { defer wg.Done(); results <- f() }(operation)
	}
	require.Eventually(t, func() bool {
		var count int
		err := pool.QueryRow(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND NOT granted AND objid=725025").Scan(&count)
		return err == nil && count == len(operations)
	}, time.Second, 5*time.Millisecond, "all writers fence before business locks")
	close(release)
	require.NoError(t, <-gate)
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err, "mixed emitting transactions must not deadlock")
	}
	var active int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE status='active'").Scan(&active))
	require.Equal(t, 2, active)
}
