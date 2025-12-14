package handlers

import (
	"net/http"
	"strconv"

	"cassiopeia/internal/service"

	"github.com/gin-gonic/gin"
)

type OSDRHandler struct {
	service service.NASAService
}

func NewOSDRHandler(service service.NASAService) *OSDRHandler {
	return &OSDRHandler{service: service}
}

func (h *OSDRHandler) GetOSDRList(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	items, err := h.service.GetOSDRList(ctx, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get OSDR list",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"page":    page,
		"limit":   limit,
	})
}

func (h *OSDRHandler) GetAPOD(c *gin.Context) {
	ctx := c.Request.Context()

	date := c.Query("date")

	apod, err := h.service.GetAPOD(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get APOD",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    apod,
	})
}

func (h *OSDRHandler) GetNEO(c *gin.Context) {
	ctx := c.Request.Context()

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	neo, err := h.service.GetNEOWatch(ctx, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get NEO data",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    neo,
		"days":    days,
	})
}

func (h *OSDRHandler) GetDONKI(c *gin.Context) {
	ctx := c.Request.Context()

	eventType := c.DefaultQuery("type", "FLR")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "5"))

	events, err := h.service.GetDONKI(ctx, eventType, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get DONKI data",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    events,
		"type":    eventType,
		"days":    days,
	})
}

func (h *OSDRHandler) ForceSyncOSDR(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.service.FetchAndStoreOSDR(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to sync OSDR data",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OSDR data synchronized successfully",
	})
}
