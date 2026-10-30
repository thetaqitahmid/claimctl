package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/utils"

	"github.com/thetaqitahmid/claimctl/internal/connection"
)

// DBMiddleware to set the database session
func DBMiddleware(db *connection.DBConn) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("db", db)
		return c.Next()
	}
}

// AdminMiddleware checks if the user is an admin
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := handlers.GetUserFromContext(c)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
		}

		if user.Role != "admin" {
			return utils.SendError(c, fiber.StatusForbidden, "Access denied: Admin privileges required", nil)
		}

		return c.Next()
	}
}
