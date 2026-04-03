package services

import (
	"testing"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	types "github.com/thetaqitahmid/claimctl/internal/types"
)

func TestResourceService_CreateResource(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	tests := []struct {
		name          string
		createReq     CreateResourceRequest
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Successful resource creation",
			createReq: CreateResourceRequest{
				Name:    "Test Resource",
				Type:    "test-type",
				Labels:  types.JSONBArray{`{"tag": "test"}`},
				SpaceID: testutils.TestUUID(0), // Should use default space
			},
			mockSetup: func() {
				// Name uniqueness check
				mockDB.On("VerifyResourceNameIsUnique", ctx, "Test Resource").Return(int64(0), nil)
				// Default space lookup
				defaultSpace := testutils.CreateTestSpace(testutils.TestUUID(1), "Default Space", "Default space")
				mockDB.On("GetSpaceByName", ctx, "Default Space").Return(defaultSpace, nil)
				// Create resource
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("CreateNewResource", ctx, mock.AnythingOfType("db.CreateNewResourceParams")).Return(resource, nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Successful resource creation with specific space",
			createReq: CreateResourceRequest{
				Name:    "Test Resource",
				Type:    "test-type",
				Labels:  types.JSONBArray{`{"tag": "test"}`},
				SpaceID: testutils.TestUUID(2),
			},
			mockSetup: func() {
				// Name uniqueness check
				mockDB.On("VerifyResourceNameIsUnique", ctx, "Test Resource").Return(int64(0), nil)
				// Space lookup
				space := testutils.CreateTestSpace(testutils.TestUUID(2), "Test Space", "Test space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(2)).Return(space, nil)
				// Create resource
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(2))
				mockDB.On("CreateNewResource", ctx, mock.AnythingOfType("db.CreateNewResourceParams")).Return(resource, nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Invalid input values - empty name",
			createReq: CreateResourceRequest{
				Name:   "",
				Type:   "test-type",
				Labels: types.JSONBArray{`{"tag": "test"}`},
			},
			mockSetup:     func() {},
			expectedError: "invalid input values for creating resource",
		},
		{
			name: "Invalid input values - empty type",
			createReq: CreateResourceRequest{
				Name:   "Test Resource",
				Type:   "",
				Labels: types.JSONBArray{`{"tag": "test"}`},
			},
			mockSetup:     func() {},
			expectedError: "invalid input values for creating resource",
		},
		{
			name: "Invalid input values - empty labels",
			createReq: CreateResourceRequest{
				Name:   "Test Resource",
				Type:   "test-type",
				Labels: types.JSONBArray{},
			},
			mockSetup:     func() {},
			expectedError: "invalid input values for creating resource",
		},
		{
			name: "Resource name already exists",
			createReq: CreateResourceRequest{
				Name:    "Existing Resource",
				Type:    "test-type",
				Labels:  types.JSONBArray{`{"tag": "test"}`},
				SpaceID: testutils.TestUUID(1),
			},
			mockSetup: func() {
				// Name uniqueness check fails
				mockDB.On("VerifyResourceNameIsUnique", ctx, "Existing Resource").Return(int64(1), nil)
			},
			expectedError: "resource name 'Existing Resource' already exists",
		},
		{
			name: "Invalid space ID",
			createReq: CreateResourceRequest{
				Name:    "Test Resource",
				Type:    "test-type",
				Labels:  types.JSONBArray{`{"tag": "test"}`},
				SpaceID: testutils.TestUUID(999),
			},
			mockSetup: func() {
				// Name uniqueness check
				mockDB.On("VerifyResourceNameIsUnique", ctx, "Test Resource").Return(int64(0), nil)
				// Space lookup fails
				mockDB.On("GetSpace", ctx, testutils.TestUUID(999)).Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "invalid space id 00000000-0000-0000-0000-0000000003e7",
		},
		{
			name: "Default space not found",
			createReq: CreateResourceRequest{
				Name:    "Test Resource",
				Type:    "test-type",
				Labels:  types.JSONBArray{`{"tag": "test"}`},
				SpaceID: testutils.TestUUID(0), // Should use default space
			},
			mockSetup: func() {
				// Name uniqueness check
				mockDB.On("VerifyResourceNameIsUnique", ctx, "Test Resource").Return(int64(0), nil)
				// Default space lookup fails
				mockDB.On("GetSpaceByName", ctx, "Default Space").Return(db.ClaimctlSpace{}, assert.AnError)
			},
			expectedError: "failed to find Default Space",
		},
		{
			name: "Database error creating resource",
			createReq: CreateResourceRequest{
				Name:    "Test Resource",
				Type:    "test-type",
				Labels:  types.JSONBArray{`{"tag": "test"}`},
				SpaceID: testutils.TestUUID(1),
			},
			mockSetup: func() {
				// Name uniqueness check
				mockDB.On("VerifyResourceNameIsUnique", ctx, "Test Resource").Return(int64(0), nil)
				// Space lookup
				space := testutils.CreateTestSpace(testutils.TestUUID(1), "Test Space", "Test space")
				mockDB.On("GetSpace", ctx, testutils.TestUUID(1)).Return(space, nil)
				// Create resource fails
				mockDB.On("CreateNewResource", ctx, mock.AnythingOfType("db.CreateNewResourceParams")).Return(db.ClaimctlResource{}, assert.AnError)
			},
			expectedError: "failed to create new resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.CreateResource(ctx, tt.createReq)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.createReq.Name, result.Name)
				assert.Equal(t, tt.createReq.Type, result.Type)
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

func TestResourceService_GetResource(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	tests := []struct {
		name          string
		resourceID uuid.UUID
		userID uuid.UUID
		isAdmin       bool
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:       "Successfully get resource as admin",
			resourceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin:    true,
			mockSetup: func() {
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(resource, nil)
			},
			shouldSucceed: true,
		},
		{
			name:       "Successfully get resource with permission",
			resourceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin:    false,
			mockSetup: func() {
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(resource, nil)
				mockDB.On("HasSpacePermission", ctx, mock.Anything).Return(true, nil)
			},
			shouldSucceed: true,
		},
		{
			name:       "Resource not found",
			resourceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin:    true,
			mockSetup: func() {
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResource{}, assert.AnError)
			},
			expectedError: "resource with ID 00000000-0000-0000-0000-000000000001 not found",
		},
		{
			name:       "Access denied - not admin no permission",
			resourceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin:    false,
			mockSetup: func() {
				resource := testutils.CreateTestResource(testutils.TestUUID(1), "Test Resource", "test-type", testutils.TestUUID(1))
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(resource, nil)
				mockDB.On("HasSpacePermission", ctx, mock.Anything).Return(false, nil)
			},
			expectedError: "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetResource(ctx, tt.resourceID, tt.userID, tt.isAdmin)

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

func TestResourceService_GetAllResources(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	tests := []struct {
		name          string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedLen   int
	}{
		{
			name: "Successfully get all resources",
			mockSetup: func() {
				resources := []db.ClaimctlResource{
					testutils.CreateTestResource(testutils.TestUUID(1), "Resource 1", "type1", testutils.TestUUID(1)),
					testutils.CreateTestResource(testutils.TestUUID(2), "Resource 2", "type2", testutils.TestUUID(1)),
				}
				mockDB.On("FindAllResources", ctx).Return(resources, nil)
			},
			shouldSucceed: true,
			expectedLen:   2,
		},
		{
			name: "Database error getting resources",
			mockSetup: func() {
				mockDB.On("FindAllResources", ctx).Return([]db.ClaimctlResource{}, assert.AnError)
			},
			expectedError: "failed to retrieve resources",
		},
		{
			name: "No resources found",
			mockSetup: func() {
				mockDB.On("FindAllResources", ctx).Return([]db.ClaimctlResource{}, nil)
			},
			shouldSucceed: true,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetAllResources(ctx, nil)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedLen >= 0 {
					assert.Len(t, *result, tt.expectedLen)
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

func TestResourceService_UpdateResource(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	existingResource := testutils.CreateTestResource(testutils.TestUUID(1), "Old Name", "old-type", testutils.TestUUID(1))

	tests := []struct {
		name          string
		updateReq     UpdateResourceRequest
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Successful resource update",
			updateReq: UpdateResourceRequest{
				ID: testutils.TestUUID(1),
				Name:   "New Name",
				Type:   "new-type",
				Labels: types.JSONBArray{`{"tag": "new"}`},
			},
			mockSetup: func() {
				// Get existing resource
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(existingResource, nil)
				// Update resource
				mockDB.On("UpdateResourceById", ctx, mock.AnythingOfType("db.UpdateResourceByIdParams")).Return(nil)
				// Get updated resource
				updatedResource := testutils.CreateTestResource(testutils.TestUUID(1), "New Name", "new-type", testutils.TestUUID(1))
				updatedResource.Labels = types.JSONBArray{`{"tag": "new"}`}
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(updatedResource, nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Resource not found",
			updateReq: UpdateResourceRequest{
				ID: testutils.TestUUID(999),
				Name: "New Name",
			},
			mockSetup: func() {
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(999)).Return(db.ClaimctlResource{}, assert.AnError)
			},
			expectedError: "failed to retrieve updated resource with ID 00000000-0000-0000-0000-0000000003e7",
		},
		{
			name: "Database error updating resource",
			updateReq: UpdateResourceRequest{
				ID: testutils.TestUUID(1),
				Name: "New Name",
			},
			mockSetup: func() {
				// Get existing resource
				mockDB.On("FindResourceById", ctx, testutils.TestUUID(1)).Return(existingResource, nil)
				// Update resource fails
				mockDB.On("UpdateResourceById", ctx, mock.AnythingOfType("db.UpdateResourceByIdParams")).Return(assert.AnError)
			},
			expectedError: "failed to update resource with ID 00000000-0000-0000-0000-000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.UpdateResource(ctx, tt.updateReq)

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

func TestResourceService_DeleteResource(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	tests := []struct {
		name          string
		resourceID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:       "Successful resource deletion",
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("DeleteResourceById", ctx, testutils.TestUUID(1)).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name:       "Database error deleting resource",
			resourceID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("DeleteResourceById", ctx, testutils.TestUUID(1)).Return(assert.AnError)
			},
			expectedError: "failed to delete resource with ID 00000000-0000-0000-0000-000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			err := service.DeleteResource(ctx, tt.resourceID)

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

func TestResourceService_GetResourceWithStatus(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	tests := []struct {
		name          string
		resourceID uuid.UUID
		userID uuid.UUID
		isAdmin       bool
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedStats *ResourceWithStatus
	}{
		{
			name:       "Successfully get resource with status as admin",
			resourceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin:    true,
			mockSetup: func() {
				resourceStatus := db.GetResourceReservationStatusRow{
					ID: testutils.TestUUID(1),
					Name:               "Test Resource",
					Type:               "test-type",
					Labels:             types.JSONBArray{`{"tag": "test"}`},
					CreatedAt:          pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
					UpdatedAt:          pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
					SpaceID: testutils.TestUUID(1),
					ActiveReservations: 2,
					QueueLength:        3,
					NextUserID: testutils.TestUUID(5),
					NextQueuePosition:  4,
				}
				mockDB.On("GetResourceReservationStatus", ctx, testutils.TestUUID(1)).Return(resourceStatus, nil)
				mockDB.On("GetHealthConfig", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResourceHealthConfig{}, assert.AnError)
				mockDB.On("GetHealthStatus", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResourceHealthStatus{}, assert.AnError)
			},
			shouldSucceed: true,
			expectedStats: &ResourceWithStatus{
				ActiveReservations: 2,
				QueueLength:        3,
				NextUserID: testutils.TestUUID(5),
				NextQueuePosition:  4,
			},
		},
		{
			name:       "Resource not found",
			resourceID: testutils.TestUUID(1),
			userID: testutils.TestUUID(1),
			isAdmin:    true,
			mockSetup: func() {
				mockDB.On("GetResourceReservationStatus", ctx, testutils.TestUUID(1)).Return(db.GetResourceReservationStatusRow{}, assert.AnError)
			},
			expectedError: "resource with ID 00000000-0000-0000-0000-000000000001 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetResourceWithStatus(ctx, tt.resourceID, tt.userID, tt.isAdmin)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedStats != nil {
					assert.Equal(t, tt.expectedStats.ActiveReservations, result.ActiveReservations)
					assert.Equal(t, tt.expectedStats.QueueLength, result.QueueLength)
					assert.Equal(t, tt.expectedStats.NextUserID, result.NextUserID)
					assert.Equal(t, tt.expectedStats.NextQueuePosition, result.NextQueuePosition)
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

func TestResourceService_GetAllResourcesWithStatus(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	mockRealtime := &testutils.MockRealtimeService{}
	service := NewResourceService(mockDB, mockRealtime)

	tests := []struct {
		name          string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedLen   int
	}{
		{
			name: "Successfully get all resources with status",
			mockSetup: func() {
				resourcesStatus := []db.GetAllResourcesWithReservationStatusRow{
					{
						ID: testutils.TestUUID(1),
						Name:               "Resource 1",
						Type:               "type1",
						Labels:             types.JSONBArray{`{"tag": "test"}`},
						CreatedAt:          pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						UpdatedAt:          pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						SpaceID: testutils.TestUUID(1),
						ActiveReservations: 1,
						QueueLength:        2,
						NextUserID: testutils.TestUUID(3),
						NextQueuePosition:  4,
					},
					{
						ID: testutils.TestUUID(2),
						Name:               "Resource 2",
						Type:               "type2",
						Labels:             types.JSONBArray{`{"tag": "test"}`},
						CreatedAt:          pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						UpdatedAt:          pgtype.Int8{Int64: testutils.FixedTimestamp(), Valid: true},
						SpaceID: testutils.TestUUID(1),
						ActiveReservations: 0,
						QueueLength:        1,
						NextUserID: testutils.TestUUID(5),
						NextQueuePosition:  6,
					},
				}
				mockDB.On("GetAllResourcesWithReservationStatus", ctx).Return(resourcesStatus, nil)
				mockDB.On("GetHealthConfig", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResourceHealthConfig{}, assert.AnError)
				mockDB.On("GetHealthStatus", ctx, testutils.TestUUID(1)).Return(db.ClaimctlResourceHealthStatus{}, assert.AnError)
				mockDB.On("GetHealthConfig", ctx, testutils.TestUUID(2)).Return(db.ClaimctlResourceHealthConfig{}, assert.AnError)
				mockDB.On("GetHealthStatus", ctx, testutils.TestUUID(2)).Return(db.ClaimctlResourceHealthStatus{}, assert.AnError)
			},
			shouldSucceed: true,
			expectedLen:   2,
		},
		{
			name: "Database error getting resources with status",
			mockSetup: func() {
				mockDB.On("GetAllResourcesWithReservationStatus", ctx).Return([]db.GetAllResourcesWithReservationStatusRow{}, assert.AnError)
			},
			expectedError: "failed to retrieve resources with status",
		},
		{
			name: "No resources found",
			mockSetup: func() {
				mockDB.On("GetAllResourcesWithReservationStatus", ctx).Return([]db.GetAllResourcesWithReservationStatusRow{}, nil)
			},
			shouldSucceed: true,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetAllResourcesWithStatus(ctx, nil)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedLen >= 0 {
					assert.Len(t, *result, tt.expectedLen)
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
