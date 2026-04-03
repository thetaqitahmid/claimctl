package server

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
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

// APITokenMiddleware checks if the user is authenticated via API Token
func APITokenMiddleware(service services.APITokenService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Check for Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next() // No token, let JWT middleware handle it
		}

		// 2. Check if it's a Bearer token and starts with "res_"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Next()
		}

		token := parts[1]
		if !strings.HasPrefix(token, "res_") {
			return c.Next()
		}

		// 3. Validate Token
		user, err := service.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid API Token"})
		}

		// set "user_record" in locals, and configure JWT middleware to skip if "user_record" is present.
		c.Locals("user_record", user)
		return c.Next()
	}
}
