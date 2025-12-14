package models

import (
	"time"

	"gorm.io/datatypes"
)

type ISSLog struct {
	ID        uint           `gorm:"primaryKey"`
	FetchedAt time.Time      `gorm:"not null;default:now()"`
	SourceURL string         `gorm:"not null"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
}

type ISSTrend struct {
	Movement    bool       `json:"movement"`
	DeltaKm     float64    `json:"delta_km"`
	DtSec       float64    `json:"dt_sec"`
	VelocityKmh *float64   `json:"velocity_kmh,omitempty"`
	FromTime    *time.Time `json:"from_time,omitempty"`
	ToTime      *time.Time `json:"to_time,omitempty"`
	FromLat     *float64   `json:"from_lat,omitempty"`
	FromLon     *float64   `json:"from_lon,omitempty"`
	ToLat       *float64   `json:"to_lat,omitempty"`
	ToLon       *float64   `json:"to_lon,omitempty"`
}
