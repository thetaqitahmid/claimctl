package handlers

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
)

func (h *ReservationHandler) Claim(c *fiber.Ctx) error {
	var req services.ClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	user, err := GetUserFromContext(c)
	if err != nil {
		return fiber.ErrUnauthorized
	}
	service, ok := h.reservationService.(services.ClaimService)
	if !ok {
		return fiber.ErrInternalServerError
	}
	result, err := service.Claim(c.UserContext(), user.ID, req, c.Get("Idempotency-Key"))
	if err != nil {
		return sendClaimError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func sendClaimError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "Failed to create reservation"
	if errors.Is(err, services.ErrInvalidClaim) {
		status = fiber.StatusBadRequest
		message = err.Error()
	}
	if errors.Is(err, services.ErrAccessDenied) {
		status = fiber.StatusForbidden
		message = "Access denied"
	}
	if errors.Is(err, services.ErrClaimConflict) {
		status = fiber.StatusConflict
		message = err.Error()
	}
	return c.Status(status).JSON(fiber.Map{"error": message})
}

func (h *ReservationHandler) Policy(c *fiber.Ctx) error {
	service, ok := h.reservationService.(services.ClaimService)
	if !ok {
		return fiber.ErrInternalServerError
	}
	return c.JSON(service.Policy())
}
