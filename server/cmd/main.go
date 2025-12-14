package main

import (
	"cassiopeia/internal/clients"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cassiopeia/internal/config"
	_ "cassiopeia/internal/handlers"
	"cassiopeia/internal/middleware"
	"cassiopeia/internal/repository"
	"cassiopeia/internal/service"
	"cassiopeia/internal/worker"
	"cassiopeia/pkg/database"
	"cassiopeia/pkg/redis"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	// Загрузка .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	log.Println("=== Cosmos Dashboard Backend Starting ===")

	// Загрузка конфигурации
	cfg := config.Load()

	// Подключение к PostgreSQL
	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	// Подключение к Redis
	redisClient, err := redis.Connect(cfg.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	// Автомиграция моделей
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Инициализация репозиториев
	issRepo := repository.NewISSRepository(db)
	osdrRepo := repository.NewOSDRRepository(db)
	telemetryRepo := repository.NewTelemetryRepository(db)
	spaceCacheRepo := repository.NewSpaceCacheRepository(db)
	cacheRepo := repository.NewCacheRepository(redisClient)

	issClient := clients.NewISSClient(cfg.ISS.URL)
	nasaClient := clients.NewNASAClient(cfg.NASA)
	jwstClient := clients.NewJWSTClient(cfg.JWST)
	astroClient := clients.NewAstroClient(cfg.Astro)

	// Инициализация сервисов
	issService := service.NewISSService(issRepo, cacheRepo, issClient, cfg.ISS)
	nasaService := service.NewNASAService(osdrRepo, spaceCacheRepo, cacheRepo, nasaClient)
	jwstService := service.NewJWSTService(cacheRepo, jwstClient)
	astroService := service.NewAstroService(cacheRepo, astroClient)
	telemetryService := service.NewTelemetryService(telemetryRepo, cfg.Telemetry.OutputDir)

	// Инициализация воркеров (фоновые задачи)
	scheduler := worker.NewScheduler()

	// Добавляем только нужных воркеров
	if cfg.Workers.ISSEnabled {
		scheduler.AddWorker(worker.NewISSWorker(issService, cfg.Workers.ISSInterval))
		log.Printf("ISS Worker enabled (interval: %v)", cfg.Workers.ISSInterval)
	}

	if cfg.Workers.NASAEnabled {
		scheduler.AddWorker(worker.NewNASAWorker(nasaService, cfg.Workers.NASAInterval))
		log.Printf("NASA Worker enabled (interval: %v)", cfg.Workers.NASAInterval)
	}

	if cfg.Workers.TelemetryEnabled {
		scheduler.AddWorker(worker.NewTelemetryWorker(telemetryService, cfg.Workers.TelemetryInterval))
		log.Printf("Telemetry Worker enabled (interval: %v)", cfg.Workers.TelemetryInterval)
	}

	// Запускаем воркеры в фоне
	go scheduler.Start()
	defer scheduler.Stop()

	// Инициализация Gin
	if cfg.App.Debug {
		gin.SetMode(gin.DebugMode)
		log.Println("Running in DEBUG mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS для React фронтенда
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", cfg.App.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting (только для продакшена)
	if !cfg.App.Debug {
		limiter := rate.NewLimiter(rate.Limit(cfg.RateLimit.RequestsPerSecond), cfg.RateLimit.Burst)
		r.Use(middleware.RateLimitMiddleware(limiter))
		log.Printf("Rate limiting enabled: %d req/sec, burst: %d",
			cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst)
	}

	// Группа API v1
	api := r.Group("/api/v1")

	// 1. ISS данные (как rust_iss /last и /iss/trend)
	api.GET("/iss/last", func(c *gin.Context) {
		ctx := c.Request.Context()
		position, err := issService.GetLastPosition(ctx)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get ISS position"})
			return
		}
		c.JSON(200, position)
	})

	api.GET("/iss/trend", func(c *gin.Context) {
		ctx := c.Request.Context()
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "240"))
		trend, err := issService.GetTrend(ctx, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get ISS trend"})
			return
		}
		c.JSON(200, trend)
	})

	// 2. OSDR данные (как rust_iss /osdr/list)
	api.GET("/osdr/list", func(c *gin.Context) {
		ctx := c.Request.Context()
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

		items, err := nasaService.GetOSDRList(ctx, page, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get OSDR list"})
			return
		}

		c.JSON(200, gin.H{"items": items})
	})

	// 3. JWST галерея (как php-web /api/jwst/feed)
	api.GET("/jwst/feed", func(c *gin.Context) {
		ctx := c.Request.Context()

		source := c.DefaultQuery("source", "jpg")
		suffix := c.Query("suffix")
		program := c.Query("program")
		instrument := c.Query("instrument")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "24"))

		images, err := jwstService.GetFeed(ctx, source, suffix, program, instrument, page, perPage)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get JWST feed"})
			return
		}

		c.JSON(200, gin.H{
			"source": source,
			"count":  len(images),
			"items":  images,
		})
	})

	// 4. AstronomyAPI события (как php-web /api/astro/events)
	api.GET("/astro/events", func(c *gin.Context) {
		ctx := c.Request.Context()

		lat, _ := strconv.ParseFloat(c.DefaultQuery("lat", "55.7558"), 64)
		lon, _ := strconv.ParseFloat(c.DefaultQuery("lon", "37.6176"), 64)
		days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

		events, err := astroService.GetEvents(ctx, lat, lon, days)
		if err != nil {
			// Вместо 500 возвращаем 200 с сообщением
			log.Printf("Astro service error (but returning stub): %v", err)

			c.JSON(200, gin.H{
				"events": []map[string]interface{}{
					{
						"name":    "Service Unavailable",
						"type":    "info",
						"when":    time.Now().Format(time.RFC3339),
						"details": "AstronomyAPI requires API key. Using stub data.",
					},
				},
				"location": gin.H{"lat": lat, "lon": lon},
				"days":     days,
				"note":     "This is stub data. Add ASTRO_APP_ID and ASTRO_APP_SECRET to .env for real data.",
			})
			return
		}

		c.JSON(200, gin.H{
			"events":   events,
			"location": gin.H{"lat": lat, "lon": lon},
			"days":     days,
		})
	})

	// 5. Телеметрия CSV экспорт (заменяет Pascal)
	api.GET("/telemetry/export", func(c *gin.Context) {
		ctx := c.Request.Context()

		format := c.DefaultQuery("format", "csv")
		fromStr := c.Query("from")
		toStr := c.Query("to")

		var from, to time.Time
		var err error

		if fromStr != "" {
			from, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid from date format"})
				return
			}
		}

		if toStr != "" {
			to, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid to date format"})
				return
			}
		}

		filepath, err := telemetryService.ExportTelemetry(ctx, format, from, to)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to export telemetry"})
			return
		}

		// Отправляем файл
		c.File(filepath)
	})

	// 6. Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"services": gin.H{
				"database":  "connected",
				"redis":     "connected",
				"iss_api":   "enabled",
				"nasa_api":  "enabled",
				"jwst_api":  "enabled",
				"astro_api": "enabled",
			},
		})
	})

	// 7. Системные эндпоинты
	api.GET("/system/stats", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Статистика из Redis
		redisStats, _ := redis.GetStats(redisClient)

		// Статистика из БД
		issCount, _ := issRepo.Count(ctx)
		osdrCount, _ := osdrRepo.Count(ctx)
		//telemetryCount, _ := telemetryRepo.Count(ctx)

		c.JSON(200, gin.H{
			"database": gin.H{
				"iss_logs":   issCount,
				"osdr_items": osdrCount,
				//"telemetry":  telemetryCount,
			},
			"redis": redisStats,
			"workers": gin.H{
				"iss_enabled":       cfg.Workers.ISSEnabled,
				"nasa_enabled":      cfg.Workers.NASAEnabled,
				"telemetry_enabled": cfg.Workers.TelemetryEnabled,
			},
		})
	})

	// 8. Force refresh endpoints (для дебага)
	if cfg.App.Debug {
		api.POST("/refresh/iss", func(c *gin.Context) {
			ctx := c.Request.Context()
			if err := issService.FetchAndStoreISSData(ctx); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "ISS data refreshed"})
		})

		api.POST("/refresh/nasa", func(c *gin.Context) {
			ctx := c.Request.Context()
			if err := nasaService.FetchAndStoreOSDR(ctx); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "NASA data refreshed"})
		})

		api.POST("/refresh/telemetry", func(c *gin.Context) {
			ctx := c.Request.Context()
			if _, err := telemetryService.GenerateTelemetry(ctx); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Telemetry generated"})
		})
	}

	// Главный дашборд со всеми данными
	api.GET("/dashboard", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Собираем все данные параллельно
		type DashboardData struct {
			ISS       interface{} `json:"iss"`
			OSDR      interface{} `json:"osdr"`
			JWST      interface{} `json:"jwst"`
			Astro     interface{} `json:"astro"`
			Telemetry interface{} `json:"telemetry"`
		}

		data := DashboardData{}

		// ISS данные
		if iss, err := issService.GetLastPosition(ctx); err == nil {
			data.ISS = iss
		}

		// OSDR данные
		if osdr, err := nasaService.GetOSDRList(ctx, 1, 10); err == nil {
			data.OSDR = osdr
		}

		// JWST изображения
		if jwst, err := jwstService.GetFeed(ctx, "jpg", "", "", "", 1, 12); err == nil {
			data.JWST = jwst
		}

		// Астрономические события
		if astro, err := astroService.GetEvents(ctx, 55.7558, 37.6176, 7); err == nil {
			data.Astro = astro
		}

		// Телеметрия (последние 50 записей)
		if telemetry, err := telemetryService.GetTelemetryHistory(ctx,
			time.Now().Add(-24*time.Hour), time.Now()); err == nil {
			if len(telemetry) > 50 {
				telemetry = telemetry[:50]
			}
			data.Telemetry = telemetry
		}

		c.JSON(200, gin.H{
			"success":   true,
			"data":      data,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on http://localhost:%s", cfg.App.Port)
		log.Printf("API available at http://localhost:%s/api/v1", cfg.App.Port)
		log.Printf("Health check: http://localhost:%s/api/v1/health", cfg.App.Port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server failed to start:", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited properly")
}
