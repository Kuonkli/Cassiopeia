package repository

import (
	"context"
	"time"

	"cassiopeia/internal/models"

	"gorm.io/gorm"
)

type SpaceCacheRepository interface {
	Create(ctx context.Context, cache *models.SpaceCache) error
	GetLatest(ctx context.Context, source string) (*models.SpaceCache, error)
	GetBySource(ctx context.Context, source string, limit int) ([]models.SpaceCache, error)
	GetByDateRange(ctx context.Context, source string, from, to time.Time) ([]models.SpaceCache, error)
	DeleteOld(ctx context.Context, olderThan time.Time) error
	DeleteBySource(ctx context.Context, source string) error
}

type spaceCacheRepository struct {
	db *gorm.DB
}

func NewSpaceCacheRepository(db *gorm.DB) SpaceCacheRepository {
	return &spaceCacheRepository{db: db}
}

func (r *spaceCacheRepository) Create(ctx context.Context, cache *models.SpaceCache) error {
	return r.db.WithContext(ctx).Create(cache).Error
}

func (r *spaceCacheRepository) GetLatest(ctx context.Context, source string) (*models.SpaceCache, error) {
	var cache models.SpaceCache
	err := r.db.WithContext(ctx).
		Where("source = ?", source).
		Order("fetched_at DESC").
		First(&cache).
		Error
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

func (r *spaceCacheRepository) GetBySource(ctx context.Context, source string, limit int) ([]models.SpaceCache, error) {
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var caches []models.SpaceCache
	err := r.db.WithContext(ctx).
		Where("source = ?", source).
		Order("fetched_at DESC").
		Limit(limit).
		Find(&caches).
		Error
	return caches, err
}

func (r *spaceCacheRepository) GetByDateRange(ctx context.Context, source string, from, to time.Time) ([]models.SpaceCache, error) {
	var caches []models.SpaceCache
	err := r.db.WithContext(ctx).
		Where("source = ? AND fetched_at BETWEEN ? AND ?", source, from, to).
		Order("fetched_at DESC").
		Find(&caches).
		Error
	return caches, err
}

func (r *spaceCacheRepository) DeleteOld(ctx context.Context, olderThan time.Time) error {
	return r.db.WithContext(ctx).
		Where("fetched_at < ?", olderThan).
		Delete(&models.SpaceCache{}).
		Error
}

func (r *spaceCacheRepository) DeleteBySource(ctx context.Context, source string) error {
	return r.db.WithContext(ctx).
		Where("source = ?", source).
		Delete(&models.SpaceCache{}).
		Error
}
