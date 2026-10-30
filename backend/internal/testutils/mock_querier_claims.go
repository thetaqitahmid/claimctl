package testutils

import (
	"context"
	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

func (m *MockQuerier) LockClaimUser(ctx context.Context, id uuid.UUID) (db.ClaimctlUser, error) {
	a := m.Called(ctx, id)
	return a.Get(0).(db.ClaimctlUser), a.Error(1)
}
func (m *MockQuerier) CountLiveClaims(ctx context.Context, id uuid.UUID) (int64, error) {
	a := m.Called(ctx, id)
	return a.Get(0).(int64), a.Error(1)
}
func (m *MockQuerier) FindClaimCandidates(ctx context.Context) ([]db.ClaimctlResource, error) {
	a := m.Called(ctx)
	return a.Get(0).([]db.ClaimctlResource), a.Error(1)
}
func (m *MockQuerier) GetClaimIdempotency(ctx context.Context, p db.GetClaimIdempotencyParams) (db.ClaimctlClaimIdempotency, error) {
	a := m.Called(ctx, p)
	return a.Get(0).(db.ClaimctlClaimIdempotency), a.Error(1)
}
func (m *MockQuerier) SaveClaimIdempotency(ctx context.Context, p db.SaveClaimIdempotencyParams) error {
	return m.Called(ctx, p).Error(0)
}
