package services

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	types "github.com/thetaqitahmid/claimctl/internal/types"
)

func TestReservationHistoryService_AddManualHistoryLog(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewReservationHistoryService(mockDB)

	tests := []struct {
		name          string
		logReq        AddHistoryLogRequest
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Successfully add history log",
			logReq: AddHistoryLogRequest{
				ResourceID: testutils.TestUUID(1),
				ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(1), Valid: true},
				Action:        "created",
				UserID: testutils.TestUUID(1),
				Details:       types.JSONB{"queue_position": 1, "status": "pending"},
			},
			mockSetup: func() {
				// Validate resource exists
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(resource, nil)
				// Validate user exists
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(user, nil)
				// Add history log
				history := testutils.CreateTestReservationHistory(testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), testutils.TestUUID(1), "created")
				mockDB.On("AddReservationHistoryLog", ctx, mock.AnythingOfType("db.AddReservationHistoryLogParams")).Return(history, nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Resource not found",
			logReq: AddHistoryLogRequest{
				ResourceID: testutils.TestUUID(999),
				ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(1), Valid: true},
				Action:        "created",
				UserID: testutils.TestUUID(1),
				Details:       types.JSONB{"status": "pending"},
			},
			mockSetup: func() {
				// Resource doesn't exist
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(999)).Return(db.ClaimctlResource{}, assert.AnError)
			},
			expectedError: "resource with ID 00000000-0000-0000-0000-0000000003e7 not found",
		},
		{
			name: "User not found",
			logReq: AddHistoryLogRequest{
				ResourceID: testutils.TestUUID(1),
				ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(1), Valid: true},
				Action:        "created",
				UserID: testutils.TestUUID(999),
				Details:       types.JSONB{"status": "pending"},
			},
			mockSetup: func() {
				// Resource exists
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(resource, nil)
				// User doesn't exist
				mockDB.On("FindUserById", ctx, testutils.TestUUID(999)).Return(db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "user with ID 00000000-0000-0000-0000-0000000003e7 not found",
		},
		{
			name: "Database error adding history log",
			logReq: AddHistoryLogRequest{
				ResourceID: testutils.TestUUID(1),
				ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(1), Valid: true},
				Action:        "created",
				UserID: testutils.TestUUID(1),
				Details:       types.JSONB{"status": "pending"},
			},
			mockSetup: func() {
				// Validate resource exists
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(resource, nil)
				// Validate user exists
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(user, nil)
				// Add history log fails
				mockDB.On("AddReservationHistoryLog", ctx, mock.AnythingOfType("db.AddReservationHistoryLogParams")).Return(db.ClaimctlReservationHistory{}, assert.AnError)
			},
			expectedError: "failed to add reservation history log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.AddManualHistoryLog(ctx, tt.logReq)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.logReq.Action, result.Action)
				assert.Equal(t, tt.logReq.UserID, result.UserID)
				assert.Equal(t, tt.logReq.ResourceID, result.ResourceID)
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

func TestReservationHistoryService_GetRecentHistoryByAction(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewReservationHistoryService(mockDB)

	tests := []struct {
		name          string
		action        string
		limit         int32
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedLen   int
	}{
		{
			name:   "Successfully get recent history by action",
			action: "created",
			limit:  10,
			mockSetup: func() {
				history := []db.GetRecentHistoryByActionRow{
					{
						ID: testutils.TestUUID(1),
						ResourceID: testutils.TestUUID(1),
						ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(1), Valid: true},
						Action:        "created",
						UserID: testutils.TestUUID(1),
						Timestamp:     pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						Details:       types.JSONB{"status": "pending"},
					},
					{
						ID: testutils.TestUUID(2),
						ResourceID: testutils.TestUUID(2),
						ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(2), Valid: true},
						Action:        "created",
						UserID: testutils.TestUUID(2),
						Timestamp:     pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						Details:       types.JSONB{"status": "pending"},
					},
				}
				mockDB.On("GetRecentHistoryByAction", ctx, mock.AnythingOfType("db.GetRecentHistoryByActionParams")).Return(history, nil)
			},
			shouldSucceed: true,
			expectedLen:   2,
		},
		{
			name:   "No history found for action",
			action: "nonexistent",
			limit:  10,
			mockSetup: func() {
				mockDB.On("GetRecentHistoryByAction", ctx, mock.AnythingOfType("db.GetRecentHistoryByActionParams")).Return([]db.GetRecentHistoryByActionRow{}, nil)
			},
			shouldSucceed: true,
			expectedLen:   0,
		},
		{
			name:   "Database error getting history",
			action: "created",
			limit:  10,
			mockSetup: func() {
				mockDB.On("GetRecentHistoryByAction", ctx, mock.AnythingOfType("db.GetRecentHistoryByActionParams")).Return([]db.GetRecentHistoryByActionRow{}, assert.AnError)
			},
			expectedError: "failed to get recent history for action created",
		},
		{
			name:   "Get history with limit",
			action: "completed",
			limit:  5,
			mockSetup: func() {
				history := []db.GetRecentHistoryByActionRow{
					{
						ID: testutils.TestUUID(1),
						ResourceID: testutils.TestUUID(1),
						ReservationID: pgtype.UUID{Bytes: testutils.TestUUID(1), Valid: true},
						Action:        "completed",
						UserID: testutils.TestUUID(1),
						Timestamp:     pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						Details:       types.JSONB{"end_time": testutils.FixedTimestamp()},
					},
				}
				mockDB.On("GetRecentHistoryByAction", ctx, mock.MatchedBy(func(params db.GetRecentHistoryByActionParams) bool {
					return params.Action == "completed" && params.Limit == 5
				})).Return(history, nil)
			},
			shouldSucceed: true,
			expectedLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetRecentHistoryByAction(ctx, tt.action, tt.limit)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedLen >= 0 {
					assert.Len(t, *result, tt.expectedLen)
				}
				// Verify all entries have the correct action
				for _, entry := range *result {
					assert.Equal(t, tt.action, entry.Action)
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
