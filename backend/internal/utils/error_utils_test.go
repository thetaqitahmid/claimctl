package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetUUIDParam_ReturnsParseErrorAfterSendingResponse(t *testing.T) {
	app := fiber.New()
	worked := false
	app.Get("/:id", func(c *fiber.Ctx) error {
		id, err := GetUUIDParam(c, "id")
		if err != nil {
			require.Equal(t, uuid.Nil, id)
			return nil
		}
		worked = true
		return c.SendStatus(200)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/malformed", nil))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 400, resp.StatusCode)
	require.False(t, worked)
	resp, err = app.Test(httptest.NewRequest("GET", "/"+uuid.NewString(), nil))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	require.True(t, worked)
}
