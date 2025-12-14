package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

type OSDRItem struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	DatasetID  string         `gorm:"uniqueIndex"`
	Title      string         `gorm:"type:text"`
	Status     string         `gorm:"type:varchar(50)"`
	UpdatedAt  *time.Time     `gorm:"index"`
	InsertedAt time.Time      `gorm:"not null;default:now()"`
	Raw        datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
}
