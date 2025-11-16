package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/model"
)

func (h *Handler) createTeam(c *gin.Context) {
	var req model.Team

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

	team, err := h.services.Team.CreateTeam(&req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"team": team,
	})
}

func (h *Handler) getTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "team_name query parameter is required",
			},
		})
		return
	}

	team, err := h.services.Team.GetTeam(teamName)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, team)
}