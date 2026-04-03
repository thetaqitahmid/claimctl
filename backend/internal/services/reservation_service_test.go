package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

type MockRealtimeService struct{}

func (m *MockRealtimeService) Subscribe(ctx context.Context) (chan types.Event, error) {
	return make(chan types.Event), nil
}

func (m *MockRealtimeService) Broadcast(event types.Event) {}

type MockNotificationService struct {
	mock.Mock
}

// Notify implements NotificationService.
func (m *MockNotificationService) Notify(ctx context.Context, userID uuid.UUID, event string, payload NotificationPayload) error {
	args := m.Called(ctx, userID, event, payload)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyReservationCreated(ctx context.Context, reservation *db.ClaimctlReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyReservationCancelled(ctx context.Context, reservation *db.ClaimctlReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyReservationActivated(ctx context.Context, reservation *db.ClaimctlReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyReservationCompleted(ctx context.Context, reservation *db.ClaimctlReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyReservationExpired(ctx context.Context, reservation *db.ClaimctlReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyReservationUpcoming(ctx context.Context, reservation *db.ClaimctlReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func TestReservationService_CreateReservation(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		userID uuid.UUID
		resourceID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:       "Successful reservation creation",
			userID: testutils.TestUUID(1),
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				// Check maintenance status
				mockDB.On("GetResourceMaintenanceStatus", ctx, testutils.TestUUID(1)).Return(pgtype.Bool{Bool: false, Valid: true}, nil)
				// No existing reservation found
				mockDB.On("FindUserReservationForResource", ctx, mock.AnythingOfType("db.FindUserReservationForResourceParams")).Return(db.ClaimctlReservation{}, assert.AnError)
				// Create reservation
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("CreateReservation", ctx, mock.AnythingOfType("db.CreateReservationParams")).Return(reservation, nil)

				// Find Resource for Notification
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_created", mock.Anything).Return(nil)

				// Mock webhooks trigger
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.created"
				})).Return([]db.ClaimctlWebhook{}, nil)

				// History log
			},
			shouldSucceed: true,
		},
		{
			name:       "User already has reservation for resource",
			userID: testutils.TestUUID(1),
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				// Check maintenance status
				mockDB.On("GetResourceMaintenanceStatus", ctx, testutils.TestUUID(1)).Return(pgtype.Bool{Bool: false, Valid: true}, nil)
				existingReservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 1)
				mockDB.On("FindUserReservationForResource", ctx, mock.AnythingOfType("db.FindUserReservationForResourceParams")).Return(existingReservation, nil)
			},
			expectedError: "user already has a reservation for this resource",
		},
		{
			name:       "Database error creating reservation",
			userID: testutils.TestUUID(1),
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				// Check maintenance status
				mockDB.On("GetResourceMaintenanceStatus", ctx, testutils.TestUUID(1)).Return(pgtype.Bool{Bool: false, Valid: true}, nil)
				// No existing reservation found
				mockDB.On("FindUserReservationForResource", ctx, mock.AnythingOfType("db.FindUserReservationForResourceParams")).Return(db.ClaimctlReservation{}, assert.AnError)
				// Create reservation fails
				mockDB.On("CreateReservation", ctx, mock.AnythingOfType("db.CreateReservationParams")).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			expectedError: "failed to create reservation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			mockDB.On("AcquireResourceLock", ctx, mock.Anything).Return(uuid.UUID{}, nil).Maybe()
			mockHistory.ExpectedCalls = nil
			tt.mockSetup()

			result, err := service.CreateReservation(ctx, tt.userID, tt.resourceID, nil)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				assert.Nil(t, result)
			}

			mockDB.AssertExpectations(t)
			mockHistory.AssertExpectations(t)
		})
	}
}

