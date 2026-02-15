package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"github.com/G0tem/go-service-task/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CheckPasswordHash compare password with hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (h *Handler) getUserByEmail(e string) (*model.User, error) {
	var user model.User
	if err := h.db.Where(&model.User{Email: e}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (h *Handler) getUserByUsername(u string) (*model.User, error) {
	var user model.User
	if err := h.db.Where(&model.User{Username: u}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// Метод, который использует Circuit Breaker
func (h *Handler) sendInviteEmailWithBreaker(email string, teamID, userID uuid.UUID) error {
	// Execute оборачивает вызов в Circuit Breaker
	_, err := h.emailBreaker.Execute(func() (interface{}, error) {
		// Этот код выполняется только если цепь CLOSED или HALF-OPEN
		return nil, h.callEmailService(email, teamID, userID)
	})

	return err
}

// Отправляет запрос для "уведомления" о приглашении
func (h *Handler) callEmailService(email string, teamID, userID uuid.UUID) error {
	requestBody := map[string]interface{}{
		"to":      email,
		"team_id": teamID,
		"user_id": userID,
		"type":    "team_invitation",
	}

	jsonData, _ := json.Marshal(requestBody)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Post(
		h.cfg.MailUrl,
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return fmt.Errorf("email service connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("email service returned error: %d", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
