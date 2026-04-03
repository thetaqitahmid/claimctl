package testutils

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) Log(ctx context.Context, actorID uuid.UUID, action, entityType, entityID string, changes interface{}, ip string) {
	m.Called(ctx, actorID, action, entityType, entityID, changes, ip)
}

func (m *MockAuditService) GetAuditLogs(ctx context.Context, limit, offset int32) ([]db.GetAuditLogsRow, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]db.GetAuditLogsRow), args.Error(1)
}
