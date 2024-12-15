package models

import (
	"time"

	"github.com/google/uuid"
)

// Team represents a team in the system
type Team struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"not null;unique"`
	Description string    `json:"description"`
	CreatedByID uuid.UUID `json:"created_by_id" gorm:"type:uuid;not null"`
	CreatedBy   User      `json:"created_by" gorm:"foreignKey:CreatedByID"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	Agents      []Agent   `json:"agents,omitempty" gorm:"many2many:agent_teams"`
}

// TableName specifies the table name for Team
func (Team) TableName() string {
	return "teams"
}
