package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestRefreshTokenService_Issue(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	svc := NewRefreshTokenService(mockDB)

	userID := testutils.TestUUID(1)

	mockDB.On("CreateRefreshToken", ctx, mock.MatchedBy(func(p db.CreateRefreshTokenParams) bool {
		return p.UserID == userID && len(p.TokenHash) > 0 && p.FamilyID != uuid.Nil
	})).Return(db.ClaimctlRefreshToken{ID: testutils.TestUUID(2)}, nil)

	raw, err := svc.Issue(ctx, userID)

	assert.NoError(t, err)
	assert.True(t, len(raw) > 0)
	assert.Contains(t, raw, "rt_")
	mockDB.AssertExpectations(t)
}

func TestRefreshTokenService_Rotate_Success(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	svc := NewRefreshTokenService(mockDB)

	userID := testutils.TestUUID(1)
	familyID := testutils.TestUUID(2)
	rawToken := "rt_abc123"
	hash := hashToken(rawToken)

	existing := db.ClaimctlRefreshToken{
		ID:        testutils.TestUUID(3),
		UserID:    userID,
		TokenHash: hash,
		FamilyID:  familyID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
		Revoked:   false,
	}

	mockDB.On("GetRefreshTokenByHash", ctx, hash).Return(existing, nil)
	mockDB.On("DeleteRefreshToken", ctx, hash).Return(nil)
	mockDB.On("CreateRefreshToken", ctx, mock.MatchedBy(func(p db.CreateRefreshTokenParams) bool {
		return p.UserID == userID && p.FamilyID == familyID
	})).Return(db.ClaimctlRefreshToken{ID: testutils.TestUUID(4)}, nil)

	newRaw, gotUserID, err := svc.Rotate(ctx, rawToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, newRaw)
	assert.Equal(t, userID, gotUserID)
	mockDB.AssertExpectations(t)
}

func TestRefreshTokenService_Rotate_TheftDetection(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	svc := NewRefreshTokenService(mockDB)

	familyID := testutils.TestUUID(2)
	rawToken := "rt_stolen"
	hash := hashToken(rawToken)

	revoked := db.ClaimctlRefreshToken{
		TokenHash: hash,
		FamilyID:  familyID,
		Revoked:   true,
	}

	mockDB.On("GetRefreshTokenByHash", ctx, hash).Return(revoked, nil)
	mockDB.On("RevokeRefreshTokenFamily", ctx, familyID).Return(nil)

	_, _, err := svc.Rotate(ctx, rawToken)

	assert.ErrorIs(t, err, ErrRefreshTokenRevoked)
	mockDB.AssertExpectations(t)
}

func TestRefreshTokenService_Rotate_Expired(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	svc := NewRefreshTokenService(mockDB)

	rawToken := "rt_expired"
	hash := hashToken(rawToken)

	expired := db.ClaimctlRefreshToken{
		TokenHash: hash,
		FamilyID:  testutils.TestUUID(1),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
		Revoked:   false,
	}

	mockDB.On("GetRefreshTokenByHash", ctx, hash).Return(expired, nil)

	_, _, err := svc.Rotate(ctx, rawToken)

	assert.ErrorIs(t, err, ErrRefreshTokenExpired)
	mockDB.AssertExpectations(t)
}

func TestRefreshTokenService_Revoke(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	svc := NewRefreshTokenService(mockDB)

	familyID := testutils.TestUUID(1)
	rawToken := "rt_torevoke"
	hash := hashToken(rawToken)

	record := db.ClaimctlRefreshToken{
		TokenHash: hash,
		FamilyID:  familyID,
	}

	mockDB.On("GetRefreshTokenByHash", ctx, hash).Return(record, nil)
	mockDB.On("RevokeRefreshTokenFamily", ctx, familyID).Return(nil)

	err := svc.Revoke(ctx, rawToken)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestRefreshTokenService_RevokeAllByUser(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	svc := NewRefreshTokenService(mockDB)

	userID := testutils.TestUUID(1)

	mockDB.On("RevokeAllUserRefreshTokens", ctx, userID).Return(nil)

	err := svc.RevokeAllByUser(ctx, userID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}
