package testutils

import (
	"context"
	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

func (m *MockQuerier) SetReservationLifetime(ctx context.Context, p db.SetReservationLifetimeParams) (db.ClaimctlReservation, error) {
	a := m.Called(ctx, p)
	return a.Get(0).(db.ClaimctlReservation), a.Error(1)
}
func (m *MockQuerier) RenewReservationLease(ctx context.Context, p db.RenewReservationLeaseParams) (db.ClaimctlReservation, error) {
	a := m.Called(ctx, p)
	return a.Get(0).(db.ClaimctlReservation), a.Error(1)
}
func (m *MockQuerier) ReclaimResourceReservations(ctx context.Context, id uuid.UUID) ([]db.ClaimctlReservation, error) {
	a := m.Called(ctx, id)
	return a.Get(0).([]db.ClaimctlReservation), a.Error(1)
}
func (m *MockQuerier) GetBundleIdempotency(ctx context.Context, p db.GetBundleIdempotencyParams) (db.ClaimctlBundleIdempotency, error) {
	a := m.Called(ctx, p)
	return a.Get(0).(db.ClaimctlBundleIdempotency), a.Error(1)
}
func (m *MockQuerier) SaveBundleIdempotency(ctx context.Context, p db.SaveBundleIdempotencyParams) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockQuerier) SetPendingQueuePosition(ctx context.Context, p db.SetPendingQueuePositionParams) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockQuerier) FindMetricReservations(ctx context.Context, id uuid.UUID) ([]db.ClaimctlReservation, error) {
	a := m.Called(ctx, id)
	return a.Get(0).([]db.ClaimctlReservation), a.Error(1)
}
