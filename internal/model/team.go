package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Team struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:char(36)" json:"id"`
	Name      string         `gorm:"uniqueIndex;size:255;not null" json:"name"`
	CreatedBy uuid.UUID      `gorm:"type:char(36);not null;index" json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Team) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (t *Team) TableName() string {
	return "teams"
}
