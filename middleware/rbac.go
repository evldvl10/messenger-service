package middleware

import (
	"messenger-service/database"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func RBAC() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)

		// Load policy from Database
		if err := database.Casbin().LoadPolicy(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Internal server error",
				"data":    nil,
			})
		}

		// Casbin enforces policy
		accepted, err := database.Casbin().Enforce(claims["id"].(string), c.OriginalURL(), c.Method())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Internal server error",
				"data":    nil,
			})
		}

		if !accepted {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized",
				"data":    nil,
			})
		}

		return c.Next()
	}
}
