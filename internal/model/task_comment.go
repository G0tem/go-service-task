package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskComment stores comments for tasks.
type TaskComment struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:char(36)" json:"id"`
	TaskID    uuid.UUID      `gorm:"type:char(36);not null;index" json:"task_id"`
	UserID    uuid.UUID      `gorm:"type:char(36);not null;index" json:"user_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *TaskComment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (t *TaskComment) TableName() string {
	return "task_comments"
}
