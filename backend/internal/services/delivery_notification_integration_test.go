package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestDeliveryNotificationFailureAfterCommitRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, err := pool.Exec(ctx, "UPDATE claimctl.users SET role='admin' WHERE id=$1", actor)
	require.NoError(t, err)
	notification := &MockNotificationService{}
	notification.On("Notify", mock.Anything, actor, "reservation_created", mock.Anything).Run(func(args mock.Arguments) {
		notifyCtx := args.Get(0).(context.Context)
		deadline, ok := notifyCtx.Deadline()
		require.True(t, ok)
		require.False(t, deadline.IsZero())
		var count int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
		require.Equal(t, 1, count, "notification strictly follows commit")
	}).Return(errors.New("notification unavailable")).Once()
	realtime := &testutils.MockRealtimeService{}
	realtime.On("Broadcast", mock.Anything).Return().Once()
	svc := NewReservationService(store, nil, NewWebhookService(store, nil, ""), realtime, notification).(*reservationService)
	r, err := svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource}, "postcommit")
	require.NoError(t, err)
	require.NotNil(t, r)
	_, err = svc.Claim(ctx, actor, ClaimRequest{ResourceID: resource}, "postcommit")
	require.NoError(t, err)
	notification.AssertExpectations(t)
	realtime.AssertExpectations(t)
}
