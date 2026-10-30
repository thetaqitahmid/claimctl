package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/thetaqitahmid/claimctl/internal/connection"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

func TestFreshDeploymentFromNativeDatabaseBackupRealDatabase(t *testing.T) {
	source := testutils.SecurityTestDatabase(t)
	target := testutils.EmptySecurityTestDatabase(t)
	for _, binary := range []string{"pg_dump", "pg_restore", "psql"} {
		_, err := exec.LookPath(binary)
		require.NoError(t, err, "native recovery tests require PostgreSQL client tools")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	t.Chdir("../..")
	t.Setenv("APP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ENABLE_CSRF", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")
	encryption, err := utils.LoadOrGenerateKey("APP_ENCRYPTION_KEY", filepath.Join(t.TempDir(), "unused.key"))
	require.NoError(t, err)
	store := db.NewStore(source)
	settings := services.NewSettingsService(store, encryption)
	private, public, err := settings.GetOrGenerateJWTKeys(ctx)
	require.NoError(t, err)
	_, err = settings.Set(ctx, "recovery_secret", "synthetic setting", "auth", "recovery test", true)
	require.NoError(t, err)
	secret, err := services.NewSecretService(store, encryption).CreateSecret(ctx, "recovery_secret", "synthetic secret", "recovery test")
	require.NoError(t, err)
	require.NoError(t, services.NewSpaceService(store).EnsureDefaultSpaceExists(ctx))

	actor, space, resource, reservation, hook := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	password := "SyntheticRecovery1!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	email := actor.String() + "@invalid.test"
	_, err = source.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Recovered admin',$3,'admin')", actor, email, string(hash))
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.spaces(id,name) VALUES($1,'Recovery space')", space)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id,properties) VALUES($1,'Recovered resource','test',$2,'{\"endpoint\":\"https://invalid.test\"}')", resource, space)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.webhooks(id,name,url) VALUES($1,'Recovery hook','https://invalid.test')", hook)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.resource_webhooks(resource_id,webhook_id,events) VALUES($1,$2,ARRAY['reservation.created'])", resource, hook)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.reservations(id,resource_id,user_id,status,start_time,duration,lease_duration,lease_expires_at,hard_end_time) VALUES($1,$2,$3,'active',100,INTERVAL '1 month 2 days 03:04:05.123456',INTERVAL '10 minutes',NOW()+INTERVAL '1 day',NOW()+INTERVAL '2 days')", reservation, resource, actor)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.audit_logs(actor_id,action,entity_type,entity_id,created_at) VALUES($1,'create','resource',$2,100)", actor, resource.String())
	require.NoError(t, err)
	rawToken, token, err := services.NewAPITokenService(store).GenerateToken(ctx, actor, "Recovery token", nil, services.TokenOptions{Scopes: []string{"read", "reserve", "users:read"}, SpaceIDs: []uuid.UUID{space}})
	require.NoError(t, err)
	_, err = services.NewRefreshTokenService(store).Issue(ctx, actor, services.CredentialVersion(string(hash)))
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.claim_idempotency(user_id,key,fingerprint,reservation_id,result) VALUES($1,'old-claim','synthetic',$2,'{}')", actor, reservation)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.bundle_idempotency(user_id,key,fingerprint,result) VALUES($1,'old-bundle','synthetic','{}')", actor)
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.worker_leases(name,fence,lease_until) VALUES('health:recovery',$1,NOW()+INTERVAL '1 hour')", uuid.New())
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.shared_storage(key,value,expires_at) VALUES('old-csrf','synthetic',NOW()+INTERVAL '1 hour')")
	require.NoError(t, err)
	_, err = source.Exec(ctx, "INSERT INTO claimctl.rate_limits(key,count,expires_at) VALUES('old-counter',1,NOW()+INTERVAL '1 hour')")
	require.NoError(t, err)

	before := recoveryDatabaseSnapshot(t, source)
	archive := filepath.Join(t.TempDir(), "claimctl.dump")
	runRecoveryTool(t, ctx, "pg_dump", "--dbname="+source.Config().ConnString(), "--format=custom", "--file="+archive)
	runRecoveryTool(t, ctx, "pg_restore", "--dbname="+target.Config().ConnString(), "--no-owner", "--no-privileges", "--exit-on-error", "--single-transaction", archive)
	require.Equal(t, before, recoveryDatabaseSnapshot(t, target), "full native restore preserves business data, runtime data, and migrations without firing import triggers")

	runtimeTables := []string{"refresh_tokens", "claim_idempotency", "bundle_idempotency", "webhook_outbox", "worker_leases", "realtime_events", "shared_storage", "rate_limits"}
	for _, table := range runtimeTables {
		require.NotEqual(t, "[]", before["claimctl."+table], "runtime cleanup must be tested with populated %s", table)
	}
	runRecoveryTool(t, ctx, "psql", target.Config().ConnString(), "-X", "--set=ON_ERROR_STOP=1", "--file=../deployment/postgres-recovery.sql")
	after := recoveryDatabaseSnapshot(t, target)
	for _, table := range runtimeTables {
		require.Equal(t, "[]", after["claimctl."+table])
		delete(before, "claimctl."+table)
		delete(after, "claimctl."+table)
	}
	require.Equal(t, before, after, "offline cleanup must not change business data")

	t.Setenv("DATABASE_URL", target.Config().ConnString())
	recovered, err := connection.CreateDBSession()
	require.NoError(t, err, "production startup must accept restored migrations and existing admin without bootstrap credentials")
	t.Cleanup(recovered.Close)
	recoveredStore := db.NewStore(recovered)
	recoveredSettings := services.NewSettingsService(recoveredStore, encryption)
	otherPrivate, otherPublic, err := recoveredSettings.GetOrGenerateJWTKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, private.N, otherPrivate.N)
	require.Equal(t, public.N, otherPublic.N)
	require.Equal(t, "synthetic setting", recoveredSettings.GetString(ctx, "recovery_secret"))
	decrypted, err := services.NewSecretService(recoveredStore, encryption).GetSecret(ctx, secret.ID)
	require.NoError(t, err)
	require.Equal(t, "synthetic secret", decrypted.Value)
	_, _, err = services.NewSettingsService(recoveredStore, base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))).GetOrGenerateJWTKeys(ctx)
	require.Error(t, err, "a different encryption key must fail closed")
	auth, err := services.NewAPITokenService(recoveredStore).ValidateAuthorization(ctx, rawToken)
	require.NoError(t, err)
	require.Equal(t, token.Scopes, auth.Scopes)
	require.Equal(t, token.SpaceIds, auth.SpaceIDs)

	serverCtx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	instance := NewServer(serverCtx, recovered, otherPrivate, otherPublic, encryption)
	address := replicaListener(t, instance.app)
	for _, route := range instance.app.GetRoutes() {
		require.NotEqual(t, "/api/admin/backup", route.Path)
		require.NotEqual(t, "/api/admin/restore", route.Path)
	}
	for _, path := range []string{"/v1/health", "/v1/ready"} {
		response, err := http.Get(address + path)
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
	}
	response, err := http.Post(address+"/api/login", "application/json", strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	var login struct {
		User struct {
			ID uuid.UUID `json:"id"`
		} `json:"user"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&login))
	require.Equal(t, actor, login.User.ID)
	request, err := http.NewRequest("GET", address+"/api/resources/"+resource.String(), nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+rawToken)
	resourceResponse, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer resourceResponse.Body.Close()
	require.Equal(t, http.StatusOK, resourceResponse.StatusCode)
	var recoveredResource struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resourceResponse.Body).Decode(&recoveredResource))
	require.Equal(t, resource, recoveredResource.ID)
	var jobs int
	require.NoError(t, recovered.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&jobs))
	require.Zero(t, jobs, "fresh startup must not recreate historical delivery jobs")
}

func runRecoveryTool(t *testing.T, ctx context.Context, binary string, args ...string) {
	t.Helper()
	command := exec.CommandContext(ctx, binary, args...)
	command.Env = append(os.Environ(), "PGCONNECT_TIMEOUT=10")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s failed: %s", binary, output)
}

func recoveryDatabaseSnapshot(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	ctx := context.Background()
	rows, err := pool.Query(ctx, "SELECT schemaname,tablename FROM pg_tables WHERE schemaname IN ('claimctl','public') ORDER BY schemaname,tablename")
	require.NoError(t, err)
	var tables []pgx.Identifier
	for rows.Next() {
		var schema, table string
		require.NoError(t, rows.Scan(&schema, &table))
		tables = append(tables, pgx.Identifier{schema, table})
	}
	require.NoError(t, rows.Err())
	rows.Close()
	snapshot := make(map[string]string, len(tables))
	for _, table := range tables {
		var data string
		require.NoError(t, pool.QueryRow(ctx, "SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]'::jsonb)::text FROM "+table.Sanitize()+" AS row").Scan(&data))
		snapshot[strings.Join(table, ".")] = data
	}
	return snapshot
}
