package database

import (
	"fmt"
	"log"
	"time"

	"cassiopeia/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Connect(config Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Настройка пула соединений
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connected successfully")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	// Включаем расширения
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	// Автомиграция моделей
	err := db.AutoMigrate(
		&models.ISSLog{},
		&models.OSDRItem{},
		&models.Telemetry{},
		&models.SpaceCache{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate models: %w", err)
	}

	// Создаем индексы
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

func createIndexes(db *gorm.DB) error {
	// Индексы для ISSLog
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_iss_log_fetched_at ON iss_logs(fetched_at DESC)").Error; err != nil {
		return err
	}

	// Индексы для OSDRItem
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_osdr_item_updated_at ON osdr_items(updated_at DESC NULLS LAST)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_osdr_item_title ON osdr_items USING gin(title gin_trgm_ops)").Error; err != nil {
		return err
	}

	// Индексы для Telemetry
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_telemetry_recorded_at ON telemetries(recorded_at DESC)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_telemetry_created_at ON telemetries(created_at DESC)").Error; err != nil {
		return err
	}

	// Индексы для SpaceCache
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_space_cache_source_fetched ON space_caches(source, fetched_at DESC)").Error; err != nil {
		return err
	}

	return nil
}
