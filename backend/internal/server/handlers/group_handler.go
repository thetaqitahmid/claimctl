package handlers

import (
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type GroupHandler struct {
	groupService services.GroupService
	auditService services.AuditService
}

func NewGroupHandler(groupService services.GroupService, auditService services.AuditService) *GroupHandler {
	return &GroupHandler{
		groupService: groupService,
		auditService: auditService,
	}
}

// CreateGroup creates a new group
func (h *GroupHandler) CreateGroup(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	group, err := h.groupService.CreateGroup(c.Context(), req.Name, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create group", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "CREATE", "GROUP", group.ID.String(), req, c.IP())
	}

	return c.Status(fiber.StatusCreated).JSON(group)
}

// GetGroup returns a group by ID
func (h *GroupHandler) GetGroup(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	group, err := h.groupService.GetGroup(c.Context(), id)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Group not found", err)
	}

	return c.JSON(group)
}

// ListGroups lists all groups
func (h *GroupHandler) ListGroups(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	groups, err := h.groupService.ListGroups(c.Context())
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to list groups", err)
	}

	return c.JSON(groups)
}

// UpdateGroup updates a group
func (h *GroupHandler) UpdateGroup(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	group, err := h.groupService.UpdateGroup(c.Context(), id, req.Name, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to update group", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "UPDATE", "GROUP", id.String(), req, c.IP())
	}

	return c.JSON(group)
}

// DeleteGroup deletes a group
func (h *GroupHandler) DeleteGroup(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	if err := h.groupService.DeleteGroup(c.Context(), id); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete group", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "DELETE", "GROUP", id.String(), nil, c.IP())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AddUserToGroup adds a user to a group
func (h *GroupHandler) AddUserToGroup(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	groupID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req struct {
		UserID uuid.UUID `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := h.groupService.AddUserToGroup(c.Context(), groupID, req.UserID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to add user to group", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User added to group"})
}

// RemoveUserFromGroup removes a user from a group
func (h *GroupHandler) RemoveUserFromGroup(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	groupID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	userID, err := utils.GetUUIDParam(c, "userId")
	if err != nil {
		return nil
	}

	if err := h.groupService.RemoveUserFromGroup(c.Context(), groupID, userID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to remove user from group", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListGroupMembers lists members of a group
func (h *GroupHandler) ListGroupMembers(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	groupID, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	members, err := h.groupService.ListGroupMembers(c.Context(), groupID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to list group members", err)
	}

	return c.JSON(members)
}

func (h *GroupHandler) isAdmin(c *fiber.Ctx) bool {
	user, err := GetUserFromContext(c)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}
