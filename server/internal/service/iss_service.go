package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"cassiopeia/internal/clients"
	"cassiopeia/internal/models"
	"cassiopeia/internal/repository"
)

type ISSService interface {
	FetchAndStoreISSData(ctx context.Context) error
	GetLastPosition(ctx context.Context) (*models.ISSLog, error)
	GetTrend(ctx context.Context, limit int) (*models.ISSTrend, error)
	GetPositionsHistory(ctx context.Context, hours int) ([]*models.ISSLog, error)
}

type issService struct {
	repo      repository.ISSRepository
	cacheRepo repository.CacheRepository
	client    clients.ISSClient
	interval  time.Duration
}

type ISSConfig struct {
	URL      string
	Interval time.Duration
}

func NewISSService(
	repo repository.ISSRepository,
	cacheRepo repository.CacheRepository,
	client clients.ISSClient,
	config ISSConfig,
) ISSService {
	return &issService{
		repo:      repo,
		cacheRepo: cacheRepo,
		client:    client,
		interval:  config.Interval,
	}
}

func (s *issService) FetchAndStoreISSData(ctx context.Context) error {
	// Проверяем, не выполнялся ли запрос недавно
	cacheKey := "iss:last_fetch"
	if cached, err := s.cacheRepo.Get(ctx, cacheKey); err == nil && cached != "" {
		return nil // Уже обновляли недавно
	}

	log.Println("Fetching ISS data from external API...")

	data, err := s.client.GetCurrentPosition(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch ISS data: %w", err)
	}

	// Преобразуем в JSON для хранения
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal ISS data: %w", err)
	}

	// Сохраняем в БД
	issLog := &models.ISSLog{
		FetchedAt: time.Now().UTC(),
		SourceURL: "https://api.wheretheiss.at/v1/satellites/25544",
		Payload:   payload,
	}

	if err := s.repo.Create(ctx, issLog); err != nil {
		return fmt.Errorf("failed to save ISS data to DB: %w", err)
	}

	// Кэшируем последнюю позицию
	lastCacheKey := "iss:last_position"
	if err := s.cacheRepo.Set(ctx, lastCacheKey, string(payload), 2*time.Minute); err != nil {
		log.Printf("Failed to cache ISS data: %v", err)
	}

	// Устанавливаем блокировку на интервал
	if err := s.cacheRepo.Set(ctx, cacheKey, "1", s.interval); err != nil {
		log.Printf("Failed to set fetch lock: %v", err)
	}

	log.Printf("ISS data fetched and stored at %s", issLog.FetchedAt.Format(time.RFC3339))
	return nil
}

func (s *issService) GetLastPosition(ctx context.Context) (*models.ISSLog, error) {
	// Пробуем получить из кэша
	cacheKey := "iss:last_position"
	cached, err := s.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return &models.ISSLog{
				FetchedAt: time.Now().UTC(),
				SourceURL: "https://api.wheretheiss.at/v1/satellites/25544",
				Payload:   []byte(cached),
			}, nil
		}
	}

	// Если нет в кэше, берем из БД
	issLog, err := s.repo.GetLast(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get last ISS position: %w", err)
	}

	// Обновляем кэш
	if err := s.cacheRepo.Set(ctx, cacheKey, string(issLog.Payload), 2*time.Minute); err != nil {
		log.Printf("Failed to cache ISS data: %v", err)
	}

	return issLog, nil
}

func (s *issService) GetTrend(ctx context.Context, limit int) (*models.ISSTrend, error) {
	if limit <= 0 {
		limit = 240
	}

	cacheKey := fmt.Sprintf("iss:trend:%d", limit)

	// Пробуем получить из кэша
	var trend models.ISSTrend
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &trend)
	if err == nil {
		return &trend, nil
	}

	// Получаем последние позиции из БД
	positions, err := s.repo.GetLastN(ctx, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get ISS positions: %w", err)
	}

	if len(positions) < 2 {
		return &models.ISSTrend{
			Movement:    false,
			DeltaKm:     0,
			DtSec:       0,
			VelocityKmh: nil,
		}, nil
	}

	// Расчет тренда
	calculatedTrend := s.calculateTrend(positions[0], positions[1])

	// Кэшируем результат
	if err := s.cacheRepo.SetJSON(ctx, cacheKey, calculatedTrend, 30*time.Second); err != nil {
		log.Printf("Failed to cache ISS trend: %v", err)
	}

	return calculatedTrend, nil
}

func (s *issService) GetPositionsHistory(ctx context.Context, hours int) ([]*models.ISSLog, error) {
	if hours <= 0 {
		hours = 24
	}

	cacheKey := fmt.Sprintf("iss:history:%dh", hours)

	// Пробуем получить из кэша
	var positions []*models.ISSLog
	err := s.cacheRepo.GetJSON(ctx, cacheKey, &positions)
	if err == nil && len(positions) > 0 {
		return positions, nil
	}

	// Получаем из БД
	fromTime := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	positions, err = s.repo.GetSince(ctx, fromTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get ISS history: %w", err)
	}

	// Кэшируем
	if len(positions) > 0 {
		if err := s.cacheRepo.SetJSON(ctx, cacheKey, positions, 5*time.Minute); err != nil {
			log.Printf("Failed to cache ISS history: %v", err)
		}
	}

	return positions, nil
}

func (s *issService) calculateTrend(current, previous *models.ISSLog) *models.ISSTrend {
	var currentData, previousData map[string]interface{}

	if err := json.Unmarshal(current.Payload, &currentData); err != nil {
		return &models.ISSTrend{}
	}
	if err := json.Unmarshal(previous.Payload, &previousData); err != nil {
		return &models.ISSTrend{}
	}

	// Извлекаем координаты
	lat1 := extractFloat(previousData, "latitude")
	lon1 := extractFloat(previousData, "longitude")
	lat2 := extractFloat(currentData, "latitude")
	lon2 := extractFloat(currentData, "longitude")
	velocity := extractFloat(currentData, "velocity")

	// Расчет дистанции
	deltaKm := haversineDistance(lat1, lon1, lat2, lon2)
	dtSec := current.FetchedAt.Sub(previous.FetchedAt).Seconds()

	movement := deltaKm > 0.1
	var velocityKmh *float64
	if velocity > 0 {
		v := velocity * 3.6 // м/с → км/ч
		velocityKmh = &v
	}

	return &models.ISSTrend{
		Movement:    movement,
		DeltaKm:     deltaKm,
		DtSec:       dtSec,
		VelocityKmh: velocityKmh,
		FromTime:    &previous.FetchedAt,
		ToTime:      &current.FetchedAt,
		FromLat:     &lat1,
		FromLon:     &lon1,
		ToLat:       &lat2,
		ToLon:       &lon2,
	}
}

func extractFloat(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
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
	return 0
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km

	fi1 := lat1 * math.Pi / 180
	fi2 := lat2 * math.Pi / 180
	deltaFi := (lat2 - lat1) * math.Pi / 180
	deltaXi := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaFi/2)*math.Sin(deltaFi/2) + math.Cos(fi1)*math.Cos(fi2)*math.Sin(deltaXi/2)*math.Sin(deltaXi/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
