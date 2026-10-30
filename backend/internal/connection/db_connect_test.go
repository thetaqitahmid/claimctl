package connection

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDatabaseURLPrecedenceAndEscaping(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "::1")
	t.Setenv("DB_PORT", "55432")
	t.Setenv("DB_USER", "synthetic@user")
	t.Setenv("DB_PASSWORD", "p@ss:/?#%+ &$word")
	t.Setenv("DB_NAME", "synthetic db")
	t.Setenv("SSL_MODE", "require")
	config, err := pgxpool.ParseConfig(DatabaseURL())
	require.NoError(t, err)
	require.Equal(t, "synthetic@user", config.ConnConfig.User)
	require.Equal(t, "p@ss:/?#%+ &$word", config.ConnConfig.Password)
	require.Equal(t, "synthetic db", config.ConnConfig.Database)
	require.Equal(t, "::1", config.ConnConfig.Host)
	require.Equal(t, uint16(55432), config.ConnConfig.Port)
	t.Setenv("DATABASE_URL", "postgres://explicit@127.0.0.1:55432/explicit?sslmode=disable")
	require.Equal(t, "postgres://explicit@127.0.0.1:55432/explicit?sslmode=disable", DatabaseURL())
}