func TestReservationService_ActivateReservation(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		reservationID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:          "Successful reservation activation",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil).Once()
				// No active reservation for this resource
				mockDB.On("FindActiveReservationByResource", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
				// Promote queue
				mockDB.On("UpdateQueuePositions", mock.Anything, mock.Anything).Return(nil)
				// Activate reservation
				mockDB.On("ActivateReservation", ctx, mock.AnythingOfType("db.ActivateReservationParams")).Return(nil)

				// Find Resource for Broadcast
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_activated", mock.Anything).Return(nil)

				// Mock webhooks trigger
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.activated"
				})).Return([]db.ClaimctlWebhook{}, nil)

				// Get updated reservation
				updatedReservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 1)
				updatedReservation.StartTime = pgtype.Int8{Int64: time.Now().Unix(), Valid: true}
				updatedReservation.Status = pgtype.Text{String: "active", Valid: true}
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(updatedReservation, nil).Once()
				// History log
			},
			shouldSucceed: true,
		},
		{
			name:          "Reservation not found",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			expectedError: "reservation not found",
		},
		{
			name:          "Reservation not in pending status",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
			},
			expectedError: "reservation cannot be activated, current status: active",
		},
		{
			name:          "Resource already has active reservation",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// Another active reservation exists
				activeReservation := testutils.CreateTestReservation(testutils.TestUUID(2), testutils.TestUUID(1), testutils.TestUUID(2), "active", 0)
				mockDB.On("FindActiveReservationByResource", ctx, testutils.TestUUID(1)).Return(activeReservation, nil)
			},
			expectedError: "resource already has an active reservation",
		},
		{
			name:          "Database error activating reservation",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// No active reservation for this resource
				mockDB.On("FindActiveReservationByResource", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
				// Promote queue
				mockDB.On("UpdateQueuePositions", mock.Anything, mock.Anything).Return(nil)
				// Activate reservation fails
				mockDB.On("ActivateReservation", ctx, mock.AnythingOfType("db.ActivateReservationParams")).Return(assert.AnError)
			},
			expectedError: "failed to activate reservation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			mockDB.On("AcquireResourceLock", ctx, mock.Anything).Return(uuid.UUID{}, nil).Maybe()
			mockHistory.ExpectedCalls = nil
			tt.mockSetup()

			result, err := service.ActivateReservation(ctx, tt.reservationID)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "active", result.Status.String)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				assert.Nil(t, result)
			}

			mockDB.AssertExpectations(t)
			mockHistory.AssertExpectations(t)
		})
	}
}

func TestReservationService_CompleteReservation(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		reservationID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:          "Successful reservation completion",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 0)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// Complete reservation
				mockDB.On("CompleteReservation", ctx, mock.AnythingOfType("db.CompleteReservationParams")).Return(nil)

				// Mock webhooks trigger
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.completed"
				})).Return([]db.ClaimctlWebhook{}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_completed", mock.Anything).Return(nil)

				// Find Resource for Broadcast
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				// History log

				// History log
				// Process queue
				mockDB.On("GetNextInQueue", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			shouldSucceed: true,
		},
		{
			name:          "Completion activates next in queue",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				// The active reservation being completed
				activeReservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 0)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(activeReservation, nil).Once()

				// Complete reservation
				mockDB.On("CompleteReservation", ctx, mock.AnythingOfType("db.CompleteReservationParams")).Return(nil)

				// Mock webhooks trigger for completed
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.completed"
				})).Return([]db.ClaimctlWebhook{}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_completed", mock.Anything).Return(nil)

				// Find Resource for Broadcast
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				// Process queue - next reservation in queue
				nextReservation := testutils.CreateTestReservation(testutils.TestUUID(2), testutils.TestUUID(1), testutils.TestUUID(2), "pending", 1)
				mockDB.On("GetNextInQueue", ctx, testutils.TestUUID(1)).Return(nextReservation, nil)

				// Find reservation for activation (called from ActivateReservation)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(2)).Return(nextReservation, nil).Once()

				// No active reservation for this resource (the completed one is no longer active)
				mockDB.On("FindActiveReservationByResource", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)

				// Promote queue positions
				mockDB.On("UpdateQueuePositions", mock.Anything, mock.Anything).Return(nil)

				// Activate reservation
				mockDB.On("ActivateReservation", ctx, mock.AnythingOfType("db.ActivateReservationParams")).Return(nil)

				// Mock webhooks trigger for activated
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.activated"
				})).Return([]db.ClaimctlWebhook{}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(2), "reservation_activated", mock.Anything).Return(nil)

				// Get updated reservation after activation
				updatedReservation := testutils.CreateTestReservation(testutils.TestUUID(2), testutils.TestUUID(1), testutils.TestUUID(2), "active", 0)
				updatedReservation.StartTime = pgtype.Int8{Int64: time.Now().Unix(), Valid: true}
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(2)).Return(updatedReservation, nil).Once()
			},
			shouldSucceed: true,
		},
		{
			name:          "Reservation not found",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			expectedError: "reservation not found",
		},
		{
			name:          "Reservation not in active status",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
			},
			expectedError: "reservation cannot be completed, current status: pending",
		},
		{
			name:          "Database error completing reservation",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 0)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// Complete reservation fails
				mockDB.On("CompleteReservation", ctx, mock.AnythingOfType("db.CompleteReservationParams")).Return(assert.AnError)
			},
			expectedError: "failed to complete reservation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			mockDB.On("AcquireResourceLock", ctx, mock.Anything).Return(uuid.UUID{}, nil).Maybe()
			mockHistory.ExpectedCalls = nil
			tt.mockSetup()

			err := service.CompleteReservation(ctx, tt.reservationID)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockDB.AssertExpectations(t)
			mockHistory.AssertExpectations(t)
		})
	}
}

