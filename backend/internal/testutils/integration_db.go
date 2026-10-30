package testutils

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func SecurityTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return securityTestDatabase(t, true)
}

func EmptySecurityTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return securityTestDatabase(t, false)
}

func securityTestDatabase(t *testing.T, applyMigrations bool) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("SECURITY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SECURITY_TEST_DATABASE_URL required for isolated database regressions")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.ConnConfig.Host, "only disposable localhost databases are allowed")
	require.Contains(t, []uint16{55432, 55433}, cfg.ConnConfig.Port, "production database ports are forbidden")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(admin.Close)
	name := "claimctl_security_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" TEMPLATE template0")
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := admin.Exec(cleanupCtx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		require.NoError(t, err)
	})
	target, err := url.Parse(dsn)
	require.NoError(t, err)
	target.Path = "/" + name
	if applyMigrations {
		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		migration, err := migrate.New("file://"+filepath.Join(filepath.Dir(file), "../../migrations"), target.String())
		require.NoError(t, err)
		require.NoError(t, migration.Up())
		sourceErr, dbErr := migration.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, dbErr)
	}
	testCfg, err := pgxpool.ParseConfig(target.String())
	require.NoError(t, err)
	testCfg.ConnConfig.RuntimeParams["application_name"] = "claimctl_security_test"
	pool, err := pgxpool.NewWithConfig(ctx, testCfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}
