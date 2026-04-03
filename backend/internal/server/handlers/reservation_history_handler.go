package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type ReservationHistoryHandler struct {
	reservationHistoryService services.ReservationHistoryService
}

func NewReservationHistoryHandler(reservationHistoryService services.ReservationHistoryService) *ReservationHistoryHandler {
	return &ReservationHistoryHandler{reservationHistoryService: reservationHistoryService}
}

// GetUserHistory retrieves the reservation history for the authenticated user
func (h *ReservationHistoryHandler) GetUserHistory(c *fiber.Ctx) error {
	// Get user ID from context
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	history, err := h.reservationHistoryService.GetUserHistory(c.Context(), user.ID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get user history", err)
	}

	return c.JSON(history)
}

// GetResourceHistory retrieves the reservation history for a specific resource (Admin only)
func (h *ReservationHistoryHandler) GetResourceHistory(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	history, err := h.reservationHistoryService.GetResourceHistory(c.Context(), resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get resource history", err)
	}

	return c.JSON(history)
}
