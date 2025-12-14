package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cassiopeia/internal/clients"
	"cassiopeia/internal/models"
	"cassiopeia/internal/repository"
)

type NASAService interface {
	FetchAndStoreOSDR(ctx context.Context) error
	FetchAndStoreAPOD(ctx context.Context) error
	FetchAndStoreNEO(ctx context.Context) error
	GetOSDRList(ctx context.Context, page, limit int) ([]models.OSDRItem, error)
	GetLatestAPOD(ctx context.Context) (map[string]interface{}, error)
	GetLatestNEO(ctx context.Context, days int) (map[string]interface{}, error)

	GetAPOD(ctx context.Context, date string) (map[string]interface{}, error)
	GetNEOWatch(ctx context.Context, days int) (map[string]interface{}, error)
	GetDONKI(ctx context.Context, eventType string, days int) ([]map[string]interface{}, error)
}

type nasaService struct {
	repo           repository.OSDRRepository
	spaceCacheRepo repository.SpaceCacheRepository
	cacheRepo      repository.CacheRepository
	client         clients.NASAClient
}

func NewNASAService(
	repo repository.OSDRRepository,
	spaceCacheRepo repository.SpaceCacheRepository,
	cacheRepo repository.CacheRepository,
	client clients.NASAClient,
) NASAService {
	return &nasaService{
		repo:           repo,
		spaceCacheRepo: spaceCacheRepo,
		cacheRepo:      cacheRepo,
		client:         client,
	}
}

func (s *nasaService) FetchAndStoreOSDR(ctx context.Context) error {
	cacheKey := "nasa:osdr:last_fetch"
	if cached, _ := s.cacheRepo.Get(ctx, cacheKey); cached != "" {
		return nil // Уже обновляли недавно
	}

	log.Println("Fetching NASA OSDR data...")

	items, err := s.client.FetchOSDR(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch OSDR data: %w", err)
	}

	// Сохраняем в БД
	var dbItems []models.OSDRItem
	for _, itemData := range items {
		payload, _ := json.Marshal(itemData)

		// Извлекаем данные из структуры
		datasetID := extractString(itemData, "dataset_id", "id", "uuid")
		title := extractString(itemData, "title", "name", "label")
		status := extractString(itemData, "status", "state", "lifecycle")

		item := models.OSDRItem{
			DatasetID: datasetID,
			Title:     title,
			Status:    status,
			UpdatedAt: extractTime(itemData),
			Raw:       payload,
		}
		dbItems = append(dbItems, item)
	}

	if len(dbItems) > 0 {
		if err := s.repo.BulkUpsert(ctx, dbItems); err != nil {
			return fmt.Errorf("failed to save OSDR data: %w", err)
		}
	}

	// Кэшируем
	s.cacheRepo.Set(ctx, cacheKey, "1", 10*time.Minute)
	log.Printf("OSDR data updated: %d items", len(dbItems))
	return nil
}

func (s *nasaService) FetchAndStoreAPOD(ctx context.Context) error {
	cacheKey := "nasa:apod:today"

	log.Println("Fetching NASA APOD...")

	apod, err := s.client.FetchAPOD(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to fetch APOD: %w", err)
	}

	// Кэшируем на 24 часа
	if err := s.cacheRepo.SetJSON(ctx, cacheKey, apod, 24*time.Hour); err != nil {
		log.Printf("Failed to cache APOD: %v", err)
		return err
	}

	log.Println("APOD data cached successfully")
	return nil
}

func (s *nasaService) FetchAndStoreNEO(ctx context.Context) error {
	cacheKey := "nasa:neo:last_week"

	log.Println("Fetching NEO data...")

	neoData, err := s.client.FetchNEOFeed(ctx, 7)
	if err != nil {
		return fmt.Errorf("failed to fetch NEO data: %w", err)
	}

	// Кэшируем на 2 часа
	if err := s.cacheRepo.SetJSON(ctx, cacheKey, neoData, 2*time.Hour); err != nil {
		log.Printf("Failed to cache NEO data: %v", err)
		return err
	}

	log.Println("NEO data cached successfully")
	return nil
}

