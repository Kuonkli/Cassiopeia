package handlers

import (
	"net/http"
	"strconv"

	"cassiopeia/internal/service"

	"github.com/gin-gonic/gin"
)

type JWSTHandler struct {
	service service.JWSTService
}

func NewJWSTHandler(service service.JWSTService) *JWSTHandler {
	return &JWSTHandler{service: service}
}

func (h *JWSTHandler) GetJWSTFeed(c *gin.Context) {
	ctx := c.Request.Context()

	source := c.DefaultQuery("source", "jpg")
	suffix := c.Query("suffix")
	program := c.Query("program")
	instrument := c.Query("instrument")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "24"))

	images, err := h.service.GetFeed(ctx, source, suffix, program, instrument, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get JWST feed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"images":  images,
			"count":   len(images),
			"page":    page,
			"perPage": perPage,
			"source":  source,
		},
	})
}

func (h *JWSTHandler) GetJWSTObservation(c *gin.Context) {
	ctx := c.Request.Context()

	observationID := c.Param("id")
	if observationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "observation ID is required",
		})
		return
	}

	observation, err := h.service.GetObservation(ctx, observationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "observation not found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    observation,
	})
}

func (h *JWSTHandler) GetJWSTProgram(c *gin.Context) {
	ctx := c.Request.Context()

	programID := c.Param("program_id")
	if programID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "program ID is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "12"))

	images, err := h.service.GetProgramImages(ctx, programID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get program images",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"program": programID,
			"images":  images,
			"count":   len(images),
			"page":    page,
			"perPage": perPage,
		},
	})
}
