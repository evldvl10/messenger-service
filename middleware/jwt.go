package middleware

import (
	"messenger-service/config"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
)

func JWT() fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: "HS512",
			Key:    []byte(config.Config("JWT_ACCESS_KEY")),
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if err.Error() == "Missing or malformed JWT" {
				return c.Status(fiber.StatusBadRequest).
					JSON(fiber.Map{
						"status":  "error",
						"message": "Missing or malformed JWT",
						"data":    nil,
					})
			}
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{
					"status":  "error",
					"message": "Invalid or expired JWT",
					"data":    nil,
				})
		},
	})
}
