package controller

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"

	"messenger-service/config"
	"messenger-service/database"
	"messenger-service/model"
	"messenger-service/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type AuthLoginInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AuthRenewTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthOtpSecretInput struct {
	Password string `json:"password"`
}

type AuthOtpVerifyInput struct {
	Token string `json:"token"`
}

type AuthOtpValidateInput struct {
	Token string `json:"token"`
}

type AuthOtpDisableInput struct {
	Password string `json:"password"`
	Token    string `json:"token"`
}

func AuthSignup(c *fiber.Ctx) error {
	user := new(model.User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Review your input",
			"data":    nil,
		})
	}

	// If existed email is found, return error
	if count := database.Postgres.
		Where(&model.User{Email: user.Email}).
		First(new(model.User)).
		RowsAffected; count > 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Email is already registered",
			"data":    nil,
		})
	}

	// If existed username is found, return error
	if count := database.Postgres.
		Where(&model.User{Username: user.Username}).
		First(new(model.User)).
		RowsAffected; count > 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Username is already registered",
			"data":    nil,
		})
	}

	// Generate hash from password.
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})

	}
	user.Password = string(hash)

	// Generate OTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.Config("OTP_ISSUER"),
		AccountName: user.Email,
		SecretSize:  15,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}
	user.Otp_secret = key.Secret()

	// Set user role
	user.Role = "user"

	// Save user to database
	if err := database.Postgres.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	// Add casbin policy
	database.Casbin().AddGroupingPolicy(fmt.Sprint(user.ID), user.Role)

	// Response
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": nil,
		"data": fiber.Map{
			"id": user.ID,
		},
	})
}

func AuthSignin(c *fiber.Ctx) error {
	input := new(AuthLoginInput)
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	userModel, err := new(model.User), *new(error)

	_, errParse := mail.ParseAddress(input.Login)
	if errParse == nil {
		err = database.Postgres.Where(&model.User{Email: input.Login}).First(&userModel).Error
	} else {
		err = database.Postgres.Where(&model.User{Username: input.Login}).First(&userModel).Error
	}

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid login or password",
			"data":    nil,
		})
	}

	idStr := strconv.FormatUint(uint64(userModel.ID), 10)

	if err := bcrypt.CompareHashAndPassword([]byte(userModel.Password), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid identity or password",
			"data":    nil,
		})
	}

	// Generate JWT Access & Refresh tokens
	tokens, err := utils.GenerateTokens(idStr, userModel.Otp_enabled)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	if err := database.Redis[0].Set(context.Background(), idStr, tokens.Refresh, 0).Err(); err != nil {
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
			"access":  tokens.Access,
			"refresh": tokens.Refresh,
			"2fa":     userModel.Otp_enabled,
		},
	})
}

func AuthTokenRenew(c *fiber.Ctx) error {
	renew := &AuthRenewTokenInput{}
	if err := c.BodyParser(renew); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	claims, err := utils.CheckAndExtractTokenMetadata(renew.RefreshToken, "JWT_REFRESH_KEY")
	if err != nil {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token",
			"data":    nil,
		})
	}

	userToken, err := database.Redis[0].Get(context.Background(), claims.Id).Result()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	if userToken != renew.RefreshToken {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized, your refresh token was already used",
			"data":    nil,
		})
	}

	// Generate JWT Access & Refresh tokens
	tokens, err := utils.GenerateTokens(claims.Id, claims.Otp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	// Save refresh token to Redis
	if err := database.Redis[0].Set(context.Background(), claims.Id, tokens.Refresh, 0).Err(); err != nil {
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
			"access":  tokens.Access,
			"refresh": tokens.Refresh,
			"2fa":     claims.Otp,
		},
	})
}

func AuthOtpSecret(c *fiber.Ctx) error {
	secret := &AuthOtpSecretInput{}
	if err := c.BodyParser(secret); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

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

	if err := bcrypt.CompareHashAndPassword([]byte(userModel.Password), []byte(secret.Password)); err != nil {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid password",
			"data":    nil,
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": nil,
		"data": fiber.Map{
			"secret": userModel.Otp_secret,
			"url": fmt.Sprintf("otpauth://totp/%s:%s?algorithm=SHA1&digits=6&issuer=%s&period=30&secret=%s",
				config.Config("OTP_ISSUER"),
				userModel.Email,
				config.Config("OTP_ISSUER"),
				userModel.Otp_secret,
			),
		},
	})
}

func AuthOtpVerify(c *fiber.Ctx) error {
	verify := &AuthOtpVerifyInput{}
	if err := c.BodyParser(verify); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

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

	if userModel.Otp_enabled {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Verification has already been performed earlier",
			"data":    nil,
		})
	}

	valid := totp.Validate(verify.Token, userModel.Otp_secret)
	if !valid {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token",
			"data":    nil,
		})
	}

	userModel.Otp_enabled = true
	database.Postgres.Save(&userModel)

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": nil,
		"data":    nil,
	})
}

func AuthOtpValidate(c *fiber.Ctx) error {
	validate := &AuthOtpValidateInput{}
	if err := c.BodyParser(validate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

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

	if !userModel.Otp_enabled {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "2FA has been disabled",
			"data":    nil,
		})
	}

	valid := totp.Validate(validate.Token, userModel.Otp_secret)
	if !valid {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token",
			"data":    nil,
		})
	}

	// Generate JWT Access & Refresh tokens
	tokens, err := utils.GenerateTokens(claims["id"].(string), false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

	// Save refresh token to Redis
	if err := database.Redis[0].Set(context.Background(), claims["id"].(string), tokens.Refresh, 0).Err(); err != nil {
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
			"access":  tokens.Access,
			"refresh": tokens.Refresh,
		},
	})
}

func AuthOtpDisable(c *fiber.Ctx) error {
	disable := &AuthOtpDisableInput{}
	if err := c.BodyParser(disable); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal server error",
			"data":    nil,
		})
	}

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

	if !userModel.Otp_enabled {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "2fa not enabled",
			"data":    nil,
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userModel.Password), []byte(disable.Password)); err != nil {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid password",
			"data":    nil,
		})
	}

	valid := totp.Validate(disable.Token, userModel.Otp_secret)
	if !valid {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token",
			"data":    nil,
		})
	}

	userModel.Otp_enabled = false
	database.Postgres.Save(&userModel)

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": nil,
		"data":    nil,
	})
}
