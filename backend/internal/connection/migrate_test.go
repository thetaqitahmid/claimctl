package connection

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationDSNPinsPublicSchema(t *testing.T) {
	dsn := "postgres://user:password@127.0.0.1:55432/claimctl?sslmode=disable&x-migrations-table=wrong"
	migrationURL, err := migrationDSN(dsn)
	require.NoError(t, err)

	u, err := url.Parse(migrationURL)
	require.NoError(t, err)
	require.Equal(t, "postgres", u.Scheme)
	require.Equal(t, "127.0.0.1:55432", u.Host)
	require.Equal(t, "/claimctl", u.Path)

	query := u.Query()
	require.Equal(t, `"public"."schema_migrations"`, query.Get("x-migrations-table"))
	require.Equal(t, "true", query.Get("x-migrations-table-quoted"))
	require.Equal(t, "disable", query.Get("sslmode"))
}

func TestMigrationDSNRejectsKeywordValueDSN(t *testing.T) {
	_, err := migrationDSN("host=127.0.0.1 dbname=claimctl")
	require.Error(t, err)
}
