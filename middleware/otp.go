package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func OTP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)

		if claims["otp"].(bool) {
			return c.Status(fiber.StatusBadRequest).
				JSON(fiber.Map{
					"status":  "error",
					"message": "2FA required",
					"data":    nil,
				})
		}

		return c.Next()
	}
}
