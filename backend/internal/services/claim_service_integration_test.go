package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestClaimsAtomicRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	user, password := seedRefreshAccount(t, pool)
	_ = password
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", user)
	require.NoError(t, err)
	var space uuid.UUID
	err = pool.QueryRow(ctx, "INSERT INTO claimctl.spaces(name) VALUES('claims') RETURNING id").Scan(&space)
	require.NoError(t, err)
	resources := make([]uuid.UUID, 12)
	for i := range resources {
		resources[i] = uuid.New()
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id,labels) VALUES($1,$2,'atomic',$3,'[\"gpu\"]')", resources[i], fmt.Sprint("atomic", i), space)
		require.NoError(t, err)
	}
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	t.Run("idempotency failure rolls back claim", func(t *testing.T) {
		_, err := pool.Exec(ctx, `CREATE FUNCTION claimctl.reject_claim_key() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected key failure'; END $$; CREATE TRIGGER reject_claim_key BEFORE INSERT ON claimctl.claim_idempotency FOR EACH ROW EXECUTE FUNCTION claimctl.reject_claim_key()`)
		require.NoError(t, err)
		_, err = svc.Claim(ctx, user, ClaimRequest{Type: "atomic", NoQueue: true}, "rollback")
		require.Error(t, err)
		count, err := store.CountLiveClaims(ctx, user)
		require.NoError(t, err)
		require.Zero(t, count)
		_, err = pool.Exec(ctx, "DROP TRIGGER reject_claim_key ON claimctl.claim_idempotency; DROP FUNCTION claimctl.reject_claim_key()")
		require.NoError(t, err)
	})
	t.Run("idempotency", func(t *testing.T) {
		var wg sync.WaitGroup
		ids := make(chan uuid.UUID, 12)
		errs := make(chan error, 12)
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				r, e := svc.Claim(ctx, user, ClaimRequest{Type: "atomic", LabelExpr: "gpu", NoQueue: true}, "same")
				errs <- e
				if e == nil {
					ids <- r.ID
				}
			}()
		}
		wg.Wait()
		close(ids)
		close(errs)
		for e := range errs {
			require.NoError(t, e)
		}
		var first uuid.UUID
		for id := range ids {
			if first == uuid.Nil {
				first = id
			}
			require.Equal(t, first, id)
		}
		_, e := svc.Claim(ctx, user, ClaimRequest{ResourceID: resources[0]}, "same")
		require.ErrorIs(t, e, ErrClaimConflict)
		var count int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE user_id=$1", user).Scan(&count))
		require.Equal(t, 1, count)
	})
	t.Run("concurrent selectors across actors never queue", func(t *testing.T) {
		actors := make([]uuid.UUID, 12)
		for i := range actors {
			actors[i], _ = seedRefreshAccount(t, pool)
			_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actors[i])
			require.NoError(t, err)
		}
		var wg sync.WaitGroup
		errs := make(chan error, 12)
		for _, actor := range actors {
			wg.Add(1)
			go func(actor uuid.UUID) {
				defer wg.Done()
				_, err := svc.Claim(ctx, actor, ClaimRequest{Type: "atomic", NoQueue: true}, "same")
				errs <- err
			}(actor)
		}
		wg.Wait()
		close(errs)
		success := 0
		for err := range errs {
			if err == nil {
				success++
			} else {
				require.ErrorIs(t, err, ErrClaimConflict)
			}
		}
		require.Equal(t, 11, success)
		var pending int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE status='pending'").Scan(&pending))
		require.Zero(t, pending)
		for _, actor := range actors {
			rows, err := store.FindUserActiveReservations(ctx, actor)
			require.NoError(t, err)
			for _, r := range rows {
				require.NoError(t, store.CancelReservation(ctx, r.ID))
			}
		}
	})
	t.Run("quota across concurrent resources and queues", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan error, 24)
		for i := 0; i < 24; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, e := svc.Claim(ctx, user, ClaimRequest{ResourceID: resources[i%len(resources)]}, "")
				results <- e
			}(i)
		}
		wg.Wait()
		close(results)
		success := 0
		for e := range results {
			if e == nil {
				success++
			} else {
				require.ErrorIs(t, e, ErrClaimConflict)
			}
		}
		require.Equal(t, 7, success)
		count, e := store.CountLiveClaims(ctx, user)
		require.NoError(t, e)
		require.Equal(t, int64(8), count)
	})
	t.Run("busy noQueue and immediate transitions", func(t *testing.T) {
		other, _ := seedRefreshAccount(t, pool)
		_, e := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", other)
		require.NoError(t, e)
		claims, e := store.FindUserActiveReservations(ctx, user)
		require.NoError(t, e)
		busy := claims[0]
		_, e = svc.Claim(ctx, other, ClaimRequest{ResourceID: busy.ResourceID, NoQueue: true}, "")
		require.ErrorIs(t, e, ErrClaimConflict)
		count, e := store.CountLiveClaims(ctx, other)
		require.NoError(t, e)
		require.Zero(t, count)
		pending, e := svc.Claim(ctx, other, ClaimRequest{ResourceID: busy.ResourceID}, "")
		require.NoError(t, e)
		require.Equal(t, "pending", pending.Status.String)
		require.NoError(t, store.CompleteReservation(ctx, busy.ID))
		activated, err := store.ActivateReservation(ctx, pending.ID)
		require.NoError(t, err)
		require.Equal(t, int64(1), activated)
		require.NoError(t, store.CancelReservation(ctx, pending.ID))
		released, e := store.FindReservationById(ctx, pending.ID)
		require.NoError(t, e)
		require.GreaterOrEqual(t, released.EndTime.Int64, released.StartTime.Int64)
		for _, r := range claims[1:] {
			require.NoError(t, store.CancelReservation(ctx, r.ID))
		}
		for i := 0; i < 20; i++ {
			r, e := svc.Claim(ctx, user, ClaimRequest{ResourceID: busy.ResourceID}, "")
			require.NoError(t, e)
			require.NoError(t, store.CompleteReservation(ctx, r.ID))
		}
	})
}
