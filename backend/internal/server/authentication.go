package server

import (
	"crypto/rsa"
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
)

func authenticateJWT(c *fiber.Ctx, raw string, key *rsa.PublicKey, q db.Querier) error {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) { return key, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithExpirationRequired(), jwt.WithIssuer("claimctl"))
	if err != nil || token == nil || !token.Valid {
		return fiber.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fiber.ErrUnauthorized
	}
	idText, ok := claims["id"].(string)
	if !ok {
		return fiber.ErrUnauthorized
	}
	id, err := uuid.Parse(idText)
	if err != nil || id == uuid.Nil {
		return fiber.ErrUnauthorized
	}
	version, ok := claims["credential_version"].(string)
	if !ok || version == "" {
		return fiber.ErrUnauthorized
	}
	user, err := q.FindUserById(c.Context(), id)
	if err != nil || user.Status != "active" || subtle.ConstantTimeCompare([]byte(version), []byte(services.CredentialVersion(user.Password))) != 1 {
		return fiber.ErrUnauthorized
	}
	expiry, err := claims.GetExpirationTime()
	if err != nil || expiry == nil {
		return fiber.ErrUnauthorized
	}
	setAuthorization(c, services.Authorization{User: user, ExpiresAt: expiry.Time})
	return nil
}

func setAuthorization(c *fiber.Ctx, auth services.Authorization) {
	c.Locals("user_record", &auth.User)
	c.Locals("authorization", auth)
	services.BindAuthorization(c.Context(), auth)
	c.SetUserContext(services.WithAuthorization(c.UserContext(), auth))
}

func BearerAuthentication(service services.APITokenService, q db.Querier, key *rsa.PublicKey) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return c.Next()
		}
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		if strings.HasPrefix(parts[1], "res_") {
			auth, err := service.ValidateAuthorization(c.Context(), parts[1])
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
			}
			setAuthorization(c, *auth)
		} else if err := authenticateJWT(c, parts[1], key, q); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		c.Request().Header.DelCookie("access_token")
		c.Request().Header.DelCookie("refresh_token")
		c.Locals("verified_bearer", true)
		return c.Next()
	}
}

func CookieAuthentication(q db.Querier, key *rsa.PublicKey) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals("verified_bearer") != true {
			if err := authenticateJWT(c, c.Cookies("access_token"), key, q); err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
			}
		}
		return c.Next()
	}
}
