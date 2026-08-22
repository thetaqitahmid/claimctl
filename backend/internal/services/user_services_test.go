package services

import (
	"database/sql"
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

func TestUserService_Login(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	tests := []struct {
		name          string
		loginReq      LoginRequest
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Successful login",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
				mockDB.On("UpdateUserLastLogin", ctx, testutils.TestUUID(1)).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Empty email",
			loginReq: LoginRequest{
				Email:    "",
				Password: "password123",
			},
			mockSetup:     func() {},
			expectedError: "email and password must be provided",
		},
		{
			name: "Empty password",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			mockSetup:     func() {},
			expectedError: "email and password must be provided",
		},
		{
			name: "User not found",
			loginReq: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				mockDB.On("FindUserByEmail", ctx, "nonexistent@example.com").Return(db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "failed to retrieve the user with email nonexistent@example.com",
		},
		{
			name: "Inactive user",
			loginReq: LoginRequest{
				Email:    "inactive@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "inactive@example.com", "Inactive User", false)
				user.Status = "inactive"
				mockDB.On("FindUserByEmail", ctx, "inactive@example.com").Return(user, nil)
			},
			expectedError: "user inactive@example.com is inactive",
		},
		{
			name: "Invalid password",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
				mockDB.On("UpdateUserFailedLoginAttempts", ctx, mock.MatchedBy(func(arg db.UpdateUserFailedLoginAttemptsParams) bool {
					return !arg.LockedUntil.Valid && arg.ID == testutils.TestUUID(1)
				})).Return(int32(1), nil)
			},
			expectedError: "Invalid password",
		},
		{
			name: "Database error updating last login",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
				mockDB.On("UpdateUserLastLogin", ctx, testutils.TestUUID(1)).Return(assert.AnError)
			},
			expectedError: "failed to update last login for user test@example.com",
		},
		{
			name: "Account locked",
			loginReq: LoginRequest{
				Email:    "locked@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "locked@example.com", "Locked User", false)
				user.LockedUntil = pgtype.Timestamptz{Valid: true, Time: time.Now().Add(10 * time.Minute)}
				mockDB.On("FindUserByEmail", ctx, "locked@example.com").Return(user, nil)
			},
			expectedError: "account temporarily locked. Please try again later",
		},
		{
			name: "Failed login increments attempt count",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				user.FailedLoginAttempts = 1
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
				mockDB.On("UpdateUserFailedLoginAttempts", ctx, mock.MatchedBy(func(arg db.UpdateUserFailedLoginAttemptsParams) bool {
					return !arg.LockedUntil.Valid && arg.ID == testutils.TestUUID(1)
				})).Return(int32(2), nil)
			},
			expectedError: "Invalid password",
		},
		{
			name: "Failed login locks account after 3 attempts",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				user.FailedLoginAttempts = 2
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
				mockDB.On("UpdateUserFailedLoginAttempts", ctx, mock.MatchedBy(func(arg db.UpdateUserFailedLoginAttemptsParams) bool {
					return !arg.LockedUntil.Valid && arg.ID == testutils.TestUUID(1)
				})).Return(int32(3), nil).Once()
				mockDB.On("UpdateUserFailedLoginAttempts", ctx, mock.MatchedBy(func(arg db.UpdateUserFailedLoginAttemptsParams) bool {
					return arg.LockedUntil.Valid && arg.ID == testutils.TestUUID(1)
				})).Return(int32(3), nil).Once()
			},
			expectedError: "account temporarily locked due to too many failed attempts",
		},
		{
			name: "Successful login resets failed attempts",
			loginReq: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				user.FailedLoginAttempts = 2
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
				mockDB.On("ResetUserFailedLoginAttempts", ctx, testutils.TestUUID(1)).Return(nil)
				mockDB.On("UpdateUserLastLogin", ctx, testutils.TestUUID(1)).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name: "OIDC user cannot password login",
			loginReq: LoginRequest{
				Email:    "oidc@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "oidc@example.com", "OIDC User", false)
				user.AuthProvider = "oidc"
				mockDB.On("FindUserByEmail", ctx, "oidc@example.com").Return(user, nil)
			},
			expectedError: "Invalid password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.Login(ctx, tt.loginReq)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.loginReq.Email, result.Email)
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

func TestUserService_CreateUser(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	tests := []struct {
		name          string
		createReq     db.CreateUserParams
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Successful user creation",
			createReq: db.CreateUserParams{
				Email:    "newuser@example.com",
				Name:     "New User",
				Password: "Password123!",
			},
			mockSetup: func() {
				mockDB.On("VerifyUserEmailIsUnique", ctx, "newuser@example.com").Return(int64(0), nil)
				mockDB.On("CreateUser", ctx, mock.AnythingOfType("db.CreateUserParams")).Return(nil)
				user := testutils.CreateTestUser(testutils.TestUUID(1), "newuser@example.com", "New User", false)
				mockDB.On("FindUserByEmail", ctx, "newuser@example.com").Return(user, nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Empty email",
			createReq: db.CreateUserParams{
				Email:    "",
				Name:     "Test User",
				Password: "Password123!",
			},
			mockSetup:     func() {},
			expectedError: "all fields must be provided to create a user",
		},
		{
			name: "Empty name",
			createReq: db.CreateUserParams{
				Email:    "test@example.com",
				Name:     "",
				Password: "Password123!",
			},
			mockSetup:     func() {},
			expectedError: "all fields must be provided to create a user",
		},
		{
			name: "Empty password",
			createReq: db.CreateUserParams{
				Email:    "test@example.com",
				Name:     "Test User",
				Password: "",
			},
			mockSetup:     func() {},
			expectedError: "all fields must be provided to create a user",
		},
		{
			name: "Invalid email format",
			createReq: db.CreateUserParams{
				Email:    "invalid-email",
				Name:     "Test User",
				Password: "Password123!",
			},
			mockSetup:     func() {},
			expectedError: "Email must be a valid email address",
		},
		{
			name: "Email already exists",
			createReq: db.CreateUserParams{
				Email:    "existing@example.com",
				Name:     "Test User",
				Password: "Password123!",
			},
			mockSetup: func() {
				mockDB.On("VerifyUserEmailIsUnique", ctx, "existing@example.com").Return(int64(1), nil)
			},
			expectedError: "email existing@example.com is already in use",
		},
		{
			name: "Invalid password",
			createReq: db.CreateUserParams{
				Email:    "test@example.com",
				Name:     "Test User",
				Password: "weak",
			},
			mockSetup: func() {
				// This test should fail before email uniqueness check, but still need to set up mock
				// because the service method calls it before password validation
				mockDB.On("VerifyUserEmailIsUnique", ctx, "test@example.com").Return(int64(0), nil)
			},
			expectedError: "Password must be at least 8 characters long and contain uppercase, lowercase, and special character",
		},
		{
			name: "Database error creating user",
			createReq: db.CreateUserParams{
				Email:    "test@example.com",
				Name:     "Test User",
				Password: "Password123!",
			},
			mockSetup: func() {
				mockDB.On("VerifyUserEmailIsUnique", ctx, "test@example.com").Return(int64(0), nil)
				mockDB.On("CreateUser", ctx, mock.AnythingOfType("db.CreateUserParams")).Return(assert.AnError)
			},
			expectedError: "failed to create new user",
		},
		{
			name: "Default status applied",
			createReq: db.CreateUserParams{
				Email:    "test@example.com",
				Name:     "Test User",
				Password: "Password123!",
				Status:   "", // Empty status should default to "active"
			},
			mockSetup: func() {
				mockDB.On("VerifyUserEmailIsUnique", ctx, "test@example.com").Return(int64(0), nil)
				mockDB.On("CreateUser", ctx, mock.MatchedBy(func(params db.CreateUserParams) bool {
					return params.Status == "active"
				})).Return(nil)
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
			},
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.CreateUser(ctx, tt.createReq)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.createReq.Email, result.Email)
				assert.Equal(t, tt.createReq.Name, result.Name)
				assert.NotEqual(t, tt.createReq.Password, result.Password) // Password should be hashed
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

func TestUserService_UpdateUser(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	existingUser := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)

	tests := []struct {
		name          string
		updateReq     *UpdateUserRequest
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Successful user update",
			updateReq: &UpdateUserRequest{
				ID:       func() *uuid.UUID { u := testutils.TestUUID(1); return &u }(),
				Email:    &[]string{"updated@example.com"}[0],
				Name:     &[]string{"Updated Name"}[0],
				Password: &[]string{"NewPassword123!"}[0],

				Role:   &[]string{"admin"}[0],
				Status: &[]string{"active"}[0],
			},
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(existingUser, nil)
				mockDB.On("UpdateUserById", ctx, mock.AnythingOfType("db.UpdateUserByIdParams")).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name: "User not found",
			updateReq: &UpdateUserRequest{
				ID:    func() *uuid.UUID { u := testutils.TestUUID(999); return &u }(),
				Email: &[]string{"test@example.com"}[0],
			},
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(999)).Return(db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "failed to retrieve user with ID 00000000-0000-0000-0000-0000000003e7",
		},
		{
			name: "Invalid role",
			updateReq: &UpdateUserRequest{
				ID:   func() *uuid.UUID { u := testutils.TestUUID(1); return &u }(),
				Role: &[]string{"invalid_role"}[0],
			},
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(existingUser, nil)
			},
			expectedError: "invalid role",
		},
		{
			name: "Invalid new password",
			updateReq: &UpdateUserRequest{
				ID:       func() *uuid.UUID { u := testutils.TestUUID(1); return &u }(),
				Password: &[]string{"weak"}[0],
			},
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(existingUser, nil)
			},
			expectedError: "Password must be at least 8 characters long and contain uppercase, lowercase, and special character",
		},
		{
			name: "Database error updating user",
			updateReq: &UpdateUserRequest{
				ID:   func() *uuid.UUID { u := testutils.TestUUID(1); return &u }(),
				Name: &[]string{"Updated Name"}[0],
			},
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(existingUser, nil)
				mockDB.On("UpdateUserById", ctx, mock.AnythingOfType("db.UpdateUserByIdParams")).Return(assert.AnError)
			},
			expectedError: "failed to update user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			err := service.UpdateUser(ctx, tt.updateReq)

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

func TestUserService_EnsureAdminExists(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	tests := []struct {
		name          string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name: "Admin already exists",
			mockSetup: func() {
				mockDB.On("CountAdminUsers", ctx).Return(int64(1), nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Create default admin successfully",
			mockSetup: func() {
				mockDB.On("CountAdminUsers", ctx).Return(int64(0), nil)
				mockDB.On("VerifyUserEmailIsUnique", ctx, "admin@claimctl.com").Return(int64(0), nil)
				mockDB.On("CreateUser", ctx, mock.AnythingOfType("db.CreateUserParams")).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name: "Admin email already exists",
			mockSetup: func() {
				mockDB.On("CountAdminUsers", ctx).Return(int64(0), nil)
				mockDB.On("VerifyUserEmailIsUnique", ctx, "admin@claimctl.com").Return(int64(1), nil)
			},
			shouldSucceed: true, // Should succeed because user already exists
		},
		{
			name: "Database error counting admins",
			mockSetup: func() {
				mockDB.On("CountAdminUsers", ctx).Return(int64(0), assert.AnError)
			},
			expectedError: "failed to count admin users",
		},
		{
			name: "Database error creating admin",
			mockSetup: func() {
				mockDB.On("CountAdminUsers", ctx).Return(int64(0), nil)
				mockDB.On("VerifyUserEmailIsUnique", ctx, "admin@claimctl.com").Return(int64(0), nil)
				mockDB.On("CreateUser", ctx, mock.AnythingOfType("db.CreateUserParams")).Return(assert.AnError)
			},
			expectedError: "failed to create default admin user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			err := service.EnsureAdminExists(ctx)

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

func TestUserService_GetUser(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)

	tests := []struct {
		name          string
		email         string
		id uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:  "Get user by email",
			email: "test@example.com",
			id:    uuid.Nil,
			mockSetup: func() {
				mockDB.On("FindUserByEmail", ctx, "test@example.com").Return(user, nil)
			},
			shouldSucceed: true,
		},
		{
			name:  "Get user by ID",
			email: "",
			id:    testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(user, nil)
			},
			shouldSucceed: true,
		},
		{
			name:          "Neither email nor ID provided",
			email:         "",
			id:            uuid.Nil,
			mockSetup:     func() {},
			expectedError: "either email or id must be provided to find a user",
		},
		{
			name:  "User not found by email",
			email: "nonexistent@example.com",
			id:    uuid.Nil,
			mockSetup: func() {
				mockDB.On("FindUserByEmail", ctx, "nonexistent@example.com").Return(db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "assert.AnError",
		},
		{
			name:  "User not found by ID",
			email: "",
			id:    testutils.TestUUID(999),
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(999)).Return(db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "assert.AnError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetUser(ctx, tt.email, tt.id)

			if tt.shouldSucceed {
				assert.NoError(t, err)
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

func TestUserService_DeleteUser(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	tests := []struct {
		name          string
		userID uuid.UUID
		mockSetup     func()
		expectedError string
		shouldSucceed bool
	}{
		{
			name:   "Successful user deletion",
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("DeleteUserById", ctx, testutils.TestUUID(1)).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name:   "Database error deleting user",
			userID: testutils.TestUUID(1),
			mockSetup: func() {
				mockDB.On("DeleteUserById", ctx, testutils.TestUUID(1)).Return(assert.AnError)
			},
			expectedError: "failed to delete user with ID 00000000-0000-0000-0000-000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			err := service.DeleteUser(ctx, tt.userID)

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

func TestUserService_GetUsers(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	tests := []struct {
		name          string
		mockSetup     func()
		expectedError string
		shouldSucceed bool
		expectedLen   int
	}{
		{
			name: "Successfully get all users",
			mockSetup: func() {
				users := []db.ClaimctlUser{
					testutils.CreateTestUser(testutils.TestUUID(1), "user1@example.com", "User 1", false),
					testutils.CreateTestUser(testutils.TestUUID(2), "user2@example.com", "User 2", true),
				}
				mockDB.On("FindAllUsers", ctx).Return(users, nil)
			},
			shouldSucceed: true,
			expectedLen:   2,
		},
		{
			name: "Database error getting users",
			mockSetup: func() {
				mockDB.On("FindAllUsers", ctx).Return([]db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "assert.AnError",
		},
		{
			name: "No users found",
			mockSetup: func() {
				mockDB.On("FindAllUsers", ctx).Return([]db.ClaimctlUser{}, nil)
			},
			shouldSucceed: true,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil // Reset mock
			tt.mockSetup()

			result, err := service.GetUsers(ctx)

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

func TestUserService_UpdatePassword(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	tests := []struct {
		name            string
		userID          uuid.UUID
		currentPassword string
		newPassword     string
		mockSetup       func()
		expectedError   string
		shouldSucceed   bool
	}{
		{
			name:            "Local user - successful password change",
			userID:          testutils.TestUUID(1),
			currentPassword: "OldPass1!",
			newPassword:     "NewPass1!",
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(1), "test@example.com", "Test User", false)
				mockDB.On("FindUserById", ctx, testutils.TestUUID(1)).Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(1)).Return("$2a$10$O0ifkKjMGoT0hBq2kMUhAuwsBLfGd8/y3J6Q0yqylfs6bi8Zs8xZ6", nil)
				mockDB.On("UpdateUserPassword", ctx, mock.MatchedBy(func(arg db.UpdateUserPasswordParams) bool {
					return arg.ID == testutils.TestUUID(1)
				})).Return(nil)
			},
			shouldSucceed: true,
		},
		{
			name:            "OIDC user - password change blocked",
			userID:          testutils.TestUUID(3),
			currentPassword: "OldPass1!",
			newPassword:     "NewPass1!",
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(3), "oidc@example.com", "OIDC User", false)
				user.AuthProvider = "oidc"
				mockDB.On("FindUserById", ctx, testutils.TestUUID(3)).Return(user, nil)
			},
			expectedError: "password change is not available for oidc users",
		},
		{
			name:            "User not found",
			userID:          testutils.TestUUID(4),
			currentPassword: "OldPass1!",
			newPassword:     "NewPass1!",
			mockSetup: func() {
				mockDB.On("FindUserById", ctx, testutils.TestUUID(4)).Return(db.ClaimctlUser{}, assert.AnError)
			},
			expectedError: "user not found",
		},
		{
			name:            "Incorrect current password",
			userID:          testutils.TestUUID(5),
			currentPassword: "WrongPass1!",
			newPassword:     "NewPass1!",
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(5), "test2@example.com", "Test User 2", false)
				mockDB.On("FindUserById", ctx, testutils.TestUUID(5)).Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(5)).Return("$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", nil)
			},
			expectedError: "incorrect current password",
		},
		{
			name:            "Weak new password",
			userID:          testutils.TestUUID(6),
			currentPassword: "OldPass1!",
			newPassword:     "weak",
			mockSetup: func() {
				user := testutils.CreateTestUser(testutils.TestUUID(6), "test3@example.com", "Test User 3", false)
				mockDB.On("FindUserById", ctx, testutils.TestUUID(6)).Return(user, nil)
				mockDB.On("GetPasswordById", ctx, testutils.TestUUID(6)).Return("$2a$10$O0ifkKjMGoT0hBq2kMUhAuwsBLfGd8/y3J6Q0yqylfs6bi8Zs8xZ6", nil)
			},
			expectedError: "New password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.ExpectedCalls = nil
			tt.mockSetup()

			err := service.UpdatePassword(ctx, tt.userID, tt.currentPassword, tt.newPassword)

			if tt.shouldSucceed {
				require.NoError(t, err)
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

func TestUserService_LoginOIDC(t *testing.T) {
	ctx := testutils.TestContext()
	mockDB := &testutils.MockQuerier{}
	service := NewUserService(mockDB)

	subText := pgtype.Text{String: "sub-123", Valid: true}

	t.Run("creates new user by subject", func(t *testing.T) {
		mockDB.ExpectedCalls = nil
		mockDB.On("FindUserByOIDCSubject", ctx, subText).Return(db.ClaimctlUser{}, sql.ErrNoRows).Once()
		mockDB.On("FindUserByEmail", ctx, "new@example.com").Return(db.ClaimctlUser{}, sql.ErrNoRows).Once()
		mockDB.On("CreateUser", ctx, mock.MatchedBy(func(p db.CreateUserParams) bool {
			return p.Email == "new@example.com" && p.AuthProvider == "oidc" &&
				p.OidcSubject.String == "sub-123" && p.Role == "user"
		})).Return(nil).Once()

		created := testutils.CreateTestUser(testutils.TestUUID(10), "new@example.com", "New User", false)
		created.AuthProvider = "oidc"
		created.OidcSubject = subText
		mockDB.On("FindUserByOIDCSubject", ctx, subText).Return(created, nil).Once()

		user, err := service.LoginOIDC(ctx, OIDCLoginRequest{
			Subject: "sub-123",
			Email:   "new@example.com",
			Name:    "New User",
		})
		require.NoError(t, err)
		assert.Equal(t, "new@example.com", user.Email)
		mockDB.AssertExpectations(t)
	})

	t.Run("promotes to admin when IsAdmin", func(t *testing.T) {
		mockDB.ExpectedCalls = nil
		existing := testutils.CreateTestUser(testutils.TestUUID(11), "adminish@example.com", "User", false)
		existing.AuthProvider = "oidc"
		existing.OidcSubject = subText
		mockDB.On("FindUserByOIDCSubject", ctx, subText).Return(existing, nil).Once()
		mockDB.On("UpdateOIDCUser", ctx, mock.MatchedBy(func(p db.UpdateOIDCUserParams) bool {
			return p.ID == testutils.TestUUID(11) && p.Role == "admin" && p.OidcSubject.String == "sub-123"
		})).Return(nil).Once()

		user, err := service.LoginOIDC(ctx, OIDCLoginRequest{
			Subject: "sub-123",
			Email:   "adminish@example.com",
			Name:    "User",
			IsAdmin: true,
		})
		require.NoError(t, err)
		assert.Equal(t, "admin", user.Role)
		mockDB.AssertExpectations(t)
	})

	t.Run("binds subject for legacy oidc user by email", func(t *testing.T) {
		mockDB.ExpectedCalls = nil
		legacySub := pgtype.Text{String: "legacy-sub", Valid: true}
		mockDB.On("FindUserByOIDCSubject", ctx, legacySub).Return(db.ClaimctlUser{}, sql.ErrNoRows).Once()

		legacy := testutils.CreateTestUser(testutils.TestUUID(12), "legacy@example.com", "Legacy", false)
		legacy.AuthProvider = "oidc"
		mockDB.On("FindUserByEmail", ctx, "legacy@example.com").Return(legacy, nil).Once()
		mockDB.On("UpdateOIDCUser", ctx, mock.MatchedBy(func(p db.UpdateOIDCUserParams) bool {
			return p.ID == testutils.TestUUID(12) && p.OidcSubject.String == "legacy-sub"
		})).Return(nil).Once()

		user, err := service.LoginOIDC(ctx, OIDCLoginRequest{
			Subject: "legacy-sub",
			Email:   "legacy@example.com",
			Name:    "Legacy",
		})
		require.NoError(t, err)
		assert.Equal(t, "legacy@example.com", user.Email)
		mockDB.AssertExpectations(t)
	})

	t.Run("rejects local account email collision", func(t *testing.T) {
		mockDB.ExpectedCalls = nil
		collSub := pgtype.Text{String: "coll-sub", Valid: true}
		mockDB.On("FindUserByOIDCSubject", ctx, collSub).Return(db.ClaimctlUser{}, sql.ErrNoRows).Once()
		local := testutils.CreateTestUser(testutils.TestUUID(13), "local@example.com", "Local", false)
		local.AuthProvider = "local"
		mockDB.On("FindUserByEmail", ctx, "local@example.com").Return(local, nil).Once()

		_, err := service.LoginOIDC(ctx, OIDCLoginRequest{
			Subject: "coll-sub",
			Email:   "local@example.com",
			Name:    "Local",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different login method")
		mockDB.AssertExpectations(t)
	})

	t.Run("rejects inactive user", func(t *testing.T) {
		mockDB.ExpectedCalls = nil
		inactive := testutils.CreateTestUser(testutils.TestUUID(14), "bad@example.com", "Bad", false)
		inactive.AuthProvider = "oidc"
		inactive.OidcSubject = subText
		inactive.Status = "inactive"
		mockDB.On("FindUserByOIDCSubject", ctx, subText).Return(inactive, nil).Once()

		_, err := service.LoginOIDC(ctx, OIDCLoginRequest{
			Subject: "sub-123",
			Email:   "bad@example.com",
			Name:    "Bad",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
		mockDB.AssertExpectations(t)
	})

	t.Run("propagates DB error on subject lookup", func(t *testing.T) {
		mockDB.ExpectedCalls = nil
		mockDB.On("FindUserByOIDCSubject", ctx, subText).Return(db.ClaimctlUser{}, assert.AnError).Once()

		_, err := service.LoginOIDC(ctx, OIDCLoginRequest{
			Subject: "sub-123",
			Email:   "x@example.com",
			Name:    "X",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lookup OIDC user by subject")
		mockDB.AssertExpectations(t)
	})

	t.Run("rejects missing subject", func(t *testing.T) {
		_, err := service.LoginOIDC(ctx, OIDCLoginRequest{Email: "x@example.com"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subject is required")
	})
}
