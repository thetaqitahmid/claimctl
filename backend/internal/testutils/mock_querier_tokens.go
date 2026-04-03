package testutils

import (
	"context"

	"github.com/google/uuid"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

// Add CreateAPIToken to MockQuerier to satisfy the interface
func (m *MockQuerier) CreateAPIToken(ctx context.Context, arg db.CreateAPITokenParams) (db.ClaimctlApiToken, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlApiToken), args.Error(1)
}

func (m *MockQuerier) ListAPITokens(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlApiToken, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.ClaimctlApiToken), args.Error(1)
}

func (m *MockQuerier) RevokeAPIToken(ctx context.Context, arg db.RevokeAPITokenParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetAPITokenByHash(ctx context.Context, tokenHash string) (db.ClaimctlApiToken, error) {
	args := m.Called(ctx, tokenHash)
	return args.Get(0).(db.ClaimctlApiToken), args.Error(1)
}

func (m *MockQuerier) UpdateAPITokenLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
