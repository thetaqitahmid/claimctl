package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

func CSRFMiddleware(storage fiber.Storage) fiber.Handler {
	ttl := time.Hour
	valid := func(token string) (bool, error) {
		if len(token) != 64 {
			return false, nil
		}
		if storage == nil {
			return false, fiber.ErrServiceUnavailable
		}
		data, err := storage.Get("csrf:" + token)
		return len(data) > 0, err
	}
	save := func(token string) error {
		if storage == nil {
			return fiber.ErrServiceUnavailable
		}
		return storage.Set("csrf:"+token, []byte{1}, ttl)
	}
	reject := func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "CSRF validation failed", "code": "CSRF_INVALID"})
	}
	return func(c *fiber.Ctx) error {
		if c.Locals("verified_bearer") == true {
			return c.Next()
		}
		token := c.Cookies("csrf_")
		exists, err := valid(token)
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		unsafe := c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && c.Method() != fiber.MethodOptions
		if unsafe {
			origin := c.Get("Origin")
			if origin != "" && origin != ExternalOrigin(c) {
				allowed := false
				for _, configured := range strings.Split(utils.GetEnv("CORS_ALLOWED_ORIGINS", ""), ",") {
					if strings.TrimSpace(configured) == origin {
						allowed = true
					}
				}
				if !allowed {
					return reject(c)
				}
			}
			if !exists || subtle.ConstantTimeCompare([]byte(token), []byte(c.Get("X-CSRF-Token"))) != 1 {
				return reject(c)
			}
		} else if !exists {
			bytes := make([]byte, 32)
			if _, err := rand.Read(bytes); err != nil {
				return fiber.ErrInternalServerError
			}
			token = hex.EncodeToString(bytes)
		}
		if err := save(token); err != nil {
			return fiber.ErrServiceUnavailable
		}
		c.Cookie(&fiber.Cookie{Name: "csrf_", Value: token, Path: "/", HTTPOnly: false, Secure: utils.GetEnvAsBool("COOKIE_SECURE", true), SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"), Expires: time.Now().Add(ttl)})
		c.Vary(fiber.HeaderCookie)
		c.Locals("csrf", token)
		return c.Next()
	}
}
