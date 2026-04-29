package testutils

import (
	"context"

	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

func (m *MockQuerier) CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.ClaimctlRefreshToken, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlRefreshToken), args.Error(1)
}

func (m *MockQuerier) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (db.ClaimctlRefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	return args.Get(0).(db.ClaimctlRefreshToken), args.Error(1)
}

func (m *MockQuerier) RevokeRefreshTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

func (m *MockQuerier) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

func (m *MockQuerier) DeleteExpiredRefreshTokens(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQuerier) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
