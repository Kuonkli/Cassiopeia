package repository

import (
	"context"
	"time"

	"cassiopeia/internal/models"

	"gorm.io/gorm"
)

type ISSRepository interface {
	Create(ctx context.Context, log *models.ISSLog) error
	GetLast(ctx context.Context) (*models.ISSLog, error)
	GetLastN(ctx context.Context, n int) ([]*models.ISSLog, error)
	GetSince(ctx context.Context, since time.Time) ([]*models.ISSLog, error)
	Count(ctx context.Context) (int64, error)
}

type issRepository struct {
	db *gorm.DB
}

func NewISSRepository(db *gorm.DB) ISSRepository {
	return &issRepository{db: db}
}

func (r *issRepository) Create(ctx context.Context, log *models.ISSLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *issRepository) GetLast(ctx context.Context) (*models.ISSLog, error) {
	var log models.ISSLog
	err := r.db.WithContext(ctx).
		Order("fetched_at DESC").
		First(&log).
		Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *issRepository) GetLastN(ctx context.Context, n int) ([]*models.ISSLog, error) {
	var logs []*models.ISSLog
	err := r.db.WithContext(ctx).
		Order("fetched_at DESC").
		Limit(n).
		Find(&logs).
		Error
	return logs, err
}

func (r *issRepository) GetSince(ctx context.Context, since time.Time) ([]*models.ISSLog, error) {
	var logs []*models.ISSLog
	err := r.db.WithContext(ctx).
		Where("fetched_at >= ?", since).
		Order("fetched_at DESC").
		Find(&logs).
		Error
	return logs, err
}

func (r *issRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ISSLog{}).
		Count(&count).
		Error
	return count, err
}
