package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/services"
)

type fakeSetupUserService struct {
	services.UserService
	needsSetup bool
	createFn   func(context.Context, services.InitialAdminRequest) (*services.AdminUserResponse, error)
}

func (f *fakeSetupUserService) NeedsSetup(context.Context) (bool, error) {
	return f.needsSetup, nil
}

func (f *fakeSetupUserService) CreateInitialAdmin(ctx context.Context, req services.InitialAdminRequest) (*services.AdminUserResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return &services.AdminUserResponse{
		PublicUserResponse: services.PublicUserResponse{Email: req.Email, Name: req.Name, Role: "admin"},
	}, nil
}

func setupApp(t *testing.T, svc services.UserService, token string) *fiber.App {
	t.Helper()
	handler := NewSetupHandler(svc, token)
	app := fiber.New()
	app.Get("/setup/status", handler.GetStatus)
	app.Post("/setup/admin", handler.CreateInitialAdmin)
	return app
}

func setupRequest() *bytes.Buffer {
	body, _ := json.Marshal(map[string]string{
		"name":     "First Admin",
		"email":    "first@example.com",
		"password": "Valid1!pass",
	})
	return bytes.NewBuffer(body)
}

func TestSetupHandler_GetStatus(t *testing.T) {
	app := setupApp(t, &fakeSetupUserService{needsSetup: true}, "st_token")
	resp, err := app.Test(httptest.NewRequest("GET", "/setup/status", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]bool
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body["needsSetup"])
}

func TestSetupHandler_CreateInitialAdmin(t *testing.T) {
	t.Run("rejects a missing token", func(t *testing.T) {
		app := setupApp(t, &fakeSetupUserService{}, "st_token")
		resp, err := app.Test(httptest.NewRequest("POST", "/setup/admin", setupRequest()))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("rejects a wrong token", func(t *testing.T) {
		app := setupApp(t, &fakeSetupUserService{}, "st_token")
		req := httptest.NewRequest("POST", "/setup/admin", setupRequest())
		req.Header.Set("X-Setup-Token", "wrong")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("creates the admin and invalidates the token", func(t *testing.T) {
		app := setupApp(t, &fakeSetupUserService{}, "st_token")

		req := httptest.NewRequest("POST", "/setup/admin", setupRequest())
		req.Header.Set("X-Setup-Token", "st_token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "first@example.com", body["email"])
		assert.Equal(t, "admin", body["role"])

		req = httptest.NewRequest("POST", "/setup/admin", setupRequest())
		req.Header.Set("X-Setup-Token", "st_token")
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("returns conflict when setup already completed", func(t *testing.T) {
		svc := &fakeSetupUserService{
			createFn: func(context.Context, services.InitialAdminRequest) (*services.AdminUserResponse, error) {
				return nil, services.ErrSetupAlreadyCompleted
			},
		}
		app := setupApp(t, svc, "st_token")

		req := httptest.NewRequest("POST", "/setup/admin", setupRequest())
		req.Header.Set("X-Setup-Token", "st_token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})
}
