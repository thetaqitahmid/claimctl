package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

func replicaListener(t *testing.T, app *fiber.App) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- app.Listener(listener) }()
	t.Cleanup(func() {
		http.DefaultClient.CloseIdleConnections()
		require.NoError(t, app.ShutdownWithTimeout(3*time.Second))
		require.NoError(t, <-done)
	})
	return "http://" + listener.Addr().String()
}

func replicaPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	first := testutils.SecurityTestDatabase(t)
	second, err := pgxpool.NewWithConfig(context.Background(), first.Config().Copy())
	require.NoError(t, err)
	t.Cleanup(second.Close)
	return first, second
}

func replicaStream(t *testing.T, address, token string) (<-chan types.Event, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	req, err := http.NewRequestWithContext(ctx, "GET", address+"/events", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	events := make(chan types.Event, 100)
	go func() {
		defer close(events)
		defer response.Body.Close()
		scanner := bufio.NewScanner(response.Body)
		var id int64
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "id: ") {
				id, _ = strconv.ParseInt(strings.TrimPrefix(line, "id: "), 10, 64)
			}
			if strings.HasPrefix(line, "data: ") {
				var event types.Event
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event) == nil {
					event.ID = id
					select {
					case events <- event:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	t.Cleanup(cancel)
	return events, cancel
}

func nextReplicaEvent(t *testing.T, events <-chan types.Event, kind string, resource uuid.UUID, forbidden ...uuid.UUID) types.Event {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case event, ok := <-events:
			require.True(t, ok, "stream closed before event")
			for _, id := range forbidden {
				raw, _ := json.Marshal(event.Payload)
				require.NotContains(t, string(raw), id.String(), "event leaked a foreign-space resource")
			}
			if event.Type != kind {
				continue
			}
			if resource != uuid.Nil {
				raw, _ := json.Marshal(event.Payload)
				if !strings.Contains(string(raw), resource.String()) {
					continue
				}
			}
			return event
		case <-timer.C:
			t.Fatal("timed out waiting for " + kind)
			return types.Event{}
		}
	}
}

func TestReplicaRealtimeTwoInstancesRealDatabase(t *testing.T) {
	first, second := replicaPools(t)
	ctx := context.Background()
	actor, space, foreign := uuid.New(), uuid.New(), uuid.New()
	_, err := first.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Replica','synthetic','admin')", actor, actor.String()+"@invalid.test")
	require.NoError(t, err)
	_, err = first.Exec(ctx, "INSERT INTO claimctl.spaces(id,name) VALUES($1,'visible'),($2,'hidden')", space, foreign)
	require.NoError(t, err)
	r1, r2, hidden := uuid.New(), uuid.New(), uuid.New()
	for i, r := range []uuid.UUID{r1, r2, hidden} {
		target := space
		if i == 2 {
			target = foreign
		}
		_, err = first.Exec(ctx, "INSERT INTO claimctl.resources(id,name,type,space_id) VALUES($1,$2,'http',$3)", r, r.String(), target)
		require.NoError(t, err)
	}
	raw, token, err := services.NewAPITokenService(db.NewStore(first)).GenerateToken(ctx, actor, "replicas", nil, services.TokenOptions{Scopes: []string{"read"}, SpaceIDs: []uuid.UUID{space}})
	require.NoError(t, err)
	root, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	addresses := make([]string, 2)
	for i, pool := range []*pgxpool.Pool{first, second} {
		store := db.NewStore(pool)
		realtime := services.NewSharedRealtimeService(root, store)
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		app.Use(BearerAuthentication(services.NewAPITokenService(store), store, nil))
		app.Get("/events", handlers.NewRealtimeHandler(realtime, store).HandleSSE)
		addresses[i] = replicaListener(t, app)
	}
	a, cancelA := replicaStream(t, addresses[0], raw)
	b, cancelB := replicaStream(t, addresses[1], raw)
	defer cancelA()
	defer cancelB()
	defer stop()
	nextReplicaEvent(t, a, "resync", uuid.Nil)
	nextReplicaEvent(t, b, "resync", uuid.Nil)
	_, err = first.Exec(ctx, "UPDATE claimctl.resources SET is_under_maintenance=true WHERE id=$1", r1)
	require.NoError(t, err)
	nextReplicaEvent(t, a, "maintenance_change", r1)
	nextReplicaEvent(t, b, "maintenance_change", r1)

	tx, err := first.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "UPDATE claimctl.resources SET is_under_maintenance=false WHERE id=$1", r1)
	require.NoError(t, err)
	var lower int64
	require.NoError(t, tx.QueryRow(ctx, "SELECT max(id) FROM claimctl.realtime_events").Scan(&lower))
	committed := make(chan error, 1)
	go func() {
		_, err := second.Exec(ctx, "UPDATE claimctl.resources SET is_under_maintenance=true WHERE id=$1", r2)
		committed <- err
	}()
	require.Eventually(t, func() bool {
		var count int
		err := first.QueryRow(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND NOT granted AND objid=725025").Scan(&count)
		return err == nil && count > 0
	}, time.Second, 5*time.Millisecond)
	select {
	case err := <-committed:
		t.Fatalf("later writer bypassed commit fence: %v", err)
	default:
	}
	require.NoError(t, tx.Commit(ctx))
	require.NoError(t, <-committed)
	e1 := nextReplicaEvent(t, b, "maintenance_change", r1)
	e2 := nextReplicaEvent(t, b, "maintenance_change", r2)
	require.Equal(t, lower, e1.ID)
	require.Greater(t, e2.ID, e1.ID, "cursor cannot skip delayed lower-ID commit")
	nextReplicaEvent(t, a, "maintenance_change", r1)
	nextReplicaEvent(t, a, "maintenance_change", r2)

	rollback, err := first.Begin(ctx)
	require.NoError(t, err)
	_, err = rollback.Exec(ctx, "UPDATE claimctl.resources SET name='rollback' WHERE id=$1", r1)
	require.NoError(t, err)
	var aborted int64
	require.NoError(t, rollback.QueryRow(ctx, "SELECT max(id) FROM claimctl.realtime_events").Scan(&aborted))
	require.NoError(t, rollback.Rollback(ctx))
	var exists bool
	require.NoError(t, first.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM claimctl.realtime_events WHERE id=$1)", aborted).Scan(&exists))
	require.False(t, exists)
	_, err = first.Exec(ctx, "UPDATE claimctl.resources SET name='private' WHERE id=$1", hidden)
	require.NoError(t, err)
	_, err = first.Exec(ctx, "DELETE FROM claimctl.resources WHERE id=$1", r1)
	require.NoError(t, err)
	deletion := nextReplicaEvent(t, b, "maintenance_change", r1, hidden)
	require.NotContains(t, fmt.Sprint(deletion.Payload), "private")
	require.NoError(t, first.QueryRow(ctx, "SELECT space_id=$2 FROM claimctl.realtime_events WHERE id=$1", deletion.ID, space).Scan(&exists))
	require.True(t, exists, "pre-delete scope retained without resource lookup")
	_, err = first.Exec(ctx, "UPDATE claimctl.resources SET space_id=$2 WHERE id=$1", r2, foreign)
	require.NoError(t, err)
	resync := nextReplicaEvent(t, b, "resync", uuid.Nil)
	require.Nil(t, resync.Payload)
	_, err = first.Exec(ctx, "DELETE FROM claimctl.api_tokens WHERE id=$1", token.ID)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-b:
			return !ok
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "revocation disconnects without data")
	stop()
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-a:
			return !ok
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "shutdown disconnects streams")
}

