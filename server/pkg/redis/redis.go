package redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"strconv"
	"time"
)

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func Connect(config Config) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     100,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		IdleTimeout:  5 * time.Minute,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем подключение
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Получаем информацию о сервере
	info, err := client.Info(ctx, "server").Result()
	if err != nil {
		log.Printf("Failed to get Redis info: %v", err)
	} else {
		log.Printf("Redis connected: %s", addr)
		// Извлекаем версию Redis из информации
		if len(info) > 0 {
			log.Printf("Redis info: %s", info[:100])
		}
	}

	return client, nil
}

// GetStats возвращает статистику Redis
func GetStats(client *redis.Client) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	info, err := client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]string)

	// Парсим информацию
	targetMetrics := []string{
		"redis_version",
		"connected_clients",
		"used_memory_human",
		"used_memory_peak_human",
		"total_connections_received",
		"total_commands_processed",
		"keyspace_hits",
		"keyspace_misses",
		"uptime_in_seconds",
	}

	// Проходим по всем строкам информации
	for _, infoLine := range stringToLines(info) {
		if len(infoLine) > 0 && infoLine[0] != '#' {
			if key, value, found := parseInfoLine(infoLine); found {
				// Проверяем, нужна ли нам эта метрика
				for _, target := range targetMetrics {
					if key == target {
						stats[key] = value
						break
					}
				}
			}
		}
	}

	return stats, nil
}

// Utility functions
func stringToLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func parseInfoLine(line string) (key, value string, found bool) {
	for i, c := range line {
		if c == ':' {
			key = line[:i]
			value = line[i+1:]
			return key, value, true
		}
	}
	return "", "", false
}

// Helper functions
func ParseInt(s string) int {
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

func ParseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}
