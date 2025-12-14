package handlers

import (
	"net/http"
	"strconv"
	"time"

	"cassiopeia/internal/service"

	"github.com/gin-gonic/gin"
)

type AstroHandler struct {
	service service.AstroService
}

func NewAstroHandler(service service.AstroService) *AstroHandler {
	return &AstroHandler{service: service}
}

func (h *AstroHandler) GetAstroEvents(c *gin.Context) {
	ctx := c.Request.Context()

	lat, _ := strconv.ParseFloat(c.DefaultQuery("lat", "55.7558"), 64)
	lon, _ := strconv.ParseFloat(c.DefaultQuery("lon", "37.6176"), 64)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	events, err := h.service.GetEvents(ctx, lat, lon, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get astronomy events",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"events": events,
			"count":  len(events),
			"location": gin.H{
				"lat": lat,
				"lon": lon,
			},
			"days": days,
		},
	})
}

func (h *AstroHandler) GetCelestialBodies(c *gin.Context) {
	ctx := c.Request.Context()

	bodies, err := h.service.GetBodies(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get celestial bodies",
			"message": err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bodies,
		"count":   len(bodies),
	})
}

func (h *AstroHandler) GetMoonPhase(c *gin.Context) {
	ctx := c.Request.Context()

	dateStr := c.Query("date")
	var date time.Time
	var err error

	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid date format, use YYYY-MM-DD",
			})
			return
		}
	} else {
		date = time.Now()
	}

	phase, err := h.service.GetMoonPhase(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get moon phase",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    phase,
		"date":    date.Format("2006-01-02"),
	})
}
