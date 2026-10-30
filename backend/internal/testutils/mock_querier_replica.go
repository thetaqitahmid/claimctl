package testutils

import (
	"context"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

func (m *MockQuerier) LockRealtimeMutation(ctx context.Context) error  { return m.Called(ctx).Error(0) }
func (m *MockQuerier) LockJWTInitialization(ctx context.Context) error { return m.Called(ctx).Error(0) }
func (m *MockQuerier) CleanupReplicaState(ctx context.Context) error   { return m.Called(ctx).Error(0) }
func (m *MockQuerier) CleanupSharedValues(ctx context.Context) error   { return m.Called(ctx).Error(0) }
func (m *MockQuerier) CleanupRateLimits(ctx context.Context) error     { return m.Called(ctx).Error(0) }
func (m *MockQuerier) DeleteSharedValue(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
func (m *MockQuerier) ResetSharedValues(ctx context.Context) error { return m.Called(ctx).Error(0) }
func (m *MockQuerier) SetSharedValue(ctx context.Context, arg db.SetSharedValueParams) error {
	return m.Called(ctx, arg).Error(0)
}
func (m *MockQuerier) GetSharedValue(ctx context.Context, key string) ([]byte, error) {
	a := m.Called(ctx, key)
	return a.Get(0).([]byte), a.Error(1)
}
func (m *MockQuerier) IncrementRateLimit(ctx context.Context, arg db.IncrementRateLimitParams) (db.IncrementRateLimitRow, error) {
	a := m.Called(ctx, arg)
	return a.Get(0).(db.IncrementRateLimitRow), a.Error(1)
}
func (m *MockQuerier) ListRealtimeEvents(ctx context.Context, id int64) ([]db.ClaimctlRealtimeEvent, error) {
	a := m.Called(ctx, id)
	return a.Get(0).([]db.ClaimctlRealtimeEvent), a.Error(1)
}
func (m *MockQuerier) RealtimeBounds(ctx context.Context) (db.RealtimeBoundsRow, error) {
	a := m.Called(ctx)
	return a.Get(0).(db.RealtimeBoundsRow), a.Error(1)
}
