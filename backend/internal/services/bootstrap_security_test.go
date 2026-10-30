package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestUserService_BootstrapPasswordRequiredOnlyForFirstAdmin(t *testing.T) {
	for _, tc := range []struct {
		name, password string
		admins         int64
		fail           bool
	}{
		{"missing first password", "", 0, true},
		{"weak first password", "password", 0, true},
		{"existing restart without password", "", 1, false},
		{"existing restart with weak password", "password", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ADMIN_PASSWORD", tc.password)
			ctx := context.Background()
			q := &testutils.MockQuerier{}
			q.On("CountAdminUsers", ctx).Return(tc.admins, nil).Once()
			err := NewUserService(q).EnsureAdminExists(ctx)
			if tc.fail {
				require.ErrorContains(t, err, "ADMIN_PASSWORD is required for bootstrap")
			} else {
				require.NoError(t, err)
			}
			q.AssertExpectations(t)
			require.Len(t, q.Calls, 1, "bootstrap must not create or inspect credentials on rejection or restart")
		})
	}
}
