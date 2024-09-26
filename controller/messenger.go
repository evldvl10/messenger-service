package controller

import (
	"encoding/base64"
	"messenger-service/database"
	"messenger-service/model"

	"github.com/gofiber/fiber/v2"
)

func MessengerMessageImage(c *fiber.Ctx) error {
	image := new(model.MessengerImage)
	database.Postgres.First(&image, c.AllParams()["id"])
	data, _ := base64.StdEncoding.DecodeString(image.Data)
	c.Set("Content-Type", "image/png")
	return c.Send([]byte(data))
}
