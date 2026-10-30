package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestAPITokenService_GenerateToken_Suffix(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewAPITokenService(mockDB)

	userID := testutils.TestUUID(1)
	baseName := "test-token"
	mockDB.On("FindUserById", ctx, userID).Return(db.ClaimctlUser{ID: userID, Status: "active", Role: "user"}, nil)

	// Mock ListAPITokens to return existing tokens
	// Use mock.Anything for context to avoid pointer mismatches
	mockDB.On("ListAPITokens", ctx, userID).Return([]db.ClaimctlApiToken{
		{Name: baseName},
		{Name: baseName + "-1"},
	}, nil)

	// Mock CreateAPIToken
	// We expect the name to be baseName-2 because baseName and baseName-1 exist
	expectedName := baseName + "-2"
	mockDB.On("CreateAPIToken", ctx, mock.MatchedBy(func(params db.CreateAPITokenParams) bool {
		return params.Name == expectedName && params.UserID == userID && params.ExpiresAt.Valid &&
			params.ExpiresAt.Time.After(time.Now().Add(29*24*time.Hour)) &&
			params.ExpiresAt.Time.Before(time.Now().Add(31*24*time.Hour)) &&
			len(params.Scopes) == 2 && params.Scopes[0] == "read" && params.Scopes[1] == "reserve" && len(params.SpaceIds) == 0
	})).Return(db.ClaimctlApiToken{
		ID:   testutils.TestUUID(1),
		Name: expectedName,
	}, nil)

	// Execute
	_, tokenRecord, err := service.GenerateToken(ctx, userID, baseName, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, tokenRecord)
	assert.Equal(t, expectedName, tokenRecord.Name)

	mockDB.AssertExpectations(t)
}

func TestAPITokenService_ExpiryAndScopeRestrictions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		expiry time.Time
		scope  string
	}{
		{name: "past expiry", expiry: time.Now().Add(-time.Hour)},
		{name: "excess lifetime", expiry: time.Now().Add(91 * 24 * time.Hour)},
		{name: "admin scope forbidden", expiry: time.Now().Add(time.Hour), scope: "resources:write"},
		{name: "unknown scope forbidden", expiry: time.Now().Add(time.Hour), scope: "all"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			id := testutils.TestUUID(1)
			q.On("FindUserById", mock.Anything, id).Return(db.ClaimctlUser{ID: id, Status: "active", Role: "user"}, nil).Maybe()
			_, _, err := NewAPITokenService(q).GenerateToken(context.Background(), id, "token", &tc.expiry, TokenOptions{Scopes: []string{tc.scope}})
			require.Error(t, err)
			q.AssertNotCalled(t, "CreateAPIToken", mock.Anything, mock.Anything)
		})
	}
}
