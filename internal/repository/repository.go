package repository

import "github.com/jmoiron/sqlx"

type Repositories struct {
	Team        TeamRepository
	User        UserRepository
	PullRequest PullRequestRepository
	Stats       StatsRepository
}

func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		Team:        NewTeamRepository(db),
		User:        NewUserRepository(db),
		PullRequest: NewPullRequestRepository(db),
		Stats:       NewStatsRepository(db),
	}
}