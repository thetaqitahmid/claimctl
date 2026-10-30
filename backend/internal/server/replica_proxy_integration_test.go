package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestReplicaNginxTLSIngressAndSSERealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker required for isolated Nginx proxy regression")
	}
	if err := exec.Command("docker", "image", "inspect", "nginx:alpine").Run(); err != nil {
		t.Skip("nginx:alpine image required")
	}
	t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.1/32")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	store := db.NewStore(pool)
	actor := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO claimctl.users(id,email,name,password,role) VALUES($1,$2,'Proxy','synthetic','admin')", actor, actor.String()+"@invalid.test")
	require.NoError(t, err)
	token, _, err := services.NewAPITokenService(store).GenerateToken(ctx, actor, "proxy", nil)
	require.NoError(t, err)
	realtime := services.NewSharedRealtimeService(ctx, store)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(TrustedProxyMiddleware(), BearerAuthentication(services.NewAPITokenService(store), store, nil), CSRFMiddleware(NewPostgresStorage(store)))
	app.Get("/api/auth/csrf", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"csrfToken": c.Locals("csrf")}) })
	app.Post("/api/unsafe", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/api/events", handlers.NewRealtimeHandler(realtime, store).HandleSSE)
	backend := replicaListener(t, app)
	backendURL, err := url.Parse(backend)
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	template, err := os.ReadFile("../../../frontend/nginx.conf.template")
	require.NoError(t, err)
	rendered := strings.NewReplacer("${BACKEND_HOST}", "127.0.0.1", "${BACKEND_PORT}", backendURL.Port(), "${TRUSTED_INGRESS_CIDR}", "127.0.0.1/32", "listen 8080;", "listen "+address+";").Replace(string(template))
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "default.conf"), []byte(rendered), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nginx.conf"), []byte("pid /tmp/nginx.pid;\nevents {}\nhttp { access_log off; include /tmp/default.conf; }\n"), 0600))
	name := "claimctl-nginx-test-" + uuid.NewString()
	command := exec.Command("docker", "run", "--rm", "--network", "host", "--name", name,
		"-v", filepath.Join(dir, "default.conf")+":/tmp/default.conf:ro",
		"-v", filepath.Join(dir, "nginx.conf")+":/tmp/nginx.conf:ro",
		"nginx:alpine", "nginx", "-c", "/tmp/nginx.conf", "-g", "daemon off;")
	require.NoError(t, command.Start())
	t.Cleanup(func() { _ = exec.Command("docker", "stop", "-t", "1", name).Run(); _ = command.Wait() })
	proxy := "http://" + address
	require.Eventually(t, func() bool {
		client := http.Client{Timeout: 100 * time.Millisecond}
		response, err := client.Get(proxy + "/api/auth/csrf")
		if err != nil {
			return false
		}
		response.Body.Close()
		return response.StatusCode == 200
	}, 5*time.Second, 20*time.Millisecond, "isolated Nginx must start")
	response, err := http.Get(proxy + "/api/auth/csrf")
	require.NoError(t, err)
	var data struct {
		Token string `json:"csrfToken"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&data))
	response.Body.Close()
	cookie := response.Cookies()[0]
	request := func(client *http.Client) int {
		req, err := http.NewRequest("POST", proxy+"/api/unsafe", nil)
		require.NoError(t, err)
		req.Host = "claims.example:8443"
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("Origin", "https://claims.example:8443")
		req.Header.Set("X-CSRF-Token", data.Token)
		req.AddCookie(cookie)
		response, err := client.Do(req)
		require.NoError(t, err)
		response.Body.Close()
		return response.StatusCode
	}
	require.Equal(t, 200, request(http.DefaultClient), "trusted TLS ingress survives HTTP Nginx hop")
	transport := &http.Transport{DialContext: (&net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.2")}}).DialContext}
	defer transport.CloseIdleConnections()
	require.Equal(t, 403, request(&http.Client{Transport: transport, Timeout: time.Second}), "untrusted ingress cannot spoof external TLS origin")
	// The first tiny SSE frame must arrive before stream completion, not buffer.
	events, closeStream := replicaStream(t, proxy+"/api", token)
	nextReplicaEvent(t, events, "resync", uuid.Nil)
	closeStream()
	stop()
}
