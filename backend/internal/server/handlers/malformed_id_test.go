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

func TestMalformedID_StopsBeforeServiceDatabaseWork(t *testing.T) {
	q := &testutils.MockQuerier{}
	resources := NewResourceHandler(services.NewResourceService(q, nil), nil)
	health := NewHealthCheckHandler(services.NewHealthCheckService(q))
	reservations := NewReservationHandler(services.NewReservationService(q, nil, nil, nil, nil))
	for _, tc := range []struct {
		name, method string
		handler      fiber.Handler
	}{
		{"resource", "GET", resources.GetResourceByID},
		{"resource status", "GET", resources.GetResourceWithStatus},
		{"maintenance history", "GET", resources.GetMaintenanceHistory},
		{"health status", "GET", health.GetHealthStatus},
		{"health check", "POST", health.TriggerHealthCheck},
		{"reservation", "GET", reservations.GetReservation},
		{"reservation complete", "PATCH", reservations.CompleteReservation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("user_record", &db.ClaimctlUser{ID: testutils.TestUUID(1), Role: "admin", Status: "active"})
				return c.Next()
			})
			app.Add(tc.method, "/target/:id", tc.handler)
			req := httptest.NewRequest(tc.method, "/target/not-a-uuid", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, 400, resp.StatusCode)
			require.Empty(t, q.Calls)
		})
	}
}
