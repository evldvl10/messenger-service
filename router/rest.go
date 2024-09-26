package router

import (
	"messenger-service/controller"
	"messenger-service/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func Rest(app *fiber.App) {
	api := app.Group("/v1", logger.New())

	// Messenger
	messenger := api.Group("/messenger")
	messenger.Get("/image/:id", controller.MessengerMessageImage)

	// Auth
	auth := api.Group("/auth")
	auth.Post("/signup", controller.AuthSignup)
	auth.Post("/signin", controller.AuthSignin)
	auth.Post("/token/renew", controller.AuthTokenRenew)
	auth.Post("/2fa/secret", middleware.JWT(), middleware.OTP(), controller.AuthOtpSecret)
	auth.Post("/2fa/verify", middleware.JWT(), middleware.OTP(), controller.AuthOtpVerify)
	auth.Post("/2fa/validate", middleware.JWT(), controller.AuthOtpValidate)
	auth.Post("/2fa/disable", middleware.JWT(), middleware.OTP(), controller.AuthOtpDisable)

	// User
	user := api.Group("/user", middleware.JWT(), middleware.OTP())
	user.Get("/profile", controller.UserProfile)

	// Admin
	// admin := api.Group("/admin", middleware.JWT(), middleware.OTP(), middleware.RBAC())
	// admin.Post("/users", controller.AdminUsers)
}
