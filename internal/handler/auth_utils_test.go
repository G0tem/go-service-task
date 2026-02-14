package handler

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func Test_validateEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"a@b.co", true},
		{"invalid", false},
		{"", false},
		{"@nodomain.com", false},
		{"noatsign.com", false},
	}
	for _, tt := range tests {
		got := validateEmail(tt.email)
		if got != tt.want {
			t.Errorf("validateEmail(%q) = %v, want %v", tt.email, got, tt.want)
		}
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "secret123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if !CheckPasswordHash(password, string(hash)) {
		t.Error("CheckPasswordHash: same password should match hash")
	}
	if CheckPasswordHash("wrong", string(hash)) {
		t.Error("CheckPasswordHash: wrong password should not match")
	}
	if CheckPasswordHash(password, "not-a-valid-hash") {
		t.Error("CheckPasswordHash: invalid hash should not match")
	}
}
