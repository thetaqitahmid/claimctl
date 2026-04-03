package services

import (
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestSpaceService_CreateSpace(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewSpaceService(mockDB)

	tests := []struct {
		name          string
		spaceName     string
		description   string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:        "Successful space creation",
			spaceName:   "Test Space",
			description: "A test space",
			mockSetup: func() {
				// Check name uniqueness
				mockDB.On("GetSpaceByName", ctx, "Test Space").Return(db.ClaimctlSpace{}, assert.AnError)
				// Create space
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Test Space", "A test space")
				mockDB.On("CreateSpace", ctx, mock.AnythingOfType("db.CreateSpaceParams")).Return(space, nil)
			},
			shouldSucceed: true,
		},
		{
			name:        "Successful space creation with empty description",
			spaceName:   "Test Space",
			description: "",
			mockSetup: func() {
				// Check name uniqueness
				mockDB.On("GetSpaceByName", ctx, "Test Space").Return(db.ClaimctlSpace{}, assert.AnError)
				// Create space
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Test Space", "")
				mockDB.On("CreateSpace", ctx, mock.AnythingOfType("db.CreateSpaceParams")).Return(space, nil)
			},
			shouldSucceed: true,
		},
		{
			name:          "Empty space name",
			spaceName:     "",
			description:   "A test space",
			mockSetup:     func() {},
			expectedError: "space name cannot be empty",
		},
		{
			name:        "Space name already exists",
			spaceName:   "Existing Space",
			description: "An existing space",
			mockSetup: func() {
				existingSpace := testutils.CreateTestSpace(testutils.TestUUID(1), "Existing Space", "Existing space")
				mockDB.On("GetSpaceByName", ctx, "Existing Space").Return(existingSpace, nil)
			},
			expectedError: "space with name 'Existing Space' already exists",
		},
		{
			name:        "Database error creating space",
			spaceName:   "Test Space",
			description: "A test space",
			mockSetup: func() {
				// Check name uniqueness
				mockDB.On("GetSpaceByName", ctx, "Test Space").Return(db.ClaimctlSpace{}, assert.AnError)
				// Create space fails
				mockDB.On("CreateSpace", ctx, mock.AnythingOfType("db.CreateSpaceParams")).Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "failed to create space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.CreateSpace(ctx, tt.spaceName, tt.description)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.spaceName, result.Name)
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

func TestSpaceService_GetSpace(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewSpaceService(mockDB)

	tests := []struct {
		name          string
		spaceID uuid.UUID
		userID uuid.UUID
		isAdmin       bool
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:    "Successfully get space as admin",
			spaceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin: true,
			mockSetup: func() {
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Test Space", "A test space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(space, nil)
			},
			shouldSucceed: true,
		},
		{
			name:    "Successfully get space with permission",
			spaceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin: false,
			mockSetup: func() {
				mockDB.On("HasSpacePermission", ctx, mock.Anything).Return(true, nil)
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Test Space", "A test space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(space, nil)
			},
			shouldSucceed: true,
		},
		{
			name:    "Space not found",
			spaceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin: true,
			mockSetup: func() {
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "space not found",
		},
		{
			name:    "Access denied - no permission",
			spaceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin: false,
			mockSetup: func() {
				mockDB.On("HasSpacePermission", ctx, mock.Anything).Return(false, nil)
			},
			expectedError: "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetSpace(ctx, tt.spaceID, tt.userID, tt.isAdmin)

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

func TestSpaceService_GetAllSpaces(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewSpaceService(mockDB)

	tests := []struct {
		name          string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedLen   int
	}{
		{
			name: "Successfully get all spaces",
			mockSetup: func() {
				spaces := []db.ClaimctlSpace{
					testutils.CreateTestSpace(testutils.TestUUID(1), "Space 1", "First space"),
					testutils.CreateTestSpace(testutils.TestUUID(2), "Space 2", "Second space"),
				}
				mockDB.On("ListSpaces", ctx).Return(spaces, nil)
			},
			shouldSucceed: true,
			expectedLen:   2,
		},
		{
			name: "Database error getting spaces",
			mockSetup: func() {
				mockDB.On("ListSpaces", ctx).Return([]db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "failed to list spaces",
		},
		{
			name: "No spaces found",
			mockSetup: func() {
				mockDB.On("ListSpaces", ctx).Return([]db.ClaimctlSpace{}, nil)
			},
			shouldSucceed: true,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetAllSpaces(ctx)

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
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestSpaceService_UpdateSpace(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewSpaceService(mockDB)

	existingSpace := testutils.CreateTestSpace(testutils.TestUUID(1), "Old Name", "Old description")

	tests := []struct {
		name          string
		spaceID uuid.UUID
		newName       string
		description   string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:        "Successful space update",
			spaceID: testutils.TestUUID(1),
			newName:     "New Name",
			description: "New description",
			mockSetup: func() {
				// Check if space exists
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(existingSpace, nil)
				// Check name uniqueness (different name)
				mockDB.On("GetSpaceByName", ctx, "New Name").Return(db.ClaimctlSpace{}, assert.AnError)
				// Update space
				updatedSpace := testutils.CreateTestSpace(testutils.TestUUID(1), "New Name", "New description")
				mockDB.On("UpdateSpace", ctx, mock.AnythingOfType("db.UpdateSpaceParams")).Return(updatedSpace, nil)
			},
			shouldSucceed: true,
		},
		{
			name:        "Successful space update with same name",
			spaceID: testutils.TestUUID(1),
			newName:     "Old Name", // Same as existing
			description: "New description",
			mockSetup: func() {
				// Check if space exists
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(existingSpace, nil)
				// Check name uniqueness (same name)
				sameSpace := testutils.CreateTestSpace(testutils.TestUUID(1), "Old Name", "Old description")
				mockDB.On("GetSpaceByName", ctx, "Old Name").Return(sameSpace, nil)
				// Update space
				updatedSpace := testutils.CreateTestSpace(testutils.TestUUID(1), "Old Name", "New description")
				mockDB.On("UpdateSpace", ctx, mock.AnythingOfType("db.UpdateSpaceParams")).Return(updatedSpace, nil)
			},
			shouldSucceed: true,
		},
		{
			name:        "Space not found",
			spaceID: testutils.TestUUID(999),
			newName:     "New Name",
			description: "New description",
			mockSetup: func() {
				mockDB.On("GetSpace", ctx, testutils.TestUUID(999)).Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "space not found",
		},
		{
			name:        "Space name already taken by different space",
			spaceID: testutils.TestUUID(1),
			newName:     "Taken Name",
			description: "New description",
			mockSetup: func() {
				// Check if space exists
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(existingSpace, nil)
				// Check name uniqueness - taken by different space
				takenSpace := testutils.CreateTestSpace(testutils.TestUUID(2), "Taken Name", "Another space")
				mockDB.On("GetSpaceByName", ctx, "Taken Name").Return(takenSpace, nil)
			},
			expectedError: "space name 'Taken Name' already taken",
		},
		{
			name:        "Database error updating space",
			spaceID: testutils.TestUUID(1),
			newName:     "New Name",
			description: "New description",
			mockSetup: func() {
				// Check if space exists
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(existingSpace, nil)
				// Check name uniqueness
				mockDB.On("GetSpaceByName", ctx, "New Name").Return(db.ClaimctlSpace{}, assert.AnError)
				// Update space fails
				mockDB.On("UpdateSpace", ctx, mock.AnythingOfType("db.UpdateSpaceParams")).Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "failed to update space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.UpdateSpace(ctx, tt.spaceID, tt.newName, tt.description)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.newName, result.Name)
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

func TestSpaceService_DeleteSpace(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewSpaceService(mockDB)

	tests := []struct {
		name          string
		spaceID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:    "Successful space deletion",
			spaceID: testutils.TestUUID(1),
			mockSetup: func() {
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Regular Space", "A regular space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(space, nil)
				mockDB.On("DeleteSpace", ctx, testutils.TestUUID(1)).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name:    "Cannot delete Default Space",
			spaceID: testutils.TestUUID(1),
			mockSetup: func() {
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Default Space", "Default space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(space, nil)
			},
			expectedError: "cannot delete the Default Space",
		},
		{
			name:    "Space not found",
			spaceID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "space not found",
		},
		{
			name:    "Database error deleting space",
			spaceID: testutils.TestUUID(1),
			mockSetup: func() {
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Regular Space", "A regular space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(space, nil)
				mockDB.On("DeleteSpace", ctx, testutils.TestUUID(1)).Return(assert.AnError)
			},
			expectedError: "failed to delete space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			err := service.DeleteSpace(ctx, tt.spaceID)

			if tt.shouldSucceed {
				assert.NoError(t, err)
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
