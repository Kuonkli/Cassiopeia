package service

import (
	"context"
	_ "encoding/base64"
	"fmt"
	"log"
	"time"

	"cassiopeia/internal/clients"
	"cassiopeia/internal/repository"
)

type AstroService interface {
	GetEvents(ctx context.Context, lat, lon float64, days int) ([]AstroEvent, error)
	GetBodies(ctx context.Context) (map[string]interface{}, error) // ДОБАВИТЬ
	GetMoonPhase(ctx context.Context, date time.Time) (map[string]interface{}, error)
}

type astroService struct {
	cacheRepo repository.CacheRepository
	client    clients.AstroClient
}

type AstroEvent struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	When      time.Time `json:"when"`
	Magnitude float64   `json:"magnitude,omitempty"`
	Altitude  float64   `json:"altitude,omitempty"`
	Details   string    `json:"details,omitempty"`
}

func NewAstroService(
	cacheRepo repository.CacheRepository,
	client clients.AstroClient,
) AstroService {
	return &astroService{
		cacheRepo: cacheRepo,
		client:    client,
	}
}

func (s *astroService) GetEvents(ctx context.Context, lat, lon float64, days int) ([]AstroEvent, error) {
	if days < 1 || days > 30 {
		days = 7
	}

	// Генерируем ключ кэша
	cacheKey := fmt.Sprintf("astro:events:%.4f:%.4f:%d", lat, lon, days)

	// Пробуем получить из кэша
	var cachedEvents []AstroEvent
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &cachedEvents)
	if err == nil && len(cachedEvents) > 0 {
		return cachedEvents, nil
	}

	log.Printf("Fetching astronomy events for lat=%.4f, lon=%.4f, days=%d", lat, lon, days)

	// Получаем данные от API
	rawData, err := s.client.GetEvents(ctx, lat, lon, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch astronomy events: %w", err)
	}

	// Парсим данные
	events := s.parseEvents(rawData)

	// Кэшируем на 6 часов
	if err := s.cacheRepo.SetJSON(ctx, cacheKey, events, 6*time.Hour); err != nil {
		log.Printf("Failed to cache astronomy events: %v", err)
	}

	return events, nil
}

func (s *astroService) parseEvents(rawData map[string]interface{}) []AstroEvent {
	var events []AstroEvent

	// Рекурсивно ищем события в JSON
	s.traverseJSON(rawData, &events)
	return events
}

func (s *astroService) traverseJSON(data interface{}, events *[]AstroEvent) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Проверяем, является ли этот объект событием
		if s.isEventObject(v) {
			event := s.normalizeEvent(v)
			if event.Name != "" && event.Type != "" {
				*events = append(*events, event)
			}
		}

		// Рекурсивно обходим все поля
		for _, value := range v {
			s.traverseJSON(value, events)
		}

	case []interface{}:
		// Обходим массив
		for _, item := range v {
			s.traverseJSON(item, events)
		}
	}
}

func (s *astroService) isEventObject(obj map[string]interface{}) bool {
	// Проверяем наличие полей, характерных для астрономических событий
	hasType := false
	hasName := false
	hasTime := false

	for key, val := range obj {
		switch key {
		case "type", "event_type", "category", "kind":
			if val != nil {
				hasType = true
			}
		case "name", "body", "object", "target":
			if val != nil {
				hasName = true
			}
		case "time", "date", "occursAt", "peak", "instant":
			if val != nil {
				hasTime = true
			}
		}
	}

	return hasType && hasName && hasTime
}

func (s *astroService) normalizeEvent(obj map[string]interface{}) AstroEvent {
	event := AstroEvent{}

	// Извлекаем имя
	event.Name = s.extractString(obj, "name", "body", "object", "target")

	// Извлекаем тип
	event.Type = s.extractString(obj, "type", "event_type", "category", "kind")

	// Извлекаем время
	event.When = s.extractTime(obj)

	// Извлекаем дополнительные данные
	event.Magnitude = s.extractFloat(obj, "magnitude", "mag")
	event.Altitude = s.extractFloat(obj, "altitude")
	event.Details = s.extractString(obj, "note", "description")

	return event
}

func (s *astroService) extractString(obj map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := obj[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

func (s *astroService) extractFloat(obj map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if val, ok := obj[key]; ok {
			switch v := val.(type) {
			case float64:
				return v
			case float32:
				return float64(v)
			case int:
				return float64(v)
			case string:
				var f float64
				if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func (s *astroService) extractTime(obj map[string]interface{}) time.Time {
	timeKeys := []string{"time", "date", "occursAt", "peak", "instant"}

	for _, key := range timeKeys {
		if val, ok := obj[key]; ok {
			switch v := val.(type) {
			case string:
				// Пробуем разные форматы времени
				formats := []string{
					time.RFC3339,
					"2006-01-02T15:04:05Z",
					"2006-01-02 15:04:05",
					"2006-01-02",
				}

				for _, format := range formats {
					if t, err := time.Parse(format, v); err == nil {
						return t
					}
				}
			case float64:
				return time.Unix(int64(v), 0)
			}
		}
	}

	return time.Now()
}

func (s *astroService) GetBodies(ctx context.Context) (map[string]interface{}, error) {
	cacheKey := "astro:bodies"

	// Пробуем кэш
	var cachedBodies map[string]interface{}
	if err := s.cacheRepo.GetJSON(ctx, cacheKey, &cachedBodies); err == nil && cachedBodies != nil {
		return cachedBodies, nil
	}

	// Получаем от API
	bodies, err := s.client.GetBodies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch celestial bodies: %w", err)
	}

	// Кэшируем на 24 часа
	s.cacheRepo.SetJSON(ctx, cacheKey, bodies, 24*time.Hour)

	return bodies, nil
}

func (s *astroService) GetMoonPhase(ctx context.Context, date time.Time) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("astro:moon_phase:%s", date.Format("2006-01-02"))

	// Пробуем кэш
	var cachedPhase map[string]interface{}
	if err := s.cacheRepo.GetJSON(ctx, cacheKey, &cachedPhase); err == nil && cachedPhase != nil {
		return cachedPhase, nil
	}

	// Получаем от API
	phase, err := s.client.GetMoonPhase(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch moon phase: %w", err)
	}

	// Кэшируем на 24 часа
	s.cacheRepo.SetJSON(ctx, cacheKey, phase, 24*time.Hour)

	return phase, nil
}
