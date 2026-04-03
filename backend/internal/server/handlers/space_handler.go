package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type SpaceHandler struct {
	service services.SpaceService
	auditService services.AuditService
}

func NewSpaceHandler(service services.SpaceService, auditService services.AuditService) *SpaceHandler {
	return &SpaceHandler{
		service: service,
		auditService: auditService,
	}
}

func (h *SpaceHandler) GetSpaces(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	// Pass user ID and Admin status to service to filter spaces
	spaces, err := h.service.GetAllSpacesForUser(c.Context(), user.ID, h.isAdmin(c))
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get spaces", err)
	}
	return c.JSON(spaces)
}

func (h *SpaceHandler) GetSpace(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	space, err := h.service.GetSpace(c.Context(), id, user.ID, h.isAdmin(c))
	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, "Access denied", err)
	}

	return c.JSON(space)
}

type CreateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *SpaceHandler) CreateSpace(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	var req CreateSpaceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	space, err := h.service.CreateSpace(c.Context(), req.Name, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Failed to create space", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "CREATE", "SPACE", space.ID.String(), req, c.IP())
	}

	return c.Status(fiber.StatusCreated).JSON(space)
}

type UpdateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *SpaceHandler) UpdateSpace(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req UpdateSpaceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	space, err := h.service.UpdateSpace(c.Context(), id, req.Name, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Failed to update space", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "UPDATE", "SPACE", id.String(), req, c.IP())
	}

	return c.JSON(space)
}

func (h *SpaceHandler) DeleteSpace(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	if err := h.service.DeleteSpace(c.Context(), id); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Failed to delete space", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "DELETE", "SPACE", id.String(), nil, c.IP())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Permissions Handlers

func (h *SpaceHandler) GetSpacePermissions(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	permissions, err := h.service.GetSpacePermissions(c.Context(), id)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get permissions", err)
	}
	return c.JSON(permissions)
}

func (h *SpaceHandler) AddPermission(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	spaceID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req struct {
		GroupID *uuid.UUID `json:"groupId"`
		UserID *uuid.UUID `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := h.service.AddPermission(c.Context(), spaceID, req.GroupID, req.UserID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to add permission", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Permission added"})
}

func (h *SpaceHandler) RemovePermission(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	spaceID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req struct {
		GroupID *uuid.UUID `json:"groupId"`
		UserID *uuid.UUID `json:"userId"`
	}
	// Using BodyParser for DELETE is allowed but sometimes query params are preferred.
	// We'll stick to body for consistency with AddPermission unless it causes issues.
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := h.service.RemovePermission(c.Context(), spaceID, req.GroupID, req.UserID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to remove permission", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *SpaceHandler) isAdmin(c *fiber.Ctx) bool {
	user, err := GetUserFromContext(c)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}
