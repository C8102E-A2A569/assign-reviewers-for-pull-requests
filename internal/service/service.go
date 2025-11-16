package service

import (
	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/repository"
)

type Services struct {
	Team        TeamService
	User        UserService
	PullRequest PullRequestService
	Stats       StatsService
}

func NewServices(repos *repository.Repositories, logger *zap.Logger) *Services {
	return &Services{
		Team:        NewTeamService(repos, logger),
		User:        NewUserService(repos, logger),
		PullRequest: NewPullRequestService(repos, logger),
		Stats:       NewStatsService(repos, logger),
	}
}