func TestReplicaStreamExpiresRealDatabase(t *testing.T) {
	for _, shortened := range []bool{false, true} {
		t.Run(fmt.Sprintf("shortened=%t", shortened), func(t *testing.T) {
			pool := testutils.SecurityTestDatabase(t)
			store := db.NewStore(pool)
			actor := uuid.New()
			_, err := pool.Exec(context.Background(), "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Expiry','synthetic','admin')", actor, actor.String()+"@invalid.test")
			require.NoError(t, err)
			expiry := time.Now().Add(time.Second)
			if shortened {
				expiry = time.Now().Add(time.Hour)
			}
			raw, token, err := services.NewAPITokenService(store).GenerateToken(context.Background(), actor, "expires", &expiry)
			require.NoError(t, err)
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			realtime := services.NewSharedRealtimeService(ctx, store)
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(BearerAuthentication(services.NewAPITokenService(store), store, nil))
			app.Get("/events", handlers.NewRealtimeHandler(realtime, store).HandleSSE)
			address := replicaListener(t, app)
			events, cancel := replicaStream(t, address, raw)
			defer cancel()
			nextReplicaEvent(t, events, "resync", uuid.Nil)
			if shortened {
				_, err = pool.Exec(ctx, "UPDATE claimctl.api_tokens SET expires_at=clock_timestamp()+interval '500 milliseconds' WHERE id=$1", token.ID)
				require.NoError(t, err)
			}
			require.Eventually(t, func() bool {
				select {
				case _, ok := <-events:
					return !ok
				default:
					return false
				}
			}, 2*time.Second, 5*time.Millisecond, "stream closes at current credential expiry without another mutation")
		})
	}
}

