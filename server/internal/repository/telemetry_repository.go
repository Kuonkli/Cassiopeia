package repository

import (
	"context"
	"time"

	"cassiopeia/internal/models"

	"gorm.io/gorm"
)

type TelemetryRepository interface {
	Create(ctx context.Context, telemetry *models.Telemetry) error
	BatchCreate(ctx context.Context, telemetries []models.Telemetry) error
	GetByDateRange(ctx context.Context, from, to time.Time) ([]models.Telemetry, error)
	GetLatest(ctx context.Context, limit int) ([]models.Telemetry, error)
	GetStats(ctx context.Context, from, to time.Time) (*TelemetryStats, error)
	DeleteOld(ctx context.Context, olderThan time.Time) error
}

type TelemetryStats struct {
	Count          int64   `json:"count"`
	AvgVoltage     float64 `json:"avg_voltage"`
	AvgTemperature float64 `json:"avg_temperature"`
	MinVoltage     float64 `json:"min_voltage"`
	MaxVoltage     float64 `json:"max_voltage"`
	MinTemperature float64 `json:"min_temperature"`
	MaxTemperature float64 `json:"max_temperature"`
}

type telemetryRepository struct {
	db *gorm.DB
}

func NewTelemetryRepository(db *gorm.DB) TelemetryRepository {
	return &telemetryRepository{db: db}
}

func (r *telemetryRepository) Create(ctx context.Context, telemetry *models.Telemetry) error {
	return r.db.WithContext(ctx).Create(telemetry).Error
}

func (r *telemetryRepository) BatchCreate(ctx context.Context, telemetries []models.Telemetry) error {
	return r.db.WithContext(ctx).CreateInBatches(telemetries, 100).Error
}

func (r *telemetryRepository) GetByDateRange(ctx context.Context, from, to time.Time) ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	err := r.db.WithContext(ctx).
		Where("recorded_at BETWEEN ? AND ?", from, to).
		Order("recorded_at DESC").
		Find(&telemetries).
		Error
	return telemetries, err
}

func (r *telemetryRepository) GetLatest(ctx context.Context, limit int) ([]models.Telemetry, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	var telemetries []models.Telemetry
	err := r.db.WithContext(ctx).
		Order("recorded_at DESC").
		Limit(limit).
		Find(&telemetries).
		Error
	return telemetries, err
}

func (r *telemetryRepository) GetStats(ctx context.Context, from, to time.Time) (*TelemetryStats, error) {
	var stats TelemetryStats

	// Получаем количество записей
	err := r.db.WithContext(ctx).
		Model(&models.Telemetry{}).
		Where("recorded_at BETWEEN ? AND ?", from, to).
		Count(&stats.Count).
		Error
	if err != nil {
		return nil, err
	}

	if stats.Count == 0 {
		return &stats, nil
	}

	// Получаем средние значения
	row := r.db.WithContext(ctx).
		Model(&models.Telemetry{}).
		Select("AVG(voltage) as avg_voltage, AVG(temperature) as avg_temperature, "+
			"MIN(voltage) as min_voltage, MAX(voltage) as max_voltage, "+
			"MIN(temperature) as min_temperature, MAX(temperature) as max_temperature").
		Where("recorded_at BETWEEN ? AND ?", from, to).
		Row()

	err = row.Scan(&stats.AvgVoltage, &stats.AvgTemperature,
		&stats.MinVoltage, &stats.MaxVoltage,
		&stats.MinTemperature, &stats.MaxTemperature)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *telemetryRepository) DeleteOld(ctx context.Context, olderThan time.Time) error {
	return r.db.WithContext(ctx).
		Where("recorded_at < ?", olderThan).
		Delete(&models.Telemetry{}).
		Error
}
