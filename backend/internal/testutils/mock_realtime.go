package testutils

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

// MockRealtimeService implements the RealtimeService interface for testing
type MockRealtimeService struct {
	mock.Mock
}

func (m *MockRealtimeService) Subscribe(ctx context.Context) (chan types.Event, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(chan types.Event), args.Error(1)
}

func (m *MockRealtimeService) Broadcast(event types.Event) {
	m.Called(event)
}
