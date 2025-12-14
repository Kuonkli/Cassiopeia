package models

import (
	"time"
)

type Telemetry struct {
	ID          uint      `gorm:"primaryKey"`
	RecordedAt  time.Time `gorm:"not null"`
	Voltage     float64   `gorm:"type:numeric(6,2);not null"`
	Temperature float64   `gorm:"type:numeric(6,2);not null"`
	SourceFile  string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}
