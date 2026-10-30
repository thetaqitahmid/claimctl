package handlers

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

type contextReservationService struct {
	services.ReservationService
	received context.Context
}

func (s *contextReservationService) CreateReservation(ctx context.Context, userID, resourceID uuid.UUID, duration *string) (*db.ClaimctlReservation, error) {
	s.received = ctx
	return &db.ClaimctlReservation{UserID: userID, ResourceID: resourceID}, nil
}

func TestReservationHandler_UsesStableAuthorizedContext(t *testing.T) {
	user := db.ClaimctlUser{ID: testutils.TestUUID(1), Status: "active", Role: "user"}
	requestContext := services.WithAuthorization(context.Background(), services.Authorization{User: user})
	svc := &contextReservationService{}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.SetUserContext(requestContext)
		c.Locals("user_record", &user)
		return c.Next()
	})
	app.Post("/reservations", NewReservationHandler(svc).CreateReservation)
	req := httptest.NewRequest("POST", "/reservations", strings.NewReader(`{"resourceId":"`+testutils.TestUUID(2).String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 201, resp.StatusCode)
	require.Equal(t, requestContext, svc.received)
	auth, ok := services.AuthorizationFromContext(svc.received)
	require.True(t, ok)
	require.Equal(t, user.ID, auth.User.ID)
}
