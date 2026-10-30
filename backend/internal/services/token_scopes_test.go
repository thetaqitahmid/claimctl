package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestRemovedBackupScopesCannotBeIssued(t *testing.T) {
	for _, scope := range []string{"backup:read", "restore:write"} {
		t.Run(scope, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			id := testutils.TestUUID(1)
			q.On("FindUserById", mock.Anything, id).Return(db.ClaimctlUser{ID: id, Status: "active", Role: "admin"}, nil).Once()
			_, _, err := NewAPITokenService(q).GenerateToken(context.Background(), id, "removed scope", nil, TokenOptions{Scopes: []string{scope}})
			require.Error(t, err)
			q.AssertNotCalled(t, "CreateAPIToken", mock.Anything, mock.Anything)
			q.AssertExpectations(t)
		})
	}
}
