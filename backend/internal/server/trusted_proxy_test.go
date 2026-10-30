package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestTrustedProxyOriginAndSpoofing(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		t.Run(map[bool]string{false: "untrusted", true: "trusted chain"}[trusted], func(t *testing.T) {
			cidrs := ""
			if trusted {
				cidrs = "127.0.0.1/32,10.0.0.0/24"
			}
			t.Setenv("TRUSTED_PROXY_CIDRS", cidrs)
			app := fiber.New(fiber.Config{DisableStartupMessage: true, ProxyHeader: fiber.HeaderXForwardedFor})
			app.Use(TrustedProxyMiddleware())
			app.Get("/", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"origin": ExternalOrigin(c), "ip": ClientIP(c), "base": c.BaseURL(), "fiberIP": c.IP()})
			})
			address := replicaListener(t, app)
			request := func(headers map[string]string) *http.Response {
				req, err := http.NewRequest("GET", address, nil)
				require.NoError(t, err)
				for name, value := range headers {
					req.Header.Set(name, value)
				}
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				t.Cleanup(func() { resp.Body.Close() })
				return resp
			}
			response := request(nil)
			require.Equal(t, 200, response.StatusCode)
			headers := map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "claim.example:8443", "X-Forwarded-For": "203.0.113.99,198.51.100.7,10.0.0.8"}
			response = request(headers)
			if !trusted {
				require.Equal(t, 400, response.StatusCode)
				for _, header := range []string{"Forwarded", "X-Forwarded-Protocol", "X-Forwarded-Ssl", "X-Url-Scheme", "X-Real-IP", "X-Forwarded-Unknown"} {
					require.Equal(t, 400, request(map[string]string{header: "spoofed"}).StatusCode)
				}
				return
			}
			require.Equal(t, 200, response.StatusCode)
			var data map[string]string
			require.NoError(t, json.NewDecoder(response.Body).Decode(&data))
			require.Equal(t, "https://claim.example:8443", data["origin"])
			require.Equal(t, data["origin"], data["base"])
			require.Equal(t, "198.51.100.7", data["ip"])
			require.Equal(t, data["ip"], data["fiberIP"])
			for _, headers := range []map[string]string{
				{"X-Forwarded-Proto": "https,http"}, {"X-Forwarded-Host": "evil.example/path"}, {"X-Forwarded-Host": "victim.example,evil.example"}, {"X-Forwarded-For": "garbage"}, {"Forwarded": "proto=https"},
			} {
				require.Equal(t, 400, request(headers).StatusCode)
			}
		})
	}
	t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-prefix")
	require.Panics(t, func() { TrustedProxyMiddleware() })
}

func TestCSRFMissingStorageFailsClosed(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware(nil))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	address := replicaListener(t, app)
	response, err := http.Get(address)
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, 503, response.StatusCode)
}
