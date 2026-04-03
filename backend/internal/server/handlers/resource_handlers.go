package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type NewDbResource = db.CreateNewResourceParams
type UpdateResourceType = db.UpdateResourceByIdParams

type ResourceHandler struct {
	resourceService services.ResourceService
	auditService services.AuditService
}

func NewResourceHandler(resourceService services.ResourceService, auditService services.AuditService) *ResourceHandler {
	return &ResourceHandler{
		resourceService: resourceService,
		auditService: auditService,
	}
}

// GetResources returns all resources
// @Summary Get all resources
// @Description Retrieve a list of all resources. Note: This might return all resources regardless of permissions in the current implementation.
// @Tags resources
// @Accept json
// @Produce json
// @Success 200 {array} ClaimctlResource
// @Failure 500 {object} map[string]string
// @Router /resources [get]
func (h *ResourceHandler) GetResources(c *fiber.Ctx) error {
	labelExprStr := c.Query("label_expr")
	labelFilter, err := utils.ParseLabelFilter(labelExprStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid label_expr format",
		})
	}

	resources, err := h.resourceService.GetAllResources(c.Context(), labelFilter)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get resources", err)
	}

	return c.JSON(resources)
}

// GetAllResourcesWithStatus returns all resources with their reservation status
// @Summary Get resources with status
// @Description Retrieve a list of all resources including their current reservation status. Filtered by user permissions.
// @Tags resources
// @Accept json
// @Produce json
// @Success 200 {array} services.ResourceWithStatus
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /resources/with-status [get]
func (h *ResourceHandler) GetAllResourcesWithStatus(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	labelExprStr := c.Query("label_expr")
	labelFilter, err := utils.ParseLabelFilter(labelExprStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid label_expr format",
		})
	}

	resourcesWithStatus, err := h.resourceService.GetAllResourcesWithStatusForUser(c.Context(), user.ID, h.isAdmin(c), labelFilter)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get resources with status", err)
	}

	return c.JSON(resourcesWithStatus)
}

// GetResourceByID returns one resource matching the ID
// @Summary Get a resource by ID
// @Description Retrieve details of a specific resource by its ID.
// @Tags resources
// @Accept json
// @Produce json
// @Param id path int true "Resource ID"
// @Success 200 {object} ClaimctlResource
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /resources/{id} [get]
func (h *ResourceHandler) GetResourceByID(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	resource, err := h.resourceService.GetResource(c.Context(), id, user.ID, h.isAdmin(c))

	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, "Access denied or resource not found", err)
	}
	return c.JSON(resource)
}

// GetResourceWithStatus returns one resource with its reservation status
// @Summary Get a resource with status by ID
// @Description Retrieve details of a specific resource including its current reservation status by its ID.
// @Tags resources
// @Accept json
// @Produce json
// @Param id path int true "Resource ID"
// @Success 200 {object} services.ResourceWithStatus
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /resources/{id}/with-status [get]
func (h *ResourceHandler) GetResourceWithStatus(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}
	resourceWithStatus, err := h.resourceService.GetResourceWithStatus(c.Context(), id, user.ID, h.isAdmin(c))

	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, "Access denied or resource not found", err)
	}
	return c.JSON(resourceWithStatus)
}

// CreateResource creates a single resource
// @Summary Create a new resource
// @Description Create a new resource. Requires admin privileges.
// @Tags resources
// @Accept json
// @Produce json
// @Param resource body db.CreateNewResourceParams true "New Resource Details"
// @Success 200 {object} ClaimctlResource
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /resources [post]
func (h *ResourceHandler) CreateResource(c *fiber.Ctx) error {
	var newResource = new(NewDbResource)
	var err error

	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to create a resource",
		})
	}

	if err := c.BodyParser(newResource); err != nil {
		return err
	}

	resource, err := h.resourceService.CreateResource(c.Context(), *newResource)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create resource", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "CREATE", "RESOURCE", resource.ID.String(), newResource, c.IP())
	}

	return c.JSON(resource)
}

// UpdateResource updates a single resource
// @Summary Update a resource
// @Description Update details of an existing resource. Requires admin privileges.
// @Tags resources
// @Accept json
// @Produce json
// @Param id path int true "Resource ID"
// @Param resource body db.UpdateResourceByIdParams true "Updated Resource Details"
// @Success 200 {object} ClaimctlResource
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /resources/{id} [patch]
func (h *ResourceHandler) UpdateResource(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to update a resource",
		})
	}
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	updatedResource := new(UpdateResourceType)
	if err := c.BodyParser(updatedResource); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not parse resource object",
		})
	}
	updatedResource.ID = id

	resource, err := h.resourceService.UpdateResource(c.Context(), *updatedResource)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to update resource", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "UPDATE", "RESOURCE", resource.ID.String(), updatedResource, c.IP())
	}

	return c.JSON(resource)
}

// DeleteResource deletes a single resource
// @Summary Delete a resource
// @Description Delete an existing resource. Requires admin privileges.
// @Tags resources
// @Accept json
// @Produce json
// @Param id path int true "Resource ID"
// @Success 200 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /resources/{id} [delete]
func (h *ResourceHandler) DeleteResource(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to delete a resource",
		})
	}
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	err = h.resourceService.DeleteResource(c.Context(), id)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete resource", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "DELETE", "RESOURCE", id.String(), nil, c.IP())
	}

	return nil
}

// isAdmin checks if the user is an admin
func (h *ResourceHandler) isAdmin(c *fiber.Ctx) bool {
	user, err := GetUserFromContext(c)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}

// SetMaintenanceModeRequest represents the request to set maintenance mode
type SetMaintenanceModeRequest struct {
	IsUnderMaintenance bool   `json:"is_under_maintenance"`
	Reason             string `json:"reason,omitempty"`
}

// SetMaintenanceMode updates the maintenance mode of a resource
// @Summary Set resource maintenance mode
// @Description Enable or disable maintenance mode for a resource. Requires admin privileges.
// @Tags resources
// @Accept json
// @Produce json
// @Param id path int true "Resource ID"
// @Param request body SetMaintenanceModeRequest true "Maintenance mode settings"
// @Success 200 {object} db.ClaimctlResource
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /resources/{id}/maintenance [put]
func (h *ResourceHandler) SetMaintenanceMode(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to change maintenance mode",
		})
	}

	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req SetMaintenanceModeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	resource, err := h.resourceService.SetMaintenanceMode(c.Context(), id, req.IsUnderMaintenance, user.ID, req.Reason)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to update maintenance mode", err)
	}

	return c.JSON(resource)
}

// GetMaintenanceHistory returns the maintenance history for a resource
// @Summary Get maintenance history
// @Description Retrieve the maintenance mode change history for a specific resource.
// @Tags resources
// @Accept json
// @Produce json
// @Param id path int true "Resource ID"
// @Success 200 {array} db.GetMaintenanceHistoryRow
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /resources/{id}/maintenance/history [get]
func (h *ResourceHandler) GetMaintenanceHistory(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	history, err := h.resourceService.GetMaintenanceHistory(c.Context(), id)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get maintenance history", err)
	}

	return c.JSON(history)
}
