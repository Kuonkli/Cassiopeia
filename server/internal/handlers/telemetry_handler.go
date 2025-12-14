package handlers

import (
	"net/http"
	_ "path/filepath"
	"strconv"
	"time"

	"cassiopeia/internal/service"

	"github.com/gin-gonic/gin"
)

type TelemetryHandler struct {
	service service.TelemetryService
}

func NewTelemetryHandler(service service.TelemetryService) *TelemetryHandler {
	return &TelemetryHandler{service: service}
}

func (h *TelemetryHandler) GenerateTelemetry(c *gin.Context) {
	ctx := c.Request.Context()

	format := c.DefaultQuery("format", "csv")

	var filepath string
	var err error

	switch format {
	case "excel", "xlsx":
		filepath, err = h.service.GenerateTelemetryExcel(ctx)
	case "csv":
		filepath, err = h.service.GenerateTelemetryCSV(ctx)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported format, use 'csv' or 'excel'",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate telemetry",
			"message": err.Error(),
		})
		return
	}

	filename := filepath
	if filepath != "" {
		filename = filepath
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "telemetry generated successfully",
		"filepath": filename,
		"format":   format,
	})
}

func (h *TelemetryHandler) ExportTelemetry(c *gin.Context) {
	ctx := c.Request.Context()

	format := c.Query("format")
	if format == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "format parameter is required (csv, excel, json)",
		})
		return
	}

	// Парсим даты
	var from, to time.Time
	var err error

	if fromStr := c.Query("from"); fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid from date format, use YYYY-MM-DD",
			})
			return
		}
	}

	if toStr := c.Query("to"); toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid to date format, use YYYY-MM-DD",
			})
			return
		}
	}

	// Экспортируем данные
	filepath, err := h.service.ExportTelemetry(ctx, format, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to export telemetry",
			"message": err.Error(),
		})
		return
	}

	// Определяем Content-Type
	var contentType string
	var filename string

	switch format {
	case "csv":
		contentType = "text/csv"
		filename = "telemetry_export.csv"
	case "excel", "xlsx":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		filename = "telemetry_export.xlsx"
	case "json":
		contentType = "application/json"
		filename = "telemetry_export.json"
	default:
		contentType = "application/octet-stream"
		filename = "telemetry_export." + format
	}

	// Отправляем файл
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.File(filepath)
}

func (h *TelemetryHandler) GetTelemetryHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// Парсим даты
	var from, to time.Time
	var err error

	if fromStr := c.Query("from"); fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid from date format, use YYYY-MM-DD",
			})
			return
		}
	} else {
		from = time.Now().Add(-24 * time.Hour)
	}

	if toStr := c.Query("to"); toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid to date format, use YYYY-MM-DD",
			})
			return
		}
	} else {
		to = time.Now()
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	// Получаем данные
	history, err := h.service.GetTelemetryHistory(ctx, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get telemetry history",
			"message": err.Error(),
		})
		return
	}

	// Ограничиваем количество записей
	if len(history) > limit && limit > 0 {
		history = history[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"telemetry": history,
			"count":     len(history),
			"from":      from.Format("2006-01-02"),
			"to":        to.Format("2006-01-02"),
			"limit":     limit,
		},
	})
}
