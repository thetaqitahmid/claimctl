package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func seedRefreshAccount(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("OldPass1!"), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), "INSERT INTO claimctl.users (id,email,name,password) VALUES ($1,$2,'Refresh Test',$3)", id, id.String()+"@example.invalid", string(hash))
	require.NoError(t, err)
	return id, string(hash)
}

func TestRefreshTokenService_ConcurrentRotationRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	store := db.NewStore(pool)
	id, password := seedRefreshAccount(t, pool)
	svc := NewRefreshTokenService(store)
	ctx := context.Background()
	raw, err := svc.Issue(ctx, id, CredentialVersion(password))
	require.NoError(t, err)
	start := make(chan struct{})
	results := make(chan error, 12)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, _, err := svc.Rotate(ctx, raw); results <- err }()
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, ErrRefreshTokenRevoked)
		}
	}
	require.Equal(t, 1, successes)
	var total, live int
	require.NoError(t, pool.QueryRow(ctx, "SELECT COUNT(*),COUNT(*) FILTER (WHERE NOT revoked) FROM claimctl.refresh_tokens WHERE user_id=$1", id).Scan(&total, &live))
	require.Equal(t, 2, total, "only one successor and the original tombstone may exist")
	require.Zero(t, live, "concurrent replay must commit family revocation")
}

