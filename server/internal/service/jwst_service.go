package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cassiopeia/internal/clients"
	"cassiopeia/internal/repository"
)

type JWSTService interface {
	GetFeed(ctx context.Context, source, suffix, program, instrument string, page, perPage int) ([]JWSTImage, error)
	GetObservation(ctx context.Context, observationID string) (map[string]interface{}, error)
	GetProgramImages(ctx context.Context, programID string, page, perPage int) ([]JWSTImage, error)
}

type jwstService struct {
	cacheRepo repository.CacheRepository
	client    clients.JWSTClient
}

type JWSTImage struct {
	URL         string   `json:"url"`
	ObsID       string   `json:"obs"`
	Program     string   `json:"program"`
	Suffix      string   `json:"suffix"`
	Instruments []string `json:"inst"`
	Caption     string   `json:"caption"`
	Link        string   `json:"link"`
}

func NewJWSTService(
	cacheRepo repository.CacheRepository,
	client clients.JWSTClient,
) JWSTService {
	return &jwstService{
		cacheRepo: cacheRepo,
		client:    client,
	}
}

func (s *jwstService) GetObservation(ctx context.Context, observationID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("jwst:observation:%s", observationID)

	// Пробуем кэш
	var cachedData map[string]interface{}
	if err := s.cacheRepo.GetJSON(ctx, cacheKey, &cachedData); err == nil && cachedData != nil {
		return cachedData, nil
	}

	// Получаем от API
	data, err := s.client.Get(ctx, fmt.Sprintf("observation/%s", observationID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch observation: %w", err)
	}

	// Кэшируем на 6 часов
	s.cacheRepo.SetJSON(ctx, cacheKey, data, 6*time.Hour)

	return data, nil
}

func (s *jwstService) GetProgramImages(ctx context.Context, programID string, page, perPage int) ([]JWSTImage, error) {
	// Используем существующий GetFeed с параметром program
	return s.GetFeed(ctx, "program", "", programID, "", page, perPage)
}

func (s *jwstService) GetFeed(ctx context.Context, source, suffix, program, instrument string, page, perPage int) ([]JWSTImage, error) {
	// Генерируем ключ кэша
	cacheKey := fmt.Sprintf("jwst:feed:%s:%s:%s:%s:%d:%d",
		source, suffix, program, instrument, page, perPage)

	// Пробуем получить из кэша
	var cachedImages []JWSTImage
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &cachedImages)
	if err == nil && len(cachedImages) > 0 {
		log.Printf("JWST feed served from cache: %s", cacheKey)
		return cachedImages, nil
	}

	// Определяем путь API
	path := "all/type/jpg"
	switch source {
	case "suffix":
		if suffix != "" {
			path = fmt.Sprintf("all/suffix/%s", strings.TrimPrefix(suffix, "/"))
		}
	case "program":
		if program != "" {
			path = fmt.Sprintf("program/id/%s", program)
		}
	}

	// Получаем данные от API
	data, err := s.client.Get(ctx, path, map[string]string{
		"page":    fmt.Sprintf("%d", page),
		"perPage": fmt.Sprintf("%d", perPage),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWST data: %w", err)
	}

	// Обрабатываем данные
	images := s.processJWSTData(data, instrument)

	// Кэшируем на 15 минут
	if err := s.cacheRepo.SetJSON(ctx, cacheKey, images, 15*time.Minute); err != nil {
		log.Printf("Failed to cache JWST feed: %v", err)
	}

	return images, nil
}

func (s *jwstService) processJWSTData(data map[string]interface{}, instrumentFilter string) []JWSTImage {
	var images []JWSTImage

	// Извлекаем список элементов
	items := s.extractItems(data)

	for _, item := range items {
		// Получаем URL изображения
		imageURL := s.extractImageURL(item)
		if imageURL == "" {
			continue
		}

		// Получаем инструменты
		instruments := s.extractInstruments(item)

		// Фильтруем по инструменту если задан фильтр
		if instrumentFilter != "" && len(instruments) > 0 {
			if !containsInstrument(instruments, instrumentFilter) {
				continue
			}
		}

		// Создаем структуру изображения
		image := JWSTImage{
			URL:         imageURL,
			ObsID:       s.extractString(item, "observation_id", "observationId", "id"),
			Program:     s.extractString(item, "program"),
			Suffix:      s.extractSuffix(item),
			Instruments: instruments,
			Caption:     s.generateCaption(item, instruments),
			Link:        s.extractString(item, "location", "url", "href"),
		}

		if image.Link == "" {
			image.Link = imageURL
		}

		images = append(images, image)
	}

	return images
}

func (s *jwstService) extractItems(data map[string]interface{}) []map[string]interface{} {
	var items []map[string]interface{}

	// Проверяем различные возможные структуры
	if body, ok := data["body"].([]interface{}); ok {
		for _, item := range body {
			if itemMap, ok := item.(map[string]interface{}); ok {
				items = append(items, itemMap)
			}
		}
	} else if dataItems, ok := data["data"].([]interface{}); ok {
		for _, item := range dataItems {
			if itemMap, ok := item.(map[string]interface{}); ok {
				items = append(items, itemMap)
			}
		}
	}

	return items
}

func (s *jwstService) extractImageURL(item map[string]interface{}) string {
	// Проверяем различные поля с URL
	urlKeys := []string{"thumbnail", "thumbnailUrl", "image", "img", "url", "href", "s3_url", "file_url"}

	for _, key := range urlKeys {
		if val, ok := item[key]; ok {
			if url, ok := val.(string); ok {
				// Проверяем, что это изображение
				urlLower := strings.ToLower(url)
				if strings.HasSuffix(urlLower, ".jpg") ||
					strings.HasSuffix(urlLower, ".jpeg") ||
					strings.HasSuffix(urlLower, ".png") {
					return url
				}
			}
		}
	}

	return ""
}

func (s *jwstService) extractInstruments(item map[string]interface{}) []string {
	var instruments []string

	// Проверяем поле details.instruments
	if details, ok := item["details"].(map[string]interface{}); ok {
		if instList, ok := details["instruments"].([]interface{}); ok {
			for _, inst := range instList {
				if instMap, ok := inst.(map[string]interface{}); ok {
					if instrument, ok := instMap["instrument"].(string); ok && instrument != "" {
						instruments = append(instruments, strings.ToUpper(instrument))
					}
				} else if instrument, ok := inst.(string); ok && instrument != "" {
					instruments = append(instruments, strings.ToUpper(instrument))
				}
			}
		}
	}

	return instruments
}

func (s *jwstService) extractString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := item[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

func (s *jwstService) extractSuffix(item map[string]interface{}) string {
	if details, ok := item["details"].(map[string]interface{}); ok {
		if suffix, ok := details["suffix"].(string); ok {
			return suffix
		}
	}
	return ""
}

func (s *jwstService) generateCaption(item map[string]interface{}, instruments []string) string {
	var parts []string

	// ID наблюдения
	obsID := s.extractString(item, "observation_id", "observationId", "id")
	if obsID != "" {
		parts = append(parts, obsID)
	}

	// Программа
	program := s.extractString(item, "program")
	if program != "" {
		parts = append(parts, "P"+program)
	}

	// Суффикс
	suffix := s.extractSuffix(item)
	if suffix != "" {
		parts = append(parts, suffix)
	}

	// Инструменты
	if len(instruments) > 0 {
		parts = append(parts, strings.Join(instruments, "/"))
	}

	return strings.Join(parts, " · ")
}

func containsInstrument(instruments []string, target string) bool {
	targetUpper := strings.ToUpper(target)
	for _, inst := range instruments {
		if inst == targetUpper {
			return true
		}
	}
	return false
}
