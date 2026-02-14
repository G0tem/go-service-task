package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID `gorm:"primarykey;not null;type:char(36);"`
	Username       string    `gorm:"uniqueIndex;not null;size:50;" validate:"required,min=3,max=50" json:"username"`
	Email          string    `gorm:"uniqueIndex;not null;size:255;" validate:"required,email" json:"email"`
	EmailConfirmed bool      `gorm:"not null;column:email_confirmed;" json:"email_confirmed"`
	PasswordHash   string    `gorm:"column:password_hash;" validate:"required,min=6,max=50" json:"password"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (user *User) BeforeCreate(tx *gorm.DB) error {
	user.ID = uuid.New()
	return nil
}

func (user *User) TableName() string {
	return "users"
}
