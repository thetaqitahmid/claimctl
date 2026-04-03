package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func createTestGroup(id uuid.UUID, name, description string) db.ClaimctlGroup {
	return db.ClaimctlGroup{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func TestGroupService_CreateGroup(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewGroupService(mockDB)

	tests := []struct {
		name          string
		groupName     string
		description   string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:        "Successful group creation",
			groupName:   "Test Group",
			description: "A test group",
			mockSetup: func() {
				// Check name uniqueness
				mockDB.On("GetGroupByName", ctx, "Test Group").Return(db.ClaimctlGroup{}, assert.AnError)
				// Create group
				group := createTestGroup(testutils.TestUUID(1), "Test Group", "A test group")
				mockDB.On("CreateGroup", ctx, mock.AnythingOfType("db.CreateGroupParams")).Return(group, nil)
			},
			shouldSucceed: true,
		},
		{
			name:        "Group name already exists",
			groupName:   "Existing Group",
			description: "An existing group",
			mockSetup: func() {
				existingGroup := createTestGroup(testutils.TestUUID(1), "Existing Group", "Existing group")
				mockDB.On("GetGroupByName", ctx, "Existing Group").Return(existingGroup, nil)
			},
			expectedError: "group with name 'Existing Group' already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.CreateGroup(ctx, tt.groupName, tt.description)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.groupName, result.Name)
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

func TestGroupService_AddUserToGroup(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewGroupService(mockDB)

	tests := []struct {
		name          string
		groupID uuid.UUID
		userID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:    "Successfully add user to group",
			groupID: testutils.TestUUID(1),
			userID: testutils.TestUUID(10),
			mockSetup: func() {
				mockDB.On("AddUserToGroup", ctx, mock.AnythingOfType("db.AddUserToGroupParams")).Return(nil)
			},
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			err := service.AddUserToGroup(ctx, tt.groupID, tt.userID)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			mockDB.AssertExpectations(t)
		})
	}
}
