package handler

import (
	"fmt"
	"strings"

	"github.com/G0tem/go-service-task/internal/model"
	"github.com/G0tem/go-service-task/internal/types"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// Login
// @Summary Login
// @Description Login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body types.LoginRequest true "username or email"
// @Success 200 {object} types.LoginSuccessResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Router /auth/login [post]
func (h *Handler) login(c *fiber.Ctx) error {
	var user *model.User
	input := new(types.LoginRequest)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Error on login request",
			Error:   err.Error(),
		})
	}

	err := *new(error)

	if validateEmail(input.Identity) {
		user, err = h.getUserByEmail(input.Identity)
	} else {
		user, err = h.getUserByUsername(input.Identity)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal Server Error",
			Error:   err.Error(),
		})
	} else if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(types.FailureResponse{
			Status:  "error",
			Message: "Invalid identity or password",
		})
	}

	if !CheckPasswordHash(input.Password, user.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(types.FailureResponse{
			Status:  "error",
			Message: "Invalid identity or password",
		})
	}

	token, err := h.GetJWT(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal Server Error",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.LoginSuccessResponse{
		Status: "ok",
		Data:   types.LoginSuccessData{Token: token},
	})
}

// Register
// @Summary Register
// @Description Register
// @Tags auth
// @Accept json
// @Produce json
// @Param request body types.RegisterRequest true "username or email"
// @Success 200 {object} types.LoginSuccessResponse
// @Failure 400 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Router /auth/register [post]
func (h *Handler) register(c *fiber.Ctx) error {
	input := new(types.RegisterRequest)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Error on register request",
			Error:   err.Error(),
		})
	}

	if !validateEmail(input.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "Invalid email address",
		})
	}

	if input.Password != input.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "Password and confirm password must be same",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Failed to hash password",
			Error:   err.Error(),
		})
	}

	user := model.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal Server Error (Create user)",
			Error:   err.Error(),
		})
	}

	token, err := h.GetJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal Server Error (user.GetJWT())",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(types.LoginSuccessResponse{
		Status: "Ok",
		Data:   types.LoginSuccessData{Token: token},
	})
}

// Refresh jwt token
// @Summary Refresh jwt token
// @Description Refresh jwt token
// @Tags auth
// @Produce json
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /auth/refresh [post]
func (h *Handler) refresh(c *fiber.Ctx) error {
	authorization := string(c.Request().Header.Peek("Authorization"))
	tokenString := strings.TrimSpace(strings.Replace(authorization, "Bearer", "", 1))
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal server error (refresh)",
			Error:   err.Error(),
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal server error (email field not present in jwt)",
			Error:   "email field not present in jwt",
		})
	}
	email := claims["email"]

	var (
		user model.User
	)
	// Find user by email
	tx := h.db.Find(&user, "Email = ?", email)
	if err := tx.Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Internal server error (user not found by email %v)", email),
			Error:   err.Error(),
		})
	}

	if user.ID == uuid.Nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal server error (user not found)",
			Error:   fmt.Sprintf("user with email %v not found", email),
		})
	}

	newToken, err := h.GetJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal Server Error",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.LoginSuccessResponse{
		Status: "ok",
		Data:   types.LoginSuccessData{Token: newToken},
	})
}

// GetMe
// @Summary Get current user info
// @Description Get current user ID and email
// @Tags auth
// @Produce json
// @Success 200 {object} types.GetMeResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /auth/get-me [get]
func (h *Handler) getMe(c *fiber.Ctx) error {
	var user model.User

	claims := c.Locals("claims").(*JwtClaims)
	log.Debug().
		Str("email", claims.Email).
		Time("exp", claims.Exp).
		Msg("Attempting to get user")

	tx := h.db.Where("email = ?", claims.Email).First(&user)
	if err := tx.Error; err != nil {
		log.Error().
			Err(err).
			Str("email", claims.Email).
			Msg("Failed to get user from database")
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "Internal server error",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.GetMeResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	})
}
