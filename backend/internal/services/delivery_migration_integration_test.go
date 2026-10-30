package services

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestDeliveryMigrationAndCoordinatorRecoveryRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, err := insertDeliveryReservation(ctx, store, actor, resource)
	require.NoError(t, err)
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migration, err := migrate.New("file://"+filepath.Join(filepath.Dir(file), "../../migrations"), pool.Config().ConnString())
	require.NoError(t, err)
	defer migration.Close()
	require.NoError(t, migration.Migrate(23))
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, migration.Up())
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Zero(t, count, "up does not replay historical mutations")
	tx1, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx)
	locked, err := db.New(tx1).TryWorkerTransactionLock(ctx, "cleanup")
	require.NoError(t, err)
	require.True(t, locked)
	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)
	locked, err = db.New(tx2).TryWorkerTransactionLock(ctx, "cleanup")
	require.NoError(t, err)
	require.False(t, locked)
	require.NoError(t, tx1.Rollback(ctx))
	locked, err = db.New(tx2).TryWorkerTransactionLock(ctx, "cleanup")
	require.NoError(t, err)
	require.True(t, locked)
	require.NoError(t, tx2.Rollback(ctx))
	conn, err := pgx.Connect(ctx, pool.Config().ConnString())
	require.NoError(t, err)
	defer conn.Close(ctx)
	crashTx, err := conn.Begin(ctx)
	require.NoError(t, err)
	locked, err = db.New(crashTx).TryWorkerTransactionLock(ctx, "cleanup")
	require.NoError(t, err)
	require.True(t, locked)
	var terminated bool
	require.NoError(t, pool.QueryRow(ctx, "SELECT pg_terminate_backend($1, 5000)", conn.PgConn().PID()).Scan(&terminated))
	require.True(t, terminated)
	require.NoError(t, store.ExecTx(ctx, func(q db.Querier) error {
		locked, err := q.TryWorkerTransactionLock(ctx, "cleanup")
		require.NoError(t, err)
		require.True(t, locked)
		return nil
	}))
}
