package controller

import (
	"messenger-service/database"
	"messenger-service/model"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type UserCreateOrderInput struct {
	Pair   int    `json:"pair"`
	Action string `json:"action"`
	Type   string `json:"type"`
	Volume string `json:"volume"`
	Price  string `json:"price"`
}

func UserProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	userModel := new(model.User)

	if err := database.Postgres.First(&userModel, claims["id"]).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": nil,
		"data": fiber.Map{
			"id":       userModel.ID,
			"created":  userModel.CreatedAt.Unix(),
			"username": userModel.Username,
			"email":    userModel.Email,
			"role":     userModel.Role,
			"otp":      userModel.Otp_enabled,
		},
	})
}
