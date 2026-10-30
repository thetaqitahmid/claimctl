package testutils

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

func (m *MockQuerier) CleanupDeliveredWebhooks(ctx context.Context, cutoff pgtype.Timestamptz) error {
	a := m.Called(ctx, cutoff)
	return a.Error(0)
}

func (m *MockQuerier) ClaimWebhookDelivery(ctx context.Context, fence pgtype.UUID) (db.ClaimctlWebhookOutbox, error) {
	a := m.Called(ctx, fence)
	return a.Get(0).(db.ClaimctlWebhookOutbox), a.Error(1)
}
func (m *MockQuerier) FinishWebhookDelivery(ctx context.Context, arg db.FinishWebhookDeliveryParams) (int64, error) {
	a := m.Called(ctx, arg)
	return a.Get(0).(int64), a.Error(1)
}
func (m *MockQuerier) ListWebhookDeliveries(ctx context.Context, id uuid.UUID) ([]db.ListWebhookDeliveriesRow, error) {
	a := m.Called(ctx, id)
	return a.Get(0).([]db.ListWebhookDeliveriesRow), a.Error(1)
}
func (m *MockQuerier) AcquireWorkerLease(ctx context.Context, arg db.AcquireWorkerLeaseParams) (uuid.UUID, error) {
	a := m.Called(ctx, arg)
	return a.Get(0).(uuid.UUID), a.Error(1)
}
func (m *MockQuerier) FinishWorkerLease(ctx context.Context, arg db.FinishWorkerLeaseParams) (int64, error) {
	a := m.Called(ctx, arg)
	return a.Get(0).(int64), a.Error(1)
}
func (m *MockQuerier) TryWorkerTransactionLock(ctx context.Context, name string) (bool, error) {
	a := m.Called(ctx, name)
	return a.Bool(0), a.Error(1)
}
