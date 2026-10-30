package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestAPITokenHandler_RejectsInvalidExpiryInsteadOfUnlimitedToken(t *testing.T) {
	for _, expiry := range []string{"invalid", "-1h", "0h", "91d"} {
		t.Run(expiry, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			handler := NewAPITokenHandler(services.NewAPITokenService(q))
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("user_record", &db.ClaimctlUser{ID: testutils.TestUUID(1), Status: "active", Role: "user"})
				return c.Next()
			})
			app.Post("/tokens", handler.GenerateToken)
			req := httptest.NewRequest("POST", "/tokens", strings.NewReader(`{"name":"token","expires_in":"`+expiry+`"}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, 400, resp.StatusCode)
			require.Empty(t, q.Calls)
		})
	}
}