func TestReservationService_CancelReservation(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		reservationID uuid.UUID
		userID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:          "Successful pending reservation cancellation",
			reservationID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// Cancel reservation
				mockDB.On("CancelReservation", ctx, mock.AnythingOfType("db.CancelReservationParams")).Return(nil)

				// Mock webhooks trigger
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.cancelled"
				})).Return([]db.ClaimctlWebhook{}, nil)

				// History log
				// Promote queue positions (for pending reservation)
				mockDB.On("UpdateQueuePositions", mock.Anything, mock.Anything).Return(nil)

				// Find Resource for Broadcast
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_cancelled", mock.Anything).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name:          "Successful active reservation cancellation",
			reservationID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 0)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// Cancel reservation
				mockDB.On("CancelReservation", ctx, mock.AnythingOfType("db.CancelReservationParams")).Return(nil)

				// Mock webhooks trigger
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.cancelled"
				})).Return([]db.ClaimctlWebhook{}, nil)

				// History log
				// Process queue (for active reservation)
				mockDB.On("GetNextInQueue", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)

				// Find Resource for Broadcast
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_cancelled", mock.Anything).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name:          "Reservation not found",
			reservationID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			expectedError: "reservation not found",
		},
		{
			name:          "User does not own reservation",
			reservationID: testutils.TestUUID(1),
			userID: testutils.TestUUID(2),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
			},
			expectedError: "user does not own this reservation",
		},
		{
			name:          "Cannot cancel completed reservation",
			reservationID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "completed", 0)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
			},
			expectedError: "reservation cannot be cancelled, current status: completed",
		},
		{
			name:          "Database error cancelling reservation",
			reservationID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
				// Cancel reservation fails
				mockDB.On("CancelReservation", ctx, mock.AnythingOfType("db.CancelReservationParams")).Return(assert.AnError)
			},
			expectedError: "failed to cancel reservation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			mockDB.On("AcquireResourceLock", ctx, mock.Anything).Return(uuid.UUID{}, nil).Maybe()
			mockHistory.ExpectedCalls = nil
			tt.mockSetup()

			err := service.CancelReservation(ctx, tt.reservationID, tt.userID)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockDB.AssertExpectations(t)
			mockHistory.AssertExpectations(t)
		})
	}
}

