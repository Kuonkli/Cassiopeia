package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	App struct {
		Port        string
		Debug       bool
		FrontendURL string
	}
	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
	}
	Redis struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
	ISS struct {
		URL      string
		Interval time.Duration
	}
	NASA struct {
		APIKey   string
		OSDRURL  string
		APODURL  string
		NEOURL   string
		DONKIURL string
	}
	JWST struct {
		Host   string
		APIKey string
		Email  string
	}
	Astro struct {
		AppID   string
		Secret  string
		BaseURL string
	}
	Workers struct {
		ISSEnabled        bool
		NASAEnabled       bool
		TelemetryEnabled  bool
		ISSInterval       time.Duration
		NASAInterval      time.Duration
		TelemetryInterval time.Duration
	}
	RateLimit struct {
		RequestsPerSecond int
		Burst             int
	}
	Telemetry struct {
		OutputDir string
	}
}

func Load() *Config {
	cfg := &Config{}

	cfg.Telemetry.OutputDir = getEnv("TELEMETRY_OUTPUT_DIR", "./data/telemetry")

	// App
	cfg.App.Port = getEnv("PORT", "8080")
	cfg.App.Debug = getEnvAsBool("DEBUG", false)
	cfg.App.FrontendURL = getEnv("FRONTEND_URL", "http://localhost:3000")

	// DB
	cfg.DB.Host = getEnv("DB_HOST", "localhost")
	cfg.DB.Port = getEnv("DB_PORT", "5432")
	cfg.DB.User = getEnv("DB_USER", "postgres")
	cfg.DB.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.DB.DBName = getEnv("DB_NAME", "cosmos")
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", "disable")

	// Redis
	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnv("REDIS_PORT", "6379")
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)

	// ISS
	cfg.ISS.URL = getEnv("ISS_URL", "https://api.wheretheiss.at/v1/satellites/25544")
	cfg.ISS.Interval = getEnvAsDuration("ISS_INTERVAL", 120*time.Second)

	// NASA
	cfg.NASA.APIKey = getEnv("NASA_API_KEY", "")
	cfg.NASA.OSDRURL = getEnv("NASA_OSDR_URL", "https://osdr.nasa.gov/osdr/data/osd/files/87.1")
	cfg.NASA.APODURL = getEnv("NASA_APOD_URL", "https://api.nasa.gov/planetary/apod")
	cfg.NASA.NEOURL = getEnv("NASA_NEO_URL", "https://api.nasa.gov/neo/rest/v1/feed")
	cfg.NASA.DONKIURL = getEnv("NASA_DONKI_URL", "https://api.nasa.gov/DONKI")

	// JWST
	cfg.JWST.Host = getEnv("JWST_HOST", "https://api.jwstapi.com")
	cfg.JWST.APIKey = getEnv("JWST_API_KEY", "")
	cfg.JWST.Email = getEnv("JWST_EMAIL", "")

	// Astro
	cfg.Astro.AppID = getEnv("ASTRO_APP_ID", "")
	cfg.Astro.Secret = getEnv("ASTRO_APP_SECRET", "")
	cfg.Astro.BaseURL = getEnv("ASTRO_BASE_URL", "https://api.astronomyapi.com/api/v2")

	// Workers
	cfg.Workers.ISSEnabled = getEnvAsBool("ISS_ENABLED", true)
	cfg.Workers.NASAEnabled = getEnvAsBool("NASA_ENABLED", true)
	cfg.Workers.TelemetryEnabled = getEnvAsBool("TELEMETRY_ENABLED", true)
	cfg.Workers.ISSInterval = getEnvAsDuration("WORKER_ISS_INTERVAL", 120*time.Second)
	cfg.Workers.NASAInterval = getEnvAsDuration("WORKER_NASA_INTERVAL", 3600*time.Second)
	cfg.Workers.TelemetryInterval = getEnvAsDuration("WORKER_TELEMETRY_INTERVAL", 300*time.Second)

	// Rate Limit
	cfg.RateLimit.RequestsPerSecond = getEnvAsInt("RATE_LIMIT_RPS", 10)
	cfg.RateLimit.Burst = getEnvAsInt("RATE_LIMIT_BURST", 20)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if dur, err := time.ParseDuration(value); err == nil {
			return dur
		}
	}
	return defaultValue
}
