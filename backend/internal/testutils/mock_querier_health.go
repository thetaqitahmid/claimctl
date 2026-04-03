package testutils

import (
	"context"

	"github.com/google/uuid"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

// Health Check Methods
func (m *MockQuerier) CleanupOldHealthStatus(ctx context.Context, checkedAt int64) error {
	args := m.Called(ctx, checkedAt)
	return args.Error(0)
}

func (m *MockQuerier) CreateHealthStatus(ctx context.Context, arg db.CreateHealthStatusParams) (db.ClaimctlResourceHealthStatus, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlResourceHealthStatus), args.Error(1)
}

func (m *MockQuerier) DeleteHealthConfig(ctx context.Context, resourceID uuid.UUID) error {
	args := m.Called(ctx, resourceID)
	return args.Error(0)
}

func (m *MockQuerier) GetAllEnabledHealthConfigs(ctx context.Context) ([]db.ClaimctlResourceHealthConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.ClaimctlResourceHealthConfig), args.Error(1)
}

func (m *MockQuerier) GetHealthConfig(ctx context.Context, resourceID uuid.UUID) (db.ClaimctlResourceHealthConfig, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(db.ClaimctlResourceHealthConfig), args.Error(1)
}

func (m *MockQuerier) GetHealthHistory(ctx context.Context, arg db.GetHealthHistoryParams) ([]db.ClaimctlResourceHealthStatus, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.ClaimctlResourceHealthStatus), args.Error(1)
}

func (m *MockQuerier) GetHealthStatus(ctx context.Context, resourceID uuid.UUID) (db.ClaimctlResourceHealthStatus, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(db.ClaimctlResourceHealthStatus), args.Error(1)
}

func (m *MockQuerier) GetResourcesDueForCheck(ctx context.Context) ([]db.GetResourcesDueForCheckRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.GetResourcesDueForCheckRow), args.Error(1)
}

func (m *MockQuerier) UpsertHealthConfig(ctx context.Context, arg db.UpsertHealthConfigParams) (db.ClaimctlResourceHealthConfig, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlResourceHealthConfig), args.Error(1)
}
