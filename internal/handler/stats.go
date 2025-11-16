package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *Handler) getStats(c *gin.Context) {
	statsType := c.Query("type")

	if statsType == "" {
		// Return general stats
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
		return
	}

	if statsType == "users" {
		stats, err := h.services.Stats.GetUserStats()
		if err != nil {
			h.logger.Error("Failed to get user stats", zap.Error(err))
			h.respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"stats": stats,
		})
		return
	}

	if statsType == "prs" {
		stats, err := h.services.Stats.GetPRStats()
		if err != nil {
			h.logger.Error("Failed to get PR stats", zap.Error(err))
			h.respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"stats": stats,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"code":    "BAD_REQUEST",
			"message": "invalid stats type",
		},
	})
}
