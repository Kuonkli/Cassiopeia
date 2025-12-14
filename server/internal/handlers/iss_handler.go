package handlers

import (
	"net/http"
	"strconv"

	"cassiopeia/internal/service"

	"github.com/gin-gonic/gin"
)

type ISSHandler struct {
	service service.ISSService
}

func NewISSHandler(service service.ISSService) *ISSHandler {
	return &ISSHandler{service: service}
}

func (h *ISSHandler) GetLastISS(c *gin.Context) {
	ctx := c.Request.Context()

	position, err := h.service.GetLastPosition(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get ISS position",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, position)
}

func (h *ISSHandler) GetISSTrend(c *gin.Context) {
	ctx := c.Request.Context()

	limit := 240 // значение по умолчанию
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	trend, err := h.service.GetTrend(ctx, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get ISS trend",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, trend)
}

func (h *ISSHandler) GetISSHistory(c *gin.Context) {
	ctx := c.Request.Context()

	hours := 24 // по умолчанию последние 24 часа
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
			hours = h
		}
	}

	history, err := h.service.GetPositionsHistory(ctx, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get ISS history",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (h *ISSHandler) ForceFetchISS(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.service.FetchAndStoreISSData(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to fetch ISS data",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ISS data fetched successfully",
	})
}