func (s *nasaService) GetOSDRList(ctx context.Context, page, limit int) ([]models.OSDRItem, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	cacheKey := fmt.Sprintf("nasa:osdr:list:%d:%d", page, limit)

	// Пробуем получить из кэша
	var items []models.OSDRItem
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &items)
	if err == nil && len(items) > 0 {
		return items, nil
	}

	// Получаем из БД
	items, err = s.repo.GetPaginated(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSDR list: %w", err)
	}

	// Кэшируем на 5 минут
	if err := s.cacheRepo.SetJSON(ctx, cacheKey, items, 5*time.Minute); err != nil {
		log.Printf("Failed to cache OSDR list: %v", err)
	}

	return items, nil
}

func (s *nasaService) GetLatestAPOD(ctx context.Context) (map[string]interface{}, error) {
	cacheKey := "nasa:apod:today"

	var apodData map[string]interface{}
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &apodData)
	if err == nil && apodData != nil {
		return apodData, nil
	}

	// Если нет в кэше, фетчим свежие данные
	if err := s.FetchAndStoreAPOD(ctx); err != nil {
		return nil, err
	}

	// Пробуем снова
	err = s.cacheRepo.GetJSON(ctx, cacheKey, &apodData)
	if err != nil {
		return nil, fmt.Errorf("failed to get APOD data: %w", err)
	}

	return apodData, nil
}

func (s *nasaService) GetLatestNEO(ctx context.Context, days int) (map[string]interface{}, error) {
	if days < 1 || days > 30 {
		days = 7
	}

	cacheKey := fmt.Sprintf("nasa:neo:%dd", days)

	var neoData map[string]interface{}
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &neoData)
	if err == nil && neoData != nil {
		return neoData, nil
	}

	// Фетчим свежие данные
	if err := s.FetchAndStoreNEO(ctx); err != nil {
		return nil, err
	}

	// Пробуем снова
	err = s.cacheRepo.GetJSON(ctx, cacheKey, &neoData)
	if err != nil {
		return nil, fmt.Errorf("failed to get NEO data: %w", err)
	}

	return neoData, nil
}

// Helper functions
func extractString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

func extractTime(data map[string]interface{}) *time.Time {
	timeKeys := []string{"updated_at", "modified", "lastUpdated", "timestamp"}

	for _, key := range timeKeys {
		if val, ok := data[key]; ok {
			switch v := val.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					return &t
				}
			case float64:
				t := time.Unix(int64(v), 0)
				return &t
			}
		}
	}
	return nil
}

func (s *nasaService) GetAPOD(ctx context.Context, date string) (map[string]interface{}, error) {
	// Просто используем существующий метод
	return s.GetLatestAPOD(ctx)
}

func (s *nasaService) GetNEOWatch(ctx context.Context, days int) (map[string]interface{}, error) {
	// Используем существующий метод
	return s.GetLatestNEO(ctx, days)
}

func (s *nasaService) GetDONKI(ctx context.Context, eventType string, days int) ([]map[string]interface{}, error) {
	if days < 1 || days > 30 {
		days = 5
	}

	cacheKey := fmt.Sprintf("nasa:donki:%s:%dd", eventType, days)

	// Пробуем кэш
	var cachedEvents []map[string]interface{}
	if err := s.cacheRepo.GetJSON(ctx, cacheKey, &cachedEvents); err == nil && cachedEvents != nil {
		return cachedEvents, nil
	}

	// Получаем от API
	events, err := s.client.FetchDONKI(ctx, eventType, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DONKI data: %w", err)
	}

	// Кэшируем на 1 час
	s.cacheRepo.SetJSON(ctx, cacheKey, events, 1*time.Hour)

	return events, nil
}
