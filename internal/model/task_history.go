package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// История задач
type TaskHistory struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:char(36)" json:"id"`
	TaskID    uuid.UUID      `gorm:"type:char(36);not null;index" json:"task_id"`
	ChangedBy uuid.UUID      `gorm:"type:char(36);not null;index" json:"changed_by"`
	Field     string         `gorm:"size:64;not null" json:"field"`
	OldValue  string         `gorm:"type:text" json:"old_value"`
	NewValue  string         `gorm:"type:text" json:"new_value"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (h *TaskHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}

func (t *TaskHistory) TableName() string {
	return "task_historys"
}
