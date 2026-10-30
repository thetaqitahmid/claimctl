package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"golang.org/x/crypto/bcrypt"
)

func TestReplicaFullServerWiringRealDatabase(t *testing.T) {
	first, second := replicaPools(t)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ENABLE_CSRF", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")
	encryption := base64.StdEncoding.EncodeToString(make([]byte, 32))
	private, public, err := services.NewSettingsService(db.NewStore(first), encryption).GetOrGenerateJWTKeys(ctx)
	require.NoError(t, err)
	otherPrivate, otherPublic, err := services.NewSettingsService(db.NewStore(second), encryption).GetOrGenerateJWTKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, private.N.Cmp(otherPrivate.N))
	require.Equal(t, 0, public.N.Cmp(otherPublic.N))
	a := NewServer(ctx, first, private, public, encryption)
	b := NewServer(ctx, second, otherPrivate, otherPublic, encryption)
	firstAddress := replicaListener(t, a.app)
	secondAddress := replicaListener(t, b.app)
	actor, space, resource := uuid.New(), uuid.New(), uuid.New()
	password := "SyntheticReplica1!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	email := actor.String() + "@invalid.test"
	_, err = first.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Full replica',$3,'admin')", actor, email, string(hash))
	require.NoError(t, err)
	_, err = first.Exec(ctx, "INSERT INTO claimctl.spaces(id,name) VALUES($1,'full replica')", space)
	require.NoError(t, err)
	_, err = first.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,'full replica','http',$2)", resource, space)
	require.NoError(t, err)
	token, _, err := services.NewAPITokenService(db.NewStore(first)).GenerateToken(ctx, actor, "full replica", nil, services.TokenOptions{Scopes: []string{"read"}, SpaceIDs: []uuid.UUID{space}})
	require.NoError(t, err)
	response, err := http.Get(firstAddress + "/api/auth/csrf")
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	var challenge struct {
		Token string `json:"csrfToken"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&challenge))
	response.Body.Close()
	cookie := response.Cookies()[0]
	for i := 0; i < 6; i++ {
		address := firstAddress
		if i%2 == 0 {
			address = secondAddress
		}
		req, err := http.NewRequest("POST", address+"/api/login", strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", challenge.Token)
		req.AddCookie(cookie)
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		response.Body.Close()
		expected := 200
		if i == 5 {
			expected = 429
		}
		require.Equal(t, expected, response.StatusCode, "real route wiring must share CSRF and login counters")
	}
	events, cancel := replicaStream(t, secondAddress+"/api", token)
	defer cancel()
	nextReplicaEvent(t, events, "resync", uuid.Nil)
	_, err = first.Exec(ctx, "UPDATE claimctl.resources SET is_under_maintenance=true WHERE id=$1", resource)
	require.NoError(t, err)
	var mutationID int64
	require.NoError(t, first.QueryRow(ctx, "SELECT max(id) FROM claimctl.realtime_events").Scan(&mutationID))
	for {
		event := nextReplicaEvent(t, events, "maintenance_change", resource)
		if event.ID >= mutationID {
			require.Equal(t, mutationID, event.ID)
			break
		}
	}
	stop()
}
