package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestClaimReacquiresExpiredOwnReservationRealDatabase(t *testing.T) {
	for _, deadline := range []string{"lease_expires_at", "hard_end_time", "scheduled_end_time"} {
		t.Run(deadline, func(t *testing.T) {
			pool := testutils.SecurityTestDatabase(t)
			ctx := context.Background()
			store := db.NewStore(pool)
			actor, resource, _ := seedDelivery(t, pool, "https://example.com")
			_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
			require.NoError(t, err)
			svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
			svc.policy.MaxLiveClaims = 1
			lease := "10m"
			first, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource, LeaseDuration: &lease}, "first")
			require.NoError(t, err)
			_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET "+deadline+"=statement_timestamp()-interval '1 second' WHERE id=$1", first.ID)
			require.NoError(t, err)
			second, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource, NoQueue: true}, "second")
			require.NoError(t, err, "elapsed claims must not consume quota or block their owner's next acquisition")
			require.NotEqual(t, first.ID, second.ID)
			require.Equal(t, "active", second.Status.String)
			previous, err := store.FindReservationById(ctx, first.ID)
			require.NoError(t, err)
			require.Equal(t, "completed", previous.Status.String)
			count, err := store.CountLiveClaims(ctx, actor)
			require.NoError(t, err)
			require.EqualValues(t, 1, count)
		})
	}
}

func TestClaimRequeuesExpiredOwnPendingReservationRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	owner, resource, _ := seedDelivery(t, pool, "https://example.com")
	actor, _ := seedRefreshAccount(t, pool)
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=ANY($1)", []uuid.UUID{owner, actor})
	require.NoError(t, err)
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	svc.policy.MaxLiveClaims = 1
	_, err = svc.Claim(ctx, owner, ClaimRequest{ResourceID: resource}, "owner")
	require.NoError(t, err)
	lease := "10m"
	first, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource, LeaseDuration: &lease}, "first")
	require.NoError(t, err)
	require.Equal(t, "pending", first.Status.String)
	_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET lease_expires_at=statement_timestamp()-interval '1 second' WHERE id=$1", first.ID)
	require.NoError(t, err)
	second, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource}, "second")
	require.NoError(t, err)
	require.Equal(t, "pending", second.Status.String)
	require.EqualValues(t, 1, second.QueuePosition.Int32)
	previous, err := store.FindReservationById(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", previous.Status.String)
}

func TestBundleQuotaIgnoresExpiredClaimsRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
	require.NoError(t, err)
	svc := NewReservationService(store, nil, nil, nil, nil).(*reservationService)
	svc.policy.MaxLiveClaims = 2
	lease := "10m"
	old, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource, LeaseDuration: &lease}, "old")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET lease_expires_at=statement_timestamp()-interval '1 second' WHERE id=$1", old.ID)
	require.NoError(t, err)
	var space uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "SELECT space_id FROM claimctl.resources WHERE id=$1", resource).Scan(&space))
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	for _, id := range ids {
		_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'test',$3)", id, id.String(), space)
		require.NoError(t, err)
	}
	bundle, err := svc.Bundle(ctx, actor, BundleRequest{ResourceIDs: ids}, "replacement")
	require.NoError(t, err, "elapsed claims on unrelated resources must not exhaust bundle quota")
	require.Len(t, bundle, 2)
	count, err := store.CountLiveClaims(ctx, actor)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	_, err = svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource}, "over-quota")
	require.ErrorIs(t, err, ErrClaimConflict, "unexpired replacement claims still enforce quota")
}
