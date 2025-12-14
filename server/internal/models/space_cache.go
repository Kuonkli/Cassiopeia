package models

import (
	"gorm.io/datatypes"
	"time"
)

type SpaceCache struct {
	ID        uint           `gorm:"primaryKey"`
	Source    string         `gorm:"not null;index"`
	FetchedAt time.Time      `gorm:"not null;default:now()"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
}