func TestRefreshTokenService_ReplayEvidenceAndRollbackRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	store := db.NewStore(pool)
	id, password := seedRefreshAccount(t, pool)
	svc := NewRefreshTokenService(store)
	ctx := context.Background()
	raw, err := svc.Issue(ctx, id, CredentialVersion(password))
	require.NoError(t, err)
	next, _, err := svc.Rotate(ctx, raw)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE claimctl.refresh_tokens SET expires_at=NOW()-INTERVAL '1 hour' WHERE token_hash=$1", hashToken(raw))
	require.NoError(t, err)
	require.NoError(t, store.DeleteExpiredRefreshTokens(ctx))
	old, err := store.GetRefreshTokenByHash(ctx, hashToken(raw))
	require.NoError(t, err)
	require.True(t, old.Revoked)
	_, _, err = svc.Rotate(ctx, raw)
	require.ErrorIs(t, err, ErrRefreshTokenRevoked)
	child, err := store.GetRefreshTokenByHash(ctx, hashToken(next))
	require.NoError(t, err)
	require.True(t, child.Revoked)
	_, _, err = svc.Rotate(ctx, next)
	require.ErrorIs(t, err, ErrRefreshTokenRevoked)
	_, err = pool.Exec(ctx, "UPDATE claimctl.refresh_tokens SET expires_at=NOW()-INTERVAL '1 hour' WHERE user_id=$1", id)
	require.NoError(t, err)
	require.NoError(t, store.DeleteExpiredRefreshTokens(ctx))
	_, err = store.GetRefreshTokenByHash(ctx, hashToken(raw))
	require.True(t, isNotFound(err))
	raw, err = svc.Issue(ctx, id, CredentialVersion(password))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE FUNCTION claimctl.reject_refresh_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected insert failure'; END $$;
 CREATE TRIGGER reject_refresh_insert BEFORE INSERT ON claimctl.refresh_tokens FOR EACH ROW EXECUTE FUNCTION claimctl.reject_refresh_insert()`)
	require.NoError(t, err)
	newRaw, user, err := svc.Rotate(ctx, raw)
	require.Error(t, err)
	require.Empty(t, newRaw)
	require.Nil(t, user)
	old, err = store.GetRefreshTokenByHash(ctx, hashToken(raw))
	require.NoError(t, err)
	require.False(t, old.Revoked, "failed insertion must roll back consumption")
}

type refreshCommitGate struct {
	db.Store
	ready   chan struct{}
	release chan struct{}
}

func (s *refreshCommitGate) ExecTx(ctx context.Context, fn func(db.Querier) error) error {
	return s.Store.ExecTx(ctx, func(q db.Querier) error {
		if err := fn(q); err != nil {
			return err
		}
		close(s.ready)
		select {
		case <-s.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func waitForRefreshLock(t *testing.T, pool *pgxpool.Pool, query string) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting int
		err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM pg_stat_activity WHERE datname=current_database() AND application_name='claimctl_security_test' AND wait_event_type='Lock' AND (query LIKE $1 OR query LIKE '%LockRealtimeMutation%')", query).Scan(&waiting)
		return err == nil && waiting > 0
	}, 5*time.Second, 5*time.Millisecond, "expected a real account mutation serialization wait")
}

func TestRefreshTokenService_PasswordChangeSerializesBothOrdersRealDatabase(t *testing.T) {
	for _, rotationFirst := range []bool{false, true} {
		name := "password first"
		if rotationFirst {
			name = "rotation first"
		}
		t.Run(name, func(t *testing.T) {
			pool := testutils.SecurityTestDatabase(t)
			store := db.NewStore(pool)
			id, password := seedRefreshAccount(t, pool)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			raw, err := NewRefreshTokenService(store).Issue(ctx, id, CredentialVersion(password))
			require.NoError(t, err)
			gate := &refreshCommitGate{Store: store, ready: make(chan struct{}), release: make(chan struct{})}
			defer func() {
				select {
				case <-gate.release:
				default:
					close(gate.release)
				}
			}()
			type rotationResult struct {
				raw  string
				user *db.ClaimctlUser
				err  error
			}
			rotated := make(chan rotationResult, 1)
			changed := make(chan error, 1)
			rotate := func(s db.Store) {
				raw, user, err := NewRefreshTokenService(s).Rotate(ctx, raw)
				rotated <- rotationResult{raw, user, err}
			}
			change := func(s db.Store) { changed <- NewUserService(s).UpdatePassword(ctx, id, "OldPass1!", "NewPass2!") }
			if rotationFirst {
				go rotate(gate)
			} else {
				go change(gate)
			}
			select {
			case <-gate.ready:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if rotationFirst {
				go change(store)
				waitForRefreshLock(t, pool, "%FindUserByIdForUpdate%")
			} else {
				go rotate(store)
				waitForRefreshLock(t, pool, "%FindUserByIdForUpdate%")
			}
			close(gate.release)
			require.NoError(t, <-changed)
			result := <-rotated
			if rotationFirst {
				require.NoError(t, result.err)
				require.Equal(t, password, result.user.Password)
				child, err := store.GetRefreshTokenByHash(ctx, hashToken(result.raw))
				require.NoError(t, err)
				require.True(t, child.Revoked)
				current, err := store.FindUserById(ctx, id)
				require.NoError(t, err)
				require.NotEqual(t, CredentialVersion(result.user.Password), CredentialVersion(current.Password))
			} else {
				require.ErrorIs(t, result.err, ErrRefreshTokenRevoked)
				require.Empty(t, result.raw)
			}
			var live int
			require.NoError(t, pool.QueryRow(ctx, "SELECT COUNT(*) FROM claimctl.refresh_tokens WHERE user_id=$1 AND NOT revoked", id).Scan(&live))
			require.Zero(t, live)
		})
	}
}

func TestRefreshTokenService_IssueRejectsStaleLoginRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	store := db.NewStore(pool)
	id, _ := seedRefreshAccount(t, pool)
	ctx := context.Background()
	users := NewUserService(store)
	authenticated, err := users.Login(ctx, LoginRequest{Email: id.String() + "@example.invalid", Password: "OldPass1!"})
	require.NoError(t, err)
	require.NoError(t, users.UpdatePassword(ctx, id, "OldPass1!", "NewPass2!"))
	raw, err := NewRefreshTokenService(store).Issue(ctx, id, CredentialVersion(authenticated.Password))
	require.ErrorIs(t, err, ErrAccessDenied)
	require.Empty(t, raw)
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT COUNT(*) FROM claimctl.refresh_tokens WHERE user_id=$1", id).Scan(&count))
	require.Zero(t, count)
	_, err = pool.Exec(ctx, "UPDATE claimctl.users SET status='inactive' WHERE id=$1", id)
	require.NoError(t, err)
	current, err := store.FindUserById(ctx, id)
	require.NoError(t, err)
	_, err = NewRefreshTokenService(store).Issue(ctx, id, CredentialVersion(current.Password))
	require.True(t, errors.Is(err, ErrAccessDenied))
}
