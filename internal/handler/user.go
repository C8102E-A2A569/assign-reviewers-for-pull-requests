package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/model"
)

func (h *Handler) setIsActive(c *gin.Context) {
	var req model.SetIsActiveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	user, err := h.services.User.SetIsActive(req.UserID, req.IsActive)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *Handler) getUserReviews(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "user_id query parameter is required",
			},
		})
		return
	}

	prs, err := h.services.User.GetReviews(userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": prs,
	})
}