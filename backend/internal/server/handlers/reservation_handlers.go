package handlers

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"

	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type ReservationHandler struct {
	reservationService services.ReservationService
}

func NewReservationHandler(reservationService services.ReservationService) *ReservationHandler {
	return &ReservationHandler{reservationService: reservationService}
}

// CreateReservation creates a new reservation
// @Summary Create a reservation
// @Description Create a new reservation for a resource.
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body object{resourceId=int} true "Reservation Request"
// @Success 201 {object} ClaimctlReservation
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /reservations [post]
func (h *ReservationHandler) CreateReservation(c *fiber.Ctx) error {
	type CreateReservationRequest struct {
		ResourceID uuid.UUID `json:"resourceId"`
	}

	var req CreateReservationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userID := user.ID

	reservation, err := h.reservationService.CreateReservation(c.Context(), userID, req.ResourceID, nil)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Failed to create reservation", err)
	}

	return c.Status(fiber.StatusCreated).JSON(reservation)
}

// CreateTimedReservation creates a new reservation with a specific duration
// @Summary Create a timed reservation
// @Description Create a new reservation for a resource with a specific duration.
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body object{resourceId=int,duration=string} true "Timed Reservation Request"
// @Success 201 {object} ClaimctlReservation
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /reservations/timed [post]
func (h *ReservationHandler) CreateTimedReservation(c *fiber.Ctx) error {
	type CreateTimedReservationRequest struct {
		ResourceID uuid.UUID  `json:"resourceId"`
		Duration   string `json:"duration"` // e.g. "1h", "30m"
	}

	var req CreateTimedReservationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userID := user.ID

	reservation, err := h.reservationService.CreateReservation(c.Context(), userID, req.ResourceID, &req.Duration)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Failed to create timed reservation", err)
	}

	return c.Status(fiber.StatusCreated).JSON(reservation)
}

// ActivateReservation activates a pending reservation (admin/resource owner only)
// @Summary Activate a reservation
// @Description Activate a pending reservation. Only accessible by admin or resource owner.
// @Tags reservations
// @Accept json
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} ClaimctlReservation
// @Failure 400 {object} map[string]string
// @Router /reservations/{id}/activate [patch]
func (h *ReservationHandler) ActivateReservation(c *fiber.Ctx) error {
	reservationID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	reservation, err := h.reservationService.ActivateReservation(c.Context(), reservationID)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Failed to activate reservation", err)
	}

	return c.JSON(reservation)
}

// CompleteReservation completes an active reservation (reservation owner only)
// @Summary Complete a reservation
// @Description Complete an active reservation. Only accessible by reservation owner or admin.
// @Tags reservations
// @Accept json
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /reservations/{id}/complete [patch]
func (h *ReservationHandler) CompleteReservation(c *fiber.Ctx) error {
	reservationID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	// Get user ID from context
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userID := user.ID

	// Get reservation to check ownership
	reservation, err := h.reservationService.GetReservation(c.Context(), reservationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Reservation not found",
		})
	}

	// Check if user owns this reservation or is admin
	if reservation.UserID != userID && user.Role != "admin" {
		fmt.Printf("Authorization failed: UserID %d does not match Reservation OwnerID %d and User is not Admin\n", userID, reservation.UserID)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only reservation owner or admin can complete it",
		})
	}

	err = h.reservationService.CompleteReservation(c.Context(), reservationID)
	if err != nil {
		// Use err.Error() to return the specific service error (e.g. status mismatch)
		return utils.SendError(c, fiber.StatusBadRequest, err.Error(), err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Reservation completed successfully",
	})
}

// CancelReservation cancels a reservation
// @Summary Cancel a reservation
// @Description Cancel a pending or active reservation.
// @Tags reservations
// @Accept json
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /reservations/{id}/cancel [patch]
func (h *ReservationHandler) CancelReservation(c *fiber.Ctx) error {
	reservationID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	// Get user ID from context
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userID := user.ID

	err = h.reservationService.CancelReservation(c.Context(), reservationID, userID)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error(), err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Reservation cancelled successfully",
	})
}

// CancelAllReservations cancels all reservations for a resource (admin only)
func (h *ReservationHandler) CancelAllReservations(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	// Check if user is admin
	user, err := GetUserFromContext(c)
	if err != nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can cancel all reservations",
		})
	}

	err = h.reservationService.CancelAllForResource(c.Context(), resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to cancel reservations", err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "All reservations cancelled successfully",
	})
}

// GetReservation retrieves a specific reservation
func (h *ReservationHandler) GetReservation(c *fiber.Ctx) error {
	reservationID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	// Get user ID from context
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userID := user.ID

	reservation, err := h.reservationService.GetReservation(c.Context(), reservationID)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Reservation not found", err)
	}

	if reservation.UserID != userID && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	return c.JSON(reservation)
}

// GetUserReservations retrieves all reservations for the authenticated user
func (h *ReservationHandler) GetUserReservations(c *fiber.Ctx) error {
	// Get user ID from context
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userID := user.ID

	reservations, err := h.reservationService.GetUserReservations(c.Context(), userID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get user reservations", err)
	}

	return c.JSON(reservations)
}

// GetResourceReservations retrieves all reservations for a specific resource
func (h *ReservationHandler) GetResourceReservations(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	reservations, err := h.reservationService.GetResourceReservations(c.Context(), resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get resource reservations", err)
	}

	return c.JSON(reservations)
}

// GetAllReservations retrieves all reservations (admin only)
func (h *ReservationHandler) GetAllReservations(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil || user.Role != "admin" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated or not admin",
		})
	}

	reservations, err := h.reservationService.GetAllReservations(c.Context())
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get all reservations", err)
	}

	return c.JSON(reservations)
}

// GetNextInQueue retrieves the next reservation in queue for a resource
func (h *ReservationHandler) GetNextInQueue(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	reservation, err := h.reservationService.GetNextInQueue(c.Context(), resourceID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No reservations in queue",
		})
	}

	return c.JSON(reservation)
}

// GetUserQueuePosition retrieves user's queue position for a resource
func (h *ReservationHandler) GetUserQueuePosition(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	// Get user ID from context
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	userIDInt := user.ID

	position, err := h.reservationService.GetUserQueuePosition(c.Context(), userIDInt, resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "User not found in queue", err)
	}

	return c.JSON(fiber.Map{
		"queuePosition": position,
	})
}

// ProcessQueue manually triggers queue processing for a resource
func (h *ReservationHandler) ProcessQueue(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	err = h.reservationService.ProcessQueue(c.Context(), resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to process queue", err)
	}

	return c.JSON(fiber.Map{
		"message":   "Queue processed successfully",
		"timestamp": time.Now().Unix(),
	})
}

// GetQueueForResource retrieves the current queue (active + pending) for a resource
func (h *ReservationHandler) GetQueueForResource(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	queue, err := h.reservationService.GetQueueForResource(c.Context(), resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get queue for resource", err)
	}

	return c.JSON(queue)
}
