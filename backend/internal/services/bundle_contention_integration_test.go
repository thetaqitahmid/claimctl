package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"sync"
	"testing"
)

func TestBundleContentionRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actors := make([]uuid.UUID, 2)
	for i := range actors {
		actors[i], _ = seedRefreshAccount(t, pool)
		_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actors[i])
		require.NoError(t, err)
	}
	var space uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('bundle contention') RETURNING id").Scan(&space))
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	for _, id := range ids {
		_, err := pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'bundle',$3)", id, id.String(), space)
		require.NoError(t, err)
	}
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, actor := range actors {
		wg.Add(1)
		go func(i int, actor uuid.UUID) {
			defer wg.Done()
			<-start
			resources := []uuid.UUID{ids[i], ids[1-i]}
			_, err := svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: resources}, "same-key")
			errs <- err
		}(i, actor)
	}
	close(start)
	wg.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, ErrClaimConflict)
			conflicts++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE status='active'").Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(distinct user_id) FROM claimctl.reservations").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE status='pending'").Scan(&count))
	require.Zero(t, count)
}
