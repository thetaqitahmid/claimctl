package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
	"strconv"
	"time"
)

func (h *ReservationHandler) Renew(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}
	var req struct {
		LeaseDuration *string `json:"leaseDuration"`
	}
	if err = c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	user, err := GetUserFromContext(c)
	if err != nil {
		return fiber.ErrUnauthorized
	}
	svc, ok := h.reservationService.(services.LifecycleService)
	if !ok {
		return fiber.ErrInternalServerError
	}
	result, err := svc.Renew(c.UserContext(), user.ID, id, req.LeaseDuration)
	if err != nil {
		return sendClaimError(c, err)
	}
	return c.JSON(result)
}
func (h *ReservationHandler) Bundle(c *fiber.Ctx) error {
	var req services.BundleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	user, err := GetUserFromContext(c)
	if err != nil {
		return fiber.ErrUnauthorized
	}
	svc, ok := h.reservationService.(services.LifecycleService)
	if !ok {
		return fiber.ErrInternalServerError
	}
	result, err := svc.Bundle(c.UserContext(), user.ID, req, c.Get("Idempotency-Key"))
	if err != nil {
		return sendClaimError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}
func window(c *fiber.Ctx) (int64, int64, error) {
	to := time.Now().Unix()
	from := to - 86400
	var err error
	if c.Query("to") != "" {
		to, err = strconv.ParseInt(c.Query("to"), 10, 64)
		if err != nil {
			return 0, 0, err
		}
		from = to - 86400
	}
	if c.Query("from") != "" {
		from, err = strconv.ParseInt(c.Query("from"), 10, 64)
	}
	return from, to, err
}
func (h *ReservationHandler) Metrics(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}
	from, to, err := window(c)
	if err != nil {
		return fiber.ErrBadRequest
	}
	svc, ok := h.reservationService.(services.LifecycleService)
	if !ok {
		return fiber.ErrInternalServerError
	}
	result, err := svc.Metrics(c.UserContext(), id, from, to)
	if err != nil {
		return sendClaimError(c, err)
	}
	return c.JSON(result)
}
func (h *ReservationHandler) AggregateMetrics(c *fiber.Ctx) error {
	from, to, err := window(c)
	if err != nil {
		return fiber.ErrBadRequest
	}
	svc, ok := h.reservationService.(services.LifecycleService)
	if !ok {
		return fiber.ErrInternalServerError
	}
	result, err := svc.AggregateMetrics(c.UserContext(), from, to)
	if err != nil {
		return sendClaimError(c, err)
	}
	return c.JSON(result)
}
