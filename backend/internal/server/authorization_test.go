package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestAuthorizationMiddleware_CrossSpaceResourceEndpoints(t *testing.T) {
	resourceID, spaceID, userID := testutils.TestUUID(1), testutils.TestUUID(2), testutils.TestUUID(3)
	for _, endpoint := range []struct{ method, suffix string }{
		{"GET", ""}, {"GET", "/with-status"}, {"GET", "/reservations"},
		{"GET", "/queue"}, {"GET", "/queue/next"}, {"GET", "/queue/position"},
		{"POST", "/queue/process"}, {"GET", "/health/config"}, {"PUT", "/health/config"},
		{"GET", "/health/status"}, {"GET", "/health/history"}, {"POST", "/health/check"},
		{"GET", "/history"}, {"PUT", "/maintenance"}, {"GET", "/maintenance/history"},
		{"GET", "/webhooks"}, {"POST", "/webhooks"},
	} {
		t.Run(endpoint.method+endpoint.suffix, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			q.On("FindResourceById", mock.Anything, resourceID).Return(db.ClaimctlResource{ID: resourceID, SpaceID: spaceID}, nil)
			q.On("HasSpacePermission", mock.Anything, mock.Anything).Return(false, nil)
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				setAuthorization(c, services.Authorization{User: db.ClaimctlUser{ID: userID, Status: "active", Role: "user"}})
				return c.Next()
			})
			app.Use(AuthorizationMiddleware(q))
			path := "/api/resources/" + resourceID.String() + endpoint.suffix
			called := false
			app.Add(endpoint.method, path, func(c *fiber.Ctx) error { called = true; return c.SendStatus(200) })
			resp, err := app.Test(httptest.NewRequest(endpoint.method, path, nil))
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, 403, resp.StatusCode)
			require.False(t, called)
		})
	}
}

func TestAuthorizationMiddleware_UserReadsRequireExplicitCurrentAdminScope(t *testing.T) {
	for _, tc := range []struct {
		name, role, scope string
		status            int
	}{
		{"default admin token denied", "admin", "read", 403},
		{"explicit admin scope allowed", "admin", "users:read", 200},
		{"demoted token denied", "user", "users:read", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				setAuthorization(c, services.Authorization{User: db.ClaimctlUser{ID: testutils.TestUUID(1), Role: tc.role, Status: "active"}, TokenID: testutils.TestUUID(2), Scopes: []string{tc.scope, "reserve"}})
				return c.Next()
			})
			app.Use(AuthorizationMiddleware(&testutils.MockQuerier{}))
			called := false
			handler := func(c *fiber.Ctx) error { called = true; return c.SendStatus(200) }
			app.Get("/api/users", handler)
			app.Get("/api/users/find", handler)
			for _, path := range []string{"/api/users", "/api/users/find?email=user@example.invalid"} {
				called = false
				resp, err := app.Test(httptest.NewRequest("GET", path, nil))
				require.NoError(t, err)
				resp.Body.Close()
				require.Equal(t, tc.status, resp.StatusCode)
				require.Equal(t, tc.status == 200, called)
			}
		})
	}
}

func TestAuthorizationMiddleware_BrowserOnlyCannotBeGrantedByTokenScope(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		setAuthorization(c, services.Authorization{User: db.ClaimctlUser{ID: testutils.TestUUID(1), Status: "active", Role: "admin"}, TokenID: testutils.TestUUID(2), Scopes: []string{"browser-only"}})
		return c.Next()
	})
	app.Use(AuthorizationMiddleware(&testutils.MockQuerier{}))
	app.Get("/api/tokens", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	resp, err := app.Test(httptest.NewRequest("GET", "/api/tokens", nil))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 403, resp.StatusCode)
}

func TestAuthorizationMiddleware_TokenScopeAndCurrentRole(t *testing.T) {
	for _, tc := range []struct {
		name, role, scope string
		expected          int
	}{
		{"default token cannot mutate resources", "admin", "read", 403},
		{"explicit admin scope", "admin", "resources:write", 200},
		{"demoted token owner", "user", "resources:write", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			resourceID, spaceID := testutils.TestUUID(1), testutils.TestUUID(2)
			q.On("FindResourceById", mock.Anything, resourceID).Return(db.ClaimctlResource{SpaceID: spaceID}, nil).Maybe()
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				setAuthorization(c, services.Authorization{User: db.ClaimctlUser{ID: testutils.TestUUID(3), Status: "active", Role: tc.role}, TokenID: testutils.TestUUID(4), Scopes: []string{tc.scope}})
				return c.Next()
			})
			app.Use(AuthorizationMiddleware(q))
			path := "/api/resources/" + resourceID.String()
			app.Delete(path, func(c *fiber.Ctx) error { return c.SendStatus(200) })
			resp, err := app.Test(httptest.NewRequest("DELETE", path, nil))
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, tc.expected, resp.StatusCode)
		})
	}
}
