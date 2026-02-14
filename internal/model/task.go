package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Task struct {
	ID          uuid.UUID      `gorm:"primaryKey;type:char(36)" json:"id"`
	Title       string         `gorm:"size:255;not null;index" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	Status      string         `gorm:"size:32;not null;index" json:"status"` // todo, in_progress, done, cancelled.
	TeamID      uuid.UUID      `gorm:"type:char(36);not null;index" json:"team_id"`
	AssigneeID  uuid.UUID      `gorm:"type:char(36);index" json:"assignee_id"`
	CreatedBy   uuid.UUID      `gorm:"type:char(36);not null;index" json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Status == "" {
		t.Status = "todo"
	}
	return nil
}

func (t *Task) TableName() string {
	return "tasks"
}
