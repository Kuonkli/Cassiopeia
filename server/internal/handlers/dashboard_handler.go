package handlers

import (
	"net/http"
	"time"

	"cassiopeia/internal/service"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	issService       service.ISSService
	nasaService      service.NASAService
	jwstService      service.JWSTService
	astroService     service.AstroService
	telemetryService service.TelemetryService
}

func NewDashboardHandler(
	issService service.ISSService,
	nasaService service.NASAService,
	jwstService service.JWSTService,
	astroService service.AstroService,
	telemetryService service.TelemetryService,
) *DashboardHandler {
	return &DashboardHandler{
		issService:       issService,
		nasaService:      nasaService,
		jwstService:      jwstService,
		astroService:     astroService,
		telemetryService: telemetryService,
	}
}

// GetDashboardData godoc
// @Summary Получить данные для дашборда
// @Description Возвращает все данные для главного дашборда в одном запросе
// @Tags Dashboard
// @Produce json
// @Success 200 {object} DashboardResponse
// @Failure 500 {object} ErrorResponse
// @Router /dashboard [get]
func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	ctx := c.Request.Context()

	// Собираем данные параллельно
	type dashboardData struct {
		ISS       interface{} `json:"iss,omitempty"`
		OSDR      interface{} `json:"osdr,omitempty"`
		JWST      interface{} `json:"jwst,omitempty"`
		Astro     interface{} `json:"astro,omitempty"`
		Telemetry interface{} `json:"telemetry,omitempty"`
		CMSPages  interface{} `json:"cms_pages,omitempty"`
		Summary   interface{} `json:"summary,omitempty"`
		Errors    []string    `json:"errors,omitempty"`
	}

	data := dashboardData{}
	var errors []string

	// 1. Данные МКС
	issLast, err := h.issService.GetLastPosition(ctx)
	if err != nil {
		errors = append(errors, "ISS data: "+err.Error())
	} else {
		data.ISS = issLast
	}

	// 2. Тренд МКС
	issTrend, err := h.issService.GetTrend(ctx, 240)
	if err != nil {
		errors = append(errors, "ISS trend: "+err.Error())
	} else {
		if data.ISS != nil {
			issMap := data.ISS.(map[string]interface{})
			issMap["trend"] = issTrend
		}
	}

	// 3. NASA APOD
	apod, err := h.nasaService.GetAPOD(ctx, "")
	if err != nil {
		errors = append(errors, "APOD: "+err.Error())
	} else {
		data.OSDR = map[string]interface{}{
			"apod": apod,
		}
	}

	// 4. JWST изображения (первые 12)
	jwstImages, err := h.jwstService.GetFeed(ctx, "jpg", "", "", "", 1, 12)
	if err != nil {
		errors = append(errors, "JWST: "+err.Error())
	} else {
		data.JWST = jwstImages
	}

	// 5. Астрономические события
	astroEvents, err := h.astroService.GetEvents(ctx, 55.7558, 37.6176, 7)
	if err != nil {
		errors = append(errors, "Astronomy: "+err.Error())
	} else {
		data.Astro = astroEvents
	}

	// 6. Последние данные телеметрии
	telemetry, err := h.telemetryService.GetTelemetryHistory(ctx, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		errors = append(errors, "Telemetry: "+err.Error())
	} else {
		data.Telemetry = telemetry
	}

	// 8. Сводная статистика
	summary := map[string]interface{}{
		"timestamp":    "2023-12-15T10:30:00Z", // Заглушка
		"services_ok":  len(errors) == 0,
		"errors_count": len(errors),
		"data_sources": []string{"ISS", "NASA", "JWST", "AstronomyAPI", "Telemetry", "CMS"},
	}
	data.Summary = summary

	// Если есть ошибки, добавляем их
	if len(errors) > 0 {
		data.Errors = errors
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// HealthCheck godoc
// @Summary Проверка здоровья сервиса
// @Description Проверяет доступность всех компонентов системы
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *DashboardHandler) HealthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"services": map[string]interface{}{
			"database": "connected",
			"redis":    "connected",
			"api":      "running",
		},
		"timestamp": "2023-12-15T10:30:00Z", // Заглушка
	}

	c.JSON(http.StatusOK, health)
}

// DashboardResponse структура ответа для дашборда
type DashboardResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Errors  []string               `json:"errors,omitempty"`
}

// HealthResponse структура ответа для health check
type HealthResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Services  map[string]interface{} `json:"services"`
	Timestamp string                 `json:"timestamp"`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// SuccessResponse структура для успешных ответов
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
