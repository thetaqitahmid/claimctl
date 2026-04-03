package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestAPITokenService_GenerateToken_Suffix(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewAPITokenService(mockDB)

	userID := testutils.TestUUID(1)
	baseName := "test-token"

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
		return params.Name == expectedName && params.UserID == userID
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
