package handler

import (
	"strings"
	"time"

	"github.com/G0tem/go-service-task/internal/model"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type JwtClaims struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Exp      time.Time `json:"exp"`
}

func JWTMiddleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := string(c.Request().Header.Peek("Authorization"))
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "missing Authorization header",
			})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "invalid Authorization header",
			})
		}

		tokenStr := strings.TrimSpace(parts[1])
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "invalid or expired token",
			})
		}

		claimsMap, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "invalid claims",
			})
		}

		var expTime time.Time
		if exp, ok := claimsMap["exp"].(float64); ok {
			expTime = time.Unix(int64(exp), 0)
		}

		claims := &JwtClaims{
			UserID:   asString(claimsMap["user_id"]),
			Username: asString(claimsMap["username"]),
			Email:    asString(claimsMap["email"]),
			Exp:      expTime,
		}

		c.Locals("claims", claims)
		return c.Next()
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (h *Handler) GetJWT(user *model.User) (string, error) {
	// Create the Claims
	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(h.cfg.SecretKey))

	return t, err
}
