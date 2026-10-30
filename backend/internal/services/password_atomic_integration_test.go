package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestUserService_PasswordAndRefreshRevocationRollbackRealDatabase(t *testing.T) {
	for _, adminReset := range []bool{false, true} {
		name := "self change"
		if adminReset {
			name = "admin reset"
		}
		t.Run(name, func(t *testing.T) {
			pool := testutils.SecurityTestDatabase(t)
			store := db.NewStore(pool)
			ctx := context.Background()
			id, password := seedRefreshAccount(t, pool)
			raw, err := NewRefreshTokenService(store).Issue(ctx, id, CredentialVersion(password))
			require.NoError(t, err)
			_, err = pool.Exec(ctx, `CREATE FUNCTION claimctl.reject_refresh_revoke() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected revoke failure'; END $$;
   CREATE TRIGGER reject_refresh_revoke BEFORE UPDATE ON claimctl.refresh_tokens FOR EACH ROW EXECUTE FUNCTION claimctl.reject_refresh_revoke()`)
			require.NoError(t, err)
			users := NewUserService(store)
			if adminReset {
				newPassword := "NewPass2!"
				err = users.UpdateUser(ctx, &UpdateUserRequest{ID: &id, Password: &newPassword})
			} else {
				err = users.UpdatePassword(ctx, id, "OldPass1!", "NewPass2!")
			}
			require.Error(t, err)
			current, err := store.FindUserById(ctx, id)
			require.NoError(t, err)
			require.Equal(t, password, current.Password)
			record, err := store.GetRefreshTokenByHash(ctx, hashToken(raw))
			require.NoError(t, err)
			require.False(t, record.Revoked)
		})
	}
}
