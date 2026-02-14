package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeamMember represents membership of a user in a team with a role.
type TeamMember struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:char(36)" json:"id"`
	TeamID    uuid.UUID      `gorm:"type:char(36);not null;index:idx_team_user,unique" json:"team_id"`
	UserID    uuid.UUID      `gorm:"type:char(36);not null;index:idx_team_user,unique" json:"user_id"`
	Role      string         `gorm:"size:32;not null;index" json:"role"` // Role: "owner" или "member".
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (tm *TeamMember) BeforeCreate(tx *gorm.DB) error {
	if tm.ID == uuid.Nil {
		tm.ID = uuid.New()
	}
	return nil
}

func (tm *TeamMember) TableName() string {
	return "team_members"
}
