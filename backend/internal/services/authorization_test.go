package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

func TestResourceService_ListsFilterCrossSpaceAndRestrictedAdminToken(t *testing.T) {
	userID, allowed, denied := testutils.TestUUID(1), testutils.TestUUID(2), testutils.TestUUID(3)
	for _, admin := range []bool{false, true} {
		q := &testutils.MockQuerier{}
		auth := Authorization{User: db.ClaimctlUser{ID: userID, Status: "active", Role: "user"}}
		if admin {
			auth.User.Role = "admin"
			auth.TokenID = testutils.TestUUID(4)
			auth.SpaceIDs = []uuid.UUID{allowed}
		}
		ctx := WithAuthorization(context.Background(), auth)
		q.On("FindAllResources", ctx).Return([]db.ClaimctlResource{{ID: allowed, SpaceID: allowed}, {ID: denied, SpaceID: denied}}, nil)
		q.On("HasSpacePermission", ctx, mock.MatchedBy(func(p db.HasSpacePermissionParams) bool { return p.ID == allowed })).Return(true, nil).Maybe()
		q.On("HasSpacePermission", ctx, mock.MatchedBy(func(p db.HasSpacePermissionParams) bool { return p.ID == denied })).Return(false, nil).Maybe()
		if admin {
			q.On("GetAllResourcesWithReservationStatus", ctx).Return([]db.GetAllResourcesWithReservationStatusRow{{ID: allowed, SpaceID: allowed}, {ID: denied, SpaceID: denied}}, nil)
		} else {
			q.On("GetAllResourcesWithReservationStatusForUser", ctx, mock.Anything).Return([]db.GetAllResourcesWithReservationStatusForUserRow{{ID: allowed, SpaceID: allowed}, {ID: denied, SpaceID: denied}}, nil)
		}
		q.On("GetHealthConfig", mock.Anything, mock.Anything).Return(db.ClaimctlResourceHealthConfig{}, assert.AnError).Maybe()
		q.On("GetHealthStatus", mock.Anything, mock.Anything).Return(db.ClaimctlResourceHealthStatus{}, assert.AnError).Maybe()
		svc := NewResourceService(q, &testutils.MockRealtimeService{})
		plain, err := svc.GetAllResourcesForUser(ctx, userID, admin, nil)
		require.NoError(t, err)
		require.Len(t, *plain, 1)
		require.Equal(t, allowed, (*plain)[0].SpaceID)
		status, err := svc.GetAllResourcesWithStatusForUser(ctx, userID, admin, nil)
		require.NoError(t, err)
		require.Len(t, *status, 1)
		require.Equal(t, allowed, (*status)[0].Resource.SpaceID)
	}
}

func TestReservationService_CreateReservationDeniesCrossSpace(t *testing.T) {
	q := &testutils.MockQuerier{}
	userID, resourceID := testutils.TestUUID(1), testutils.TestUUID(2)
	q.On("LockClaimUser", mock.Anything, userID).Return(db.ClaimctlUser{ID: userID, Status: "active", Role: "user"}, nil)
	q.On("CountLiveClaims", mock.Anything, userID).Return(int64(0), nil)
	q.On("FindClaimCandidates", mock.Anything).Return([]db.ClaimctlResource{{ID: resourceID, SpaceID: testutils.TestUUID(3)}}, nil)
	q.On("FindResourceById", mock.Anything, resourceID).Return(db.ClaimctlResource{ID: resourceID, SpaceID: testutils.TestUUID(3)}, nil)
	q.On("HasSpacePermission", mock.Anything, mock.Anything).Return(false, nil)
	svc := NewReservationService(q, nil, nil, nil, nil)
	_, err := svc.CreateReservation(context.Background(), userID, resourceID, nil)
	require.ErrorIs(t, err, ErrAccessDenied)
	q.AssertNotCalled(t, "CreateReservation", mock.Anything, mock.Anything)
	q.AssertNotCalled(t, "AcquireResourceLock", mock.Anything, mock.Anything)
}

func TestCanDeliverEvent_DeniesCrossSpaceAndUnknownPayload(t *testing.T) {
	q := &testutils.MockQuerier{}
	auth := Authorization{User: db.ClaimctlUser{ID: testutils.TestUUID(1), Status: "active", Role: "user", Password: "hash"}}
	resourceID := testutils.TestUUID(2)
	q.On("FindResourceById", mock.Anything, resourceID).Return(db.ClaimctlResource{ID: resourceID, SpaceID: testutils.TestUUID(3)}, nil)
	q.On("HasSpacePermission", mock.Anything, mock.Anything).Return(false, nil)
	require.False(t, CanDeliverEvent(context.Background(), q, auth, types.Event{Payload: map[string]any{"resource_id": resourceID}}))
	require.False(t, CanDeliverEvent(context.Background(), q, auth, types.Event{Payload: map[string]any{"secret": "value"}}))
	q.On("FindUserById", mock.Anything, auth.User.ID).Return(db.ClaimctlUser{ID: auth.User.ID, Status: "active", Role: "user", Password: "new"}, nil)
	_, err := RefreshAuthorization(context.Background(), q, auth)
	require.ErrorIs(t, err, ErrAccessDenied)
}
