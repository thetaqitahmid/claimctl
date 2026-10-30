package services

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestClaimsConcurrentQueueQuotaRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	blocker, _ := seedRefreshAccount(t, pool)
	actor, _ := seedRefreshAccount(t, pool)
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id IN ($1,$2)", blocker, actor)
	require.NoError(t, err)
	var space uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('queued quota') RETURNING id").Scan(&space))
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	ids := make([]uuid.UUID, 4)
	for i := range ids {
		ids[i] = uuid.New()
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'queued',$3)", ids[i], ids[i].String(), space)
		require.NoError(t, err)
		_, err = svc.Claim(ctx, blocker, ClaimRequest{ResourceID: ids[i]}, "")
		require.NoError(t, err)
	}
	svc.policy.MaxLiveClaims = 2
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for _, id := range ids {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			r, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: id}, "")
			if err == nil && r.Status.String != "pending" {
				err = ErrInvalidClaim
			}
			results <- err
		}(id)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else {
			require.ErrorIs(t, err, ErrClaimConflict)
		}
	}
	require.Equal(t, 2, success)
	rows, err := store.FindUserActiveReservations(ctx, actor)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, r := range rows {
		require.Equal(t, "pending", r.Status.String)
	}
}
