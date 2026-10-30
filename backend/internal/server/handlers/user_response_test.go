package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestUserResponsesAndAudit_NeverExposeCredentials(t *testing.T) {
	for _, tc := range []struct{ name, method, path, body string }{
		{"list", "GET", "/users", ""},
		{"find", "GET", "/users/find?email=visible@example.invalid", ""},
		{"create", "POST", "/users", `{"email":"visible@example.invalid","name":"Visible","password":"SecretPass1!","role":"user","status":"active"}`},
		{"update", "PATCH", "/users/" + testutils.TestUUID(2).String(), `{"name":"Changed"}`},
		{"channel update", "PUT", "/me/channel-config", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := &testutils.MockQuerier{}
			user := db.ClaimctlUser{ID: testutils.TestUUID(2), Email: "visible@example.invalid", Name: "Visible", Password: "$2a$10$stored-hash-must-never-appear", Role: "user", Status: "active", AuthProvider: "local"}
			q.On("FindAllUsers", mock.Anything).Return([]db.ClaimctlUser{user}, nil).Maybe()
			q.On("FindUserByEmail", mock.Anything, user.Email).Return(user, nil).Maybe()
			q.On("FindUserByIdForUpdate", mock.Anything, user.ID).Return(user, nil).Maybe()
			q.On("VerifyUserEmailIsUnique", mock.Anything, user.Email).Return(int64(0), nil).Maybe()
			q.On("CreateUser", mock.Anything, mock.Anything).Return(nil).Maybe()
			q.On("UpdateUserById", mock.Anything, mock.Anything).Return(nil).Maybe()
			q.On("UpdateUserChannelConfig", mock.Anything, mock.Anything).Return(user, nil).Maybe()
			audit := &testutils.MockAuditService{}
			audit.On("Log", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(payload interface{}) bool {
				raw, err := json.Marshal(payload)
				return err == nil && !strings.Contains(string(raw), `"password"`) && !strings.Contains(string(raw), "SecretPass1!") && !strings.Contains(string(raw), user.Password)
			}), mock.Anything).Maybe()
			h := NewUserHandler(services.NewUserService(q), nil, nil, audit, nil)
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("user_record", &db.ClaimctlUser{ID: user.ID, Role: "admin", Status: "active"})
				return c.Next()
			})
			app.Get("/users", h.GetUsers)
			app.Get("/users/find", h.GetUser)
			app.Post("/users", h.CreateUser)
			app.Patch("/users/:id", h.UpdateUser)
			app.Put("/me/channel-config", h.UpdateChannelConfig)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Contains(t, []int{200, 201}, resp.StatusCode)
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NotContains(t, string(raw), `"password"`)
			require.NotContains(t, string(raw), user.Password)
			require.NotContains(t, string(raw), "SecretPass1!")
			if tc.name != "update" {
				require.Contains(t, string(raw), `"authProvider":"local"`)
				require.Contains(t, string(raw), `"status":"active"`)
			}
			if tc.name == "create" || tc.name == "update" {
				require.Len(t, audit.Calls, 1)
			}
		})
	}
}
