package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestUserService_NeedsSetup(t *testing.T) {
	ctx := context.Background()

	t.Run("returns true without admins", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		q.On("CountAdminUsers", ctx).Return(int64(0), nil)
		needsSetup, err := NewUserService(q).NeedsSetup(ctx)
		require.NoError(t, err)
		assert.True(t, needsSetup)
	})

	t.Run("returns false with an admin", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		q.On("CountAdminUsers", ctx).Return(int64(1), nil)
		needsSetup, err := NewUserService(q).NeedsSetup(ctx)
		require.NoError(t, err)
		assert.False(t, needsSetup)
	})
}

func TestUserService_CreateInitialAdmin(t *testing.T) {
	ctx := context.Background()
	validReq := InitialAdminRequest{
		Name:     "First Admin",
		Email:    "first@example.com",
		Password: "Valid1!pass",
	}

	t.Run("creates the first admin", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		created := testutils.CreateTestUser(testutils.TestUUID(1), validReq.Email, validReq.Name, true)

		q.On("CountAdminUsers", ctx).Return(int64(0), nil).Once()
		q.On("VerifyUserEmailIsUnique", ctx, validReq.Email).Return(int64(0), nil).Once()
		q.On("CreateUser", ctx, mock.AnythingOfType("db.CreateUserParams")).Return(nil).Once()
		q.On("FindUserByEmail", ctx, validReq.Email).Return(created, nil).Once()

		user, err := NewUserService(q).CreateInitialAdmin(ctx, validReq)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, validReq.Email, user.Email)
		assert.Equal(t, "admin", user.Role)
		q.AssertExpectations(t)
	})

	t.Run("rejects when an admin already exists", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		q.On("CountAdminUsers", ctx).Return(int64(1), nil).Once()

		_, err := NewUserService(q).CreateInitialAdmin(ctx, validReq)
		require.ErrorIs(t, err, ErrSetupAlreadyCompleted)
		q.AssertExpectations(t)
		require.Len(t, q.Calls, 1)
	})

	t.Run("rejects an email already in use", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		q.On("CountAdminUsers", ctx).Return(int64(0), nil).Once()
		q.On("VerifyUserEmailIsUnique", ctx, validReq.Email).Return(int64(1), nil).Once()

		_, err := NewUserService(q).CreateInitialAdmin(ctx, validReq)
		require.ErrorIs(t, err, ErrSetupEmailInUse)
		q.AssertExpectations(t)
	})

	t.Run("rejects a weak password before touching the database", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		req := validReq
		req.Password = "password"

		_, err := NewUserService(q).CreateInitialAdmin(ctx, req)
		require.ErrorIs(t, err, ErrInvalidSetupInput)
		require.Empty(t, q.Calls)
	})

	t.Run("returns the created admin parameters with the admin role", func(t *testing.T) {
		q := &testutils.MockQuerier{}
		created := testutils.CreateTestUser(testutils.TestUUID(2), validReq.Email, validReq.Name, true)

		var captured db.CreateUserParams
		q.On("CountAdminUsers", ctx).Return(int64(0), nil).Once()
		q.On("VerifyUserEmailIsUnique", ctx, validReq.Email).Return(int64(0), nil).Once()
		q.On("CreateUser", ctx, mock.AnythingOfType("db.CreateUserParams")).Run(func(args mock.Arguments) {
			captured = args.Get(1).(db.CreateUserParams)
		}).Return(nil).Once()
		q.On("FindUserByEmail", ctx, validReq.Email).Return(created, nil).Once()

		_, err := NewUserService(q).CreateInitialAdmin(ctx, validReq)
		require.NoError(t, err)
		assert.Equal(t, "admin", captured.Role)
		assert.Equal(t, "active", captured.Status)
		assert.Equal(t, "local", captured.AuthProvider)
		assert.NotEqual(t, validReq.Password, captured.Password)
	})
}