func TestReservationService_GetReservation(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		reservationID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:          "Successfully get reservation",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				reservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(reservation, nil)
			},
			shouldSucceed: true,
		},
		{
			name:          "Reservation not found",
			reservationID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			expectedError: "reservation not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			mockDB.On("AcquireResourceLock", ctx, mock.Anything).Return(uuid.UUID{}, nil).Maybe()
			tt.mockSetup()

			result, err := service.GetReservation(ctx, tt.reservationID)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				assert.Nil(t, result)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestReservationService_GetUserReservations(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		userID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedLen   int
	}{
		{
			name:   "Successfully get user reservations",
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				reservations := []db.FindUserActiveReservationsRow{
					{ID: testutils.TestUUID(1), ResourceID: testutils.TestUUID(1), Status: pgtype.Text{String: "active", Valid: true}},
					{ID: testutils.TestUUID(2), ResourceID: testutils.TestUUID(2), Status: pgtype.Text{String: "pending", Valid: true}},
				}
				mockDB.On("FindUserActiveReservations", ctx, testutils.TestUUID(1)).Return(reservations, nil)
			},
			shouldSucceed: true,
			expectedLen:   2,
		},
		{
			name:   "Database error getting user reservations",
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("FindUserActiveReservations", ctx, testutils.TestUUID(1)).Return([]db.FindUserActiveReservationsRow{}, assert.AnError)
			},
			expectedError: "failed to get user reservations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetUserReservations(ctx, tt.userID)

			if tt.shouldSucceed {
				require.NoError(t, err)
				if tt.expectedLen >= 0 {
					assert.Len(t, result, tt.expectedLen)
				}
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				assert.Nil(t, result)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestReservationService_ProcessQueue(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		resourceID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:       "Process queue with next reservation",
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				// Next reservation in queue
				nextReservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "pending", 1)
				mockDB.On("GetNextInQueue", ctx, testutils.TestUUID(1)).Return(nextReservation, nil)
				// Find reservation for activation
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(nextReservation, nil)
				// No active reservation for this resource
				mockDB.On("FindActiveReservationByResource", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
				// Promote queue
				mockDB.On("UpdateQueuePositions", mock.Anything, mock.Anything).Return(nil)
				// Activate reservation
				mockDB.On("ActivateReservation", ctx, mock.AnythingOfType("db.ActivateReservationParams")).Return(nil)

				// Find Resource for Broadcast
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{Name: "Test Resource"}, nil)

				mockNotification.On("Notify", ctx, testutils.TestUUID(1), "reservation_activated", mock.Anything).Return(nil)

				// Mock webhooks trigger
				mockDB.On("GetWebhooksForEvent", ctx, mock.MatchedBy(func(arg db.GetWebhooksForEventParams) bool {
					return arg.ResourceID == testutils.TestUUID(1) && arg.Column2 == "reservation.activated"
				})).Return([]db.ClaimctlWebhook{}, nil)

				// Get updated reservation
				updatedReservation := testutils.CreateTestReservation(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "active", 1)
				updatedReservation.StartTime = pgtype.Int8{Int64: time.Now().Unix(), Valid: true}
				mockDB.On("FindReservationById", ctx, testutils.TestUUID(1)).Return(updatedReservation, nil)
				// History log
			},
			shouldSucceed: true,
		},
		{
			name:       "No reservations in queue",
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("GetNextInQueue", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			shouldSucceed: true, // Should succeed with no error
		},
		{
			name:       "Database error getting next in queue",
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("GetNextInQueue", ctx, testutils.TestUUID(1)).Return(db.ClaimctlReservation{}, assert.AnError)
			},
			shouldSucceed: true, // Should succeed with no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			mockDB.On("AcquireResourceLock", ctx, mock.Anything).Return(uuid.UUID{}, nil).Maybe()
			mockHistory.ExpectedCalls = nil
			tt.mockSetup()

			err := service.ProcessQueue(ctx, tt.resourceID)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockDB.AssertExpectations(t)
			mockHistory.AssertExpectations(t)
		})
	}
}

func TestReservationService_GetUserQueuePosition(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockHistory := &testutils.MockReservationHistoryService{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	webhookSvc := NewWebhookService(mockDB, secretSvc)
	mockDB.On("GetWebhooksForEvent", ctx, mock.Anything).Return(nil, nil).Maybe()
	mockRealtime := &MockRealtimeService{}
	mockNotification := &MockNotificationService{}
	service := NewReservationService(mockDB, mockHistory, webhookSvc, mockRealtime, mockNotification)

	tests := []struct {
		name          string
		userID uuid.UUID
		resourceID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedPos   int64
	}{
		{
			name:       "Successfully get queue position",
			userID: testutils.TestUUID(1),
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				position := pgtype.Int4{Int32: 3, Valid: true}
				mockDB.On("GetUserQueuePosition", ctx, mock.AnythingOfType("db.GetUserQueuePositionParams")).Return(position, nil)
			},
			shouldSucceed: true,
			expectedPos:   3,
		},
		{
			name:       "User not in queue",
			userID: testutils.TestUUID(1),
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("GetUserQueuePosition", ctx, mock.AnythingOfType("db.GetUserQueuePositionParams")).Return(pgtype.Int4{}, assert.AnError)
			},
			expectedError: "user not in queue for this resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetUserQueuePosition(ctx, tt.userID, tt.resourceID)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPos, result)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockDB.AssertExpectations(t)
		})
	}
}
