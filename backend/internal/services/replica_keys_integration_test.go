package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestReplicaJWTConcurrentInitializationRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")
	encryption := base64.StdEncoding.EncodeToString(make([]byte, 32))
	var wg sync.WaitGroup
	type result struct {
		private *rsa.PrivateKey
		public  *rsa.PublicKey
		err     error
	}
	results := make(chan result, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			private, public, err := NewSettingsService(db.NewStore(pool), encryption).GetOrGenerateJWTKeys(context.Background())
			results <- result{private, public, err}
		}()
	}
	wg.Wait()
	close(results)
	var stable *rsa.PrivateKey
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, 0, result.private.N.Cmp(result.public.N))
		if stable == nil {
			stable = result.private
		}
		require.Equal(t, 0, stable.N.Cmp(result.private.N))
	}
	settings := NewSettingsService(db.NewStore(pool), encryption)
	private, public, err := settings.GetOrGenerateJWTKeys(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, stable.N.Cmp(private.N))
	require.Equal(t, 0, stable.N.Cmp(public.N))
	row, err := db.New(pool).GetSetting(context.Background(), "jwt_private_key")
	require.NoError(t, err)
	require.True(t, row.IsSecret)
	require.NotContains(t, row.Value, "PRIVATE KEY")
	_, _, err = NewSettingsService(db.NewStore(pool), base64.StdEncoding.EncodeToString([]byte("11111111111111111111111111111111"))).GetOrGenerateJWTKeys(context.Background())
	require.Error(t, err, "replicas cannot silently replace keys encrypted by another key")
	after, err := db.New(pool).GetSetting(context.Background(), "jwt_private_key")
	require.NoError(t, err)
	require.Equal(t, row.Value, after.Value)
}

func TestReplicaJWTInvalidPairsFailClosedRealDatabase(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	private := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	for _, kind := range []string{"partial", "corrupt", "mismatched", "database failure"} {
		t.Run(kind, func(t *testing.T) {
			pool := testutils.SecurityTestDatabase(t)
			t.Setenv("JWT_PRIVATE_KEY", "")
			t.Setenv("JWT_PUBLIC_KEY", "")
			settings := NewSettingsService(db.NewStore(pool), base64.StdEncoding.EncodeToString(make([]byte, 32)))
			_, err := settings.Set(context.Background(), "jwt_private_key", private, "auth", "", true)
			require.NoError(t, err)
			if kind == "corrupt" {
				_, err = settings.Set(context.Background(), "jwt_public_key", "not a public key", "auth", "", false)
				require.NoError(t, err)
			}
			if kind == "mismatched" {
				other, err := rsa.GenerateKey(rand.Reader, 2048)
				require.NoError(t, err)
				der, err := x509.MarshalPKIXPublicKey(&other.PublicKey)
				require.NoError(t, err)
				_, err = settings.Set(context.Background(), "jwt_public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), "auth", "", false)
				require.NoError(t, err)
			}
			if kind == "database failure" {
				_, err = pool.Exec(context.Background(), "DROP TABLE app_settings")
				require.NoError(t, err)
			}
			_, _, err = settings.GetOrGenerateJWTKeys(context.Background())
			require.Error(t, err)
			if kind != "database failure" {
				row, err := db.New(pool).GetSetting(context.Background(), "jwt_private_key")
				require.NoError(t, err)
				require.NotEmpty(t, row.Value)
				require.Equal(t, private, settings.GetString(context.Background(), "jwt_private_key"))
			}
		})
	}
}
