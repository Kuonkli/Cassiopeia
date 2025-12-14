package repository

import (
	"cassiopeia/internal/models"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OSDRRepository interface {
	Create(ctx context.Context, item *models.OSDRItem) error
	BulkUpsert(ctx context.Context, items []models.OSDRItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.OSDRItem, error)
	GetByDatasetID(ctx context.Context, datasetID string) (*models.OSDRItem, error)
	GetPaginated(ctx context.Context, page, limit int) ([]models.OSDRItem, error)
	Search(ctx context.Context, query string, limit int) ([]models.OSDRItem, error)
	Update(ctx context.Context, item *models.OSDRItem) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}

type osdrRepository struct {
	db *gorm.DB
}

func NewOSDRRepository(db *gorm.DB) OSDRRepository {
	return &osdrRepository{db: db}
}

func (r *osdrRepository) Create(ctx context.Context, item *models.OSDRItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *osdrRepository) BulkUpsert(ctx context.Context, items []models.OSDRItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if item.DatasetID == "" {
				continue
			}

			var existing models.OSDRItem
			err := tx.Where("dataset_id = ?", item.DatasetID).First(&existing).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Создаем новую запись
				if err := tx.Create(&item).Error; err != nil {
					return err
				}
			} else if err == nil {
				// Обновляем существующую
				item.ID = existing.ID
				item.CreatedAt = existing.CreatedAt
				if err := tx.Save(&item).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return nil
	})
}

func (r *osdrRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OSDRItem, error) {
	var item models.OSDRItem
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *osdrRepository) GetByDatasetID(ctx context.Context, datasetID string) (*models.OSDRItem, error) {
	var item models.OSDRItem
	err := r.db.WithContext(ctx).First(&item, "dataset_id = ?", datasetID).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *osdrRepository) GetPaginated(ctx context.Context, page, limit int) ([]models.OSDRItem, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var items []models.OSDRItem
	err := r.db.WithContext(ctx).
		Order("updated_at DESC NULLS LAST, created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&items).
		Error

	return items, err
}

func (r *osdrRepository) Search(ctx context.Context, query string, limit int) ([]models.OSDRItem, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var items []models.OSDRItem
	err := r.db.WithContext(ctx).
		Where("title ILIKE ? OR dataset_id ILIKE ?",
			"%"+query+"%", "%"+query+"%").
		Order("updated_at DESC").
		Limit(limit).
		Find(&items).
		Error

	return items, err
}

func (r *osdrRepository) Update(ctx context.Context, item *models.OSDRItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *osdrRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.OSDRItem{}, "id = ?", id).Error
}

func (r *osdrRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.OSDRItem{}).
		Count(&count).
		Error
	return count, err
}
