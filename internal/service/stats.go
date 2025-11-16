package service

import (
	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/repository"
)

type StatsService interface {
	GetUserStats() ([]struct {
		UserID          string `json:"user_id"`
		AssignmentCount int    `json:"assignment_count"`
	}, error)
	GetPRStats() ([]struct {
		PRID          string `json:"pr_id"`
		ReviewerCount int    `json:"reviewer_count"`
	}, error)
}

type statsService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

func NewStatsService(repos *repository.Repositories, logger *zap.Logger) StatsService {
	return &statsService{
		repos:  repos,
		logger: logger,
	}
}

func (s *statsService) GetUserStats() ([]struct {
	UserID          string `json:"user_id"`
	AssignmentCount int    `json:"assignment_count"`
}, error) {
	stats, err := s.repos.Stats.GetUserStats()
	if err != nil {
		s.logger.Error("Failed to get user stats", zap.Error(err))
		return nil, err
	}
	
	result := make([]struct {
		UserID          string `json:"user_id"`
		AssignmentCount int    `json:"assignment_count"`
	}, len(stats))
	
	for i, st := range stats {
		result[i].UserID = st.UserID
		result[i].AssignmentCount = st.AssignmentCount
	}
	
	return result, nil
}

func (s *statsService) GetPRStats() ([]struct {
	PRID          string `json:"pr_id"`
	ReviewerCount int    `json:"reviewer_count"`
}, error) {
	stats, err := s.repos.Stats.GetPRStats()
	if err != nil {
		s.logger.Error("Failed to get PR stats", zap.Error(err))
		return nil, err
	}
	
	result := make([]struct {
		PRID          string `json:"pr_id"`
		ReviewerCount int    `json:"reviewer_count"`
	}, len(stats))
	
	for i, st := range stats {
		result[i].PRID = st.PRID
		result[i].ReviewerCount = st.ReviewerCount
	}
	
	return result, nil
}