func TestReplicaCSRFAndAtomicLimiterTwoInstancesRealDatabase(t *testing.T) {
	first, second := replicaPools(t)
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.1/32")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	addresses := make([]string, 2)
	for i, pool := range []*pgxpool.Pool{first, second} {
		q := db.New(pool)
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		app.Use(TrustedProxyMiddleware())
		app.Get("/limited", AtomicRateLimiter(q, "concurrent", 17, time.Minute), func(c *fiber.Ctx) error { return c.SendStatus(200) })
		app.Use(CSRFMiddleware(NewPostgresStorage(q)))
		app.Get("/csrf", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"csrfToken": c.Locals("csrf")}) })
		app.Post("/unsafe", func(c *fiber.Ctx) error { return c.SendStatus(200) })
		addresses[i] = replicaListener(t, app)
	}
	response, err := http.Get(addresses[0] + "/csrf")
	require.NoError(t, err)
	var data struct {
		Token string `json:"csrfToken"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&data))
	response.Body.Close()
	require.Len(t, data.Token, 64)
	cookie := response.Cookies()[0]
	call := func(header, origin string) int {
		req, err := http.NewRequest("POST", addresses[1]+"/unsafe", nil)
		require.NoError(t, err)
		req.AddCookie(cookie)
		req.Header.Set("X-CSRF-Token", header)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", "claims.example:8443")
		req.Header.Set("Origin", origin)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		return resp.StatusCode
	}
	require.Equal(t, 200, call(data.Token, "https://claims.example:8443"), "TLS ingress to HTTP frontend preserves external origin")
	require.Equal(t, 403, call(data.Token, "https://evil.example"))
	require.Equal(t, 403, call("wrong", "https://claims.example:8443"))
	var wg sync.WaitGroup
	results := make(chan int, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := http.Get(addresses[i%2] + "/limited")
			if err != nil {
				results <- 0
				return
			}
			results <- resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	close(results)
	success, limited := 0, 0
	for status := range results {
		switch status {
		case 200:
			success++
		case 429:
			limited++
		default:
			t.Errorf("unexpected limiter status %d", status)
		}
	}
	require.Equal(t, 17, success)
	require.Equal(t, 83, limited)
	_, err = first.Exec(context.Background(), "UPDATE claimctl.rate_limits SET expires_at=clock_timestamp()-interval '1 second'")
	require.NoError(t, err)
	resp, err := http.Get(addresses[1] + "/limited")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	_, err = first.Exec(context.Background(), "UPDATE claimctl.shared_storage SET expires_at=clock_timestamp()-interval '1 second'")
	require.NoError(t, err)
	require.Equal(t, 403, call(data.Token, "https://claims.example:8443"))
	_, err = first.Exec(context.Background(), "DROP TABLE claimctl.shared_storage")
	require.NoError(t, err)
	require.Equal(t, 503, call(data.Token, "https://claims.example:8443"))
	_, err = first.Exec(context.Background(), "DROP TABLE claimctl.rate_limits")
	require.NoError(t, err)
	resp, err = http.Get(addresses[1] + "/limited")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 503, resp.StatusCode)
}
