package service

import (
	"database/sql"

	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/errors"
	"assign-reviewers-for-pull-requests/internal/model"
	"assign-reviewers-for-pull-requests/internal/repository"
)

type UserService interface {
	SetIsActive(userID string, isActive bool) (*model.User, error)
	GetReviews(userID string) ([]model.PullRequestShort, error)
}

type userService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

func NewUserService(repos *repository.Repositories, logger *zap.Logger) UserService {
	return &userService{
		repos:  repos,
		logger: logger,
	}
}

func (s *userService) SetIsActive(userID string, isActive bool) (*model.User, error) {
	user, err := s.repos.User.GetByUserID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("user")
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	if err := s.repos.User.SetIsActive(userID, isActive); err != nil {
		s.logger.Error("Failed to set user active status", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	if !isActive {
		prs, err := s.repos.PullRequest.GetPRsByReviewerUserID(userID)
		if err == nil {
			for _, pr := range prs {
				if pr.Status == "OPEN" {
					prObj, err := s.repos.PullRequest.GetByPRID(pr.PullRequestID)
					if err == nil {
						_ = s.repos.PullRequest.RemoveReviewer(prObj.ID, user.ID)
						s.logger.Info("Removed inactive reviewer from PR",
							zap.String("user_id", userID),
							zap.String("pr_id", pr.PullRequestID),
						)
					}
				}
			}
		}
	}

	user.IsActive = isActive

	s.logger.Info("User active status updated",
		zap.String("user_id", userID),
		zap.Bool("is_active", isActive),
	)

	return user, nil
}

func (s *userService) GetReviews(userID string) ([]model.PullRequestShort, error) {
	prs, err := s.repos.PullRequest.GetPRsByReviewerUserID(userID)
	if err != nil {
		s.logger.Error("Failed to get user reviews", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	return prs, nil
}