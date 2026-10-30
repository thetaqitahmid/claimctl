package services

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestLifecycleMigrationsPopulatedRoundtripRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migration, err := migrate.New("file://"+filepath.Join(filepath.Dir(file), "../../migrations"), pool.Config().ConnString())
	require.NoError(t, err)
	t.Cleanup(func() {
		sourceErr, dbErr := migration.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, dbErr)
	})
	require.NoError(t, migration.Migrate(20))
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	reservationID, tokenID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO claimctl.reservations(id,resource_id,user_id,status,queue_position)
		VALUES($1,$2,$3,'pending',1)`, reservationID, resource, actor)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO claimctl.api_tokens(id,user_id,name,token_hash)
		VALUES($1,$2,'migration fixture','synthetic')`, tokenID, actor)
	require.NoError(t, err)

	assertBusinessData := func(t *testing.T) {
		t.Helper()
		var count int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations WHERE id=$1 AND status='pending'", reservationID).Scan(&count))
		require.Equal(t, 1, count)
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.api_tokens WHERE id=$1", tokenID).Scan(&count))
		require.Equal(t, 1, count)
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservation_history WHERE reservation_id=$1", reservationID).Scan(&count))
		require.Positive(t, count)
	}
	for version := uint(21); version <= 25; version++ {
		t.Run(fmt.Sprintf("up_%d", version), func(t *testing.T) {
			require.NoError(t, migration.Migrate(version))
			assertBusinessData(t)
		})
	}
	var scopes []string
	var bounded bool
	require.NoError(t, pool.QueryRow(ctx, "SELECT scopes, expires_at IS NOT NULL FROM claimctl.api_tokens WHERE id=$1", tokenID).Scan(&scopes, &bounded))
	require.Equal(t, []string{"read", "reserve"}, scopes)
	require.True(t, bounded)
	for version := uint(24); version >= 20; version-- {
		t.Run(fmt.Sprintf("down_to_%d", version), func(t *testing.T) {
			require.NoError(t, migration.Migrate(version))
			assertBusinessData(t)
		})
	}
	_, err = pool.Exec(ctx, "UPDATE claimctl.reservations SET queue_position=2 WHERE id=$1", reservationID)
	require.NoError(t, err)
	var action string
	require.NoError(t, pool.QueryRow(ctx, "SELECT action FROM claimctl.reservation_history WHERE reservation_id=$1 AND action='queue_position_changed'", reservationID).Scan(&action))
	require.Equal(t, "queue_position_changed", action)
	require.NoError(t, migration.Up())
	assertBusinessData(t)
	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&outboxCount))
	require.Zero(t, outboxCount, "upgrades must not replay historical reservations")
}
