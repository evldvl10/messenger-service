package utils

import (
	"messenger-service/config"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Tokens struct to describe tokens object.
type Tokens struct {
	Access  string
	Refresh string
}

// TokenMetadata struct to describe metadata in JWT.
type TokenMetadata struct {
	Id  string
	Otp bool
	Exp int64
}

// GenerateNewTokens func for generate a new Access & Refresh tokens.
func GenerateTokens(id string, otp bool) (*Tokens, error) {
	// Generate JWT Access token.
	accessToken, err := generateToken(
		id,
		otp,
		"JWT_ACCESS_EXPIRE",
		"JWT_ACCESS_KEY",
	)
	if err != nil {
		return nil, err
	}

	// Generate JWT Refresh token.
	refreshToken, err := generateToken(
		id,
		otp,
		"JWT_REFRESH_EXPIRE",
		"JWT_REFRESH_KEY",
	)
	if err != nil {
		return nil, err
	}

	return &Tokens{
		Access:  accessToken,
		Refresh: refreshToken,
	}, nil
}

func generateToken(id string, otp bool, expire string, key string) (string, error) {
	minutesCount, _ := strconv.Atoi(config.Config(expire))

	claims := jwt.MapClaims{}

	claims["id"] = id
	claims["otp"] = otp
	claims["exp"] = time.Now().Add(time.Minute * time.Duration(minutesCount)).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	t, err := token.SignedString([]byte(config.Config(key)))
	if err != nil {
		// Return error, it JWT token generation failed.
		return "", err
	}

	return t, nil
}

func CheckAndExtractTokenMetadata(token string, key string) (*TokenMetadata, error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Config(key)), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
		return &TokenMetadata{
			Id:  claims["id"].(string),
			Otp: claims["otp"].(bool),
			Exp: int64(claims["exp"].(float64)),
		}, nil
	}

	return nil, err
}
