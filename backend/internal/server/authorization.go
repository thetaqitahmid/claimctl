package server

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
)

func requiredTokenScope(path, method string) string {
	read := method == fiber.MethodGet || method == fiber.MethodHead
	switch {
	case strings.HasPrefix(path, "/api/tokens"), path == "/api/user/password", strings.Contains(path, "channel-config"), path == "/api/me/test-email":
		return "browser-only"
	case path == "/api/admin/metrics":
		return "metrics:read"
	case strings.HasPrefix(path, "/api/admin/backup"):
		return "backup:read"
	case strings.HasPrefix(path, "/api/admin/restore"):
		return "restore:write"
	case strings.HasPrefix(path, "/api/admin/audit-logs"):
		return "audit:read"
	case strings.HasPrefix(path, "/api/admin/settings"):
		if read {
			return "settings:read"
		}
		return "settings:write"
	case strings.HasPrefix(path, "/api/secrets"):
		if read {
			return "secrets:read"
		}
		return "secrets:write"
	case strings.HasPrefix(path, "/api/admin/resources"), strings.HasPrefix(path, "/api/admin/reservations"):
		return "reservations:write"
	case strings.HasPrefix(path, "/api/groups"):
		return "groups:write"
	case strings.HasPrefix(path, "/api/webhooks"), strings.Contains(path, "/webhooks"):
		return "webhooks:write"
	case strings.HasPrefix(path, "/api/users"):
		if read {
			return "users:read"
		}
		return "users:write"
	case strings.HasPrefix(path, "/api/spaces"):
		if !read {
			return "spaces:write"
		}
	case strings.HasPrefix(path, "/api/resources"):
		if strings.Contains(path, "/health/") && !read {
			return "health:write"
		}
		if strings.Contains(path, "/queue/process") {
			return "reservations:write"
		}
		if !read {
			return "resources:write"
		}
	case strings.HasPrefix(path, "/api/reservations"):
		if strings.HasSuffix(path, "/activate") {
			return "reservations:write"
		}
		if !read {
			return "reserve"
		}
	case path == "/api/me", path == "/api/me/history", path == "/api/events", path == "/api/me/preferences":
		if read {
			return "read"
		}
		return "browser-only"
	default:
		return "browser-only"
	}
	if read {
		return "read"
	}
	return "browser-only"
}

func AuthorizationMiddleware(q db.Querier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth, ok := c.Locals("authorization").(services.Authorization)
		if !ok || auth.User.Status != "active" {
			return fiber.ErrUnauthorized
		}
		path := strings.TrimSuffix(c.Path(), "/")
		read := c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead
		if auth.TokenID != uuid.Nil {
			scope := requiredTokenScope(path, c.Method())
			if _, known := services.TokenScopes[scope]; !known {
				return fiber.ErrForbidden
			}
			found := false
			for _, s := range auth.Scopes {
				if s == scope {
					found = true
				}
			}
			if !found {
				return fiber.ErrForbidden
			}
			if admin := services.TokenScopes[scope]; admin && auth.User.Role != "admin" {
				return fiber.ErrForbidden
			}
			if len(auth.SpaceIDs) > 0 && ((strings.HasPrefix(path, "/api/spaces") && !read && len(strings.Split(path, "/")) == 3) || strings.HasPrefix(path, "/api/admin") || strings.HasPrefix(path, "/api/secrets") || strings.HasPrefix(path, "/api/webhooks") || strings.HasPrefix(path, "/api/users") || strings.HasPrefix(path, "/api/groups")) {
				return fiber.ErrForbidden
			}
		}
		parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
		resourceIndex := -1
		if len(parts) > 1 && parts[0] == "resources" {
			resourceIndex = 1
		}
		if len(parts) > 2 && parts[0] == "admin" && parts[1] == "resources" {
			resourceIndex = 2
		}
		if resourceIndex >= 0 && parts[resourceIndex] != "with-status" {
			id, err := uuid.Parse(parts[resourceIndex])
			if err != nil {
				return fiber.ErrBadRequest
			}
			if _, err := services.AuthorizeResource(c.Context(), q, id, auth.User.ID, auth.User.Role == "admin"); err != nil {
				return fiber.ErrForbidden
			}
			if !read && (strings.Contains(path, "/health/") || strings.HasSuffix(path, "/maintenance") || strings.HasSuffix(path, "/queue/process")) && auth.User.Role != "admin" {
				return fiber.ErrForbidden
			}
		}
		if len(parts) > 1 && parts[0] == "reservations" && parts[1] != "timed" && parts[1] != "claim" && parts[1] != "policy" && parts[1] != "bundle" {
			id, err := uuid.Parse(parts[1])
			if err != nil {
				return fiber.ErrBadRequest
			}
			res, err := q.FindReservationById(c.Context(), id)
			if err != nil {
				return fiber.ErrForbidden
			}
			if res.UserID != auth.User.ID && auth.User.Role != "admin" {
				return fiber.ErrForbidden
			}
			if _, err := services.AuthorizeResource(c.Context(), q, res.ResourceID, auth.User.ID, auth.User.Role == "admin"); err != nil {
				return fiber.ErrForbidden
			}
		}
		if len(parts) > 1 && parts[0] == "spaces" {
			id, err := uuid.Parse(parts[1])
			if err != nil {
				return fiber.ErrBadRequest
			}
			if services.AuthorizeSpace(c.Context(), q, id, auth.User.ID, auth.User.Role == "admin") != nil {
				return fiber.ErrForbidden
			}
			if len(parts) > 2 && parts[2] == "permissions" && auth.User.Role != "admin" {
				return fiber.ErrForbidden
			}
		}
		if !read && parts[0] == "resources" && (len(parts) == 1 || len(parts) == 2) {
			var req struct {
				SpaceID uuid.UUID `json:"spaceId"`
			}
			if c.Method() != fiber.MethodDelete {
				if err := c.BodyParser(&req); err != nil {
					return fiber.ErrBadRequest
				}
				if req.SpaceID == uuid.Nil && len(parts) == 1 {
					space, err := q.GetSpaceByName(c.Context(), "Default Space")
					if err != nil {
						return fiber.ErrForbidden
					}
					req.SpaceID = space.ID
				}
				if req.SpaceID != uuid.Nil && services.AuthorizeSpace(c.Context(), q, req.SpaceID, auth.User.ID, auth.User.Role == "admin") != nil {
					return fiber.ErrForbidden
				}
			}
		}
		return c.Next()
	}
}
