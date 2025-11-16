package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/service"
	"assign-reviewers-for-pull-requests/internal/errors"
)

type Handler struct {
	services *service.Services
	logger   *zap.Logger
}

func NewHandler(services *service.Services, logger *zap.Logger) *Handler {
	return &Handler{
		services: services,
		logger:   logger,
	}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	router.GET("/health", h.healthCheck)

	router.POST("/team/add", h.createTeam)
	router.GET("/team/get", h.getTeam)

	router.POST("/users/setIsActive", h.setIsActive)
	router.GET("/users/getReview", h.getUserReviews)

	router.POST("/pullRequest/create", h.createPR)
	router.POST("/pullRequest/merge", h.mergePR)
	router.POST("/pullRequest/reassign", h.reassignReviewer)

	router.GET("/stats", h.getStats)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		c.JSON(appErr.HTTPStatus, errors.ErrorResponse{Error: *appErr})
	} else {
		appErr := errors.ErrInternal(err)
		c.JSON(appErr.HTTPStatus, errors.ErrorResponse{Error: *appErr})
	}
}
