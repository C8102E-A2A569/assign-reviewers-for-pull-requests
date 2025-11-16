package repository

import (
	"github.com/jmoiron/sqlx"
)

type StatsRepository interface {
	RecordAssignment(userInternalID, prInternalID string) error
	GetUserStats() ([]struct {
		UserID          string `db:"user_id"`
		AssignmentCount int    `db:"assignment_count"`
	}, error)
	GetPRStats() ([]struct {
		PRID          string `db:"pr_id"`
		ReviewerCount int    `db:"reviewer_count"`
	}, error)
}

type statsRepository struct {
	db *sqlx.DB
}

func NewStatsRepository(db *sqlx.DB) StatsRepository {
	return &statsRepository{db: db}
}

func (r *statsRepository) RecordAssignment(userInternalID, prInternalID string) error {
	query := `
		INSERT INTO assignment_stats (user_id, pr_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, pr_id) DO NOTHING
	`
	_, err := r.db.Exec(query, userInternalID, prInternalID)
	return err
}

func (r *statsRepository) GetUserStats() ([]struct {
	UserID          string `db:"user_id"`
	AssignmentCount int    `db:"assignment_count"`
}, error) {
	query := `
		SELECT u.user_id, COUNT(*) as assignment_count
		FROM assignment_stats s
		JOIN users u ON s.user_id = u.id
		GROUP BY u.user_id
		ORDER BY assignment_count DESC
	`
	var stats []struct {
		UserID          string `db:"user_id"`
		AssignmentCount int    `db:"assignment_count"`
	}
	err := r.db.Select(&stats, query)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []struct {
			UserID          string `db:"user_id"`
			AssignmentCount int    `db:"assignment_count"`
		}{}
	}
	return stats, nil
}

func (r *statsRepository) GetPRStats() ([]struct {
	PRID          string `db:"pr_id"`
	ReviewerCount int    `db:"reviewer_count"`
}, error) {
	query := `
		SELECT pr.pull_request_id as pr_id, COUNT(*) as reviewer_count
		FROM assignment_stats s
		JOIN pull_requests pr ON s.pr_id = pr.id
		GROUP BY pr.pull_request_id
		ORDER BY reviewer_count DESC
	`
	var stats []struct {
		PRID          string `db:"pr_id"`
		ReviewerCount int    `db:"reviewer_count"`
	}
	err := r.db.Select(&stats, query)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []struct {
			PRID          string `db:"pr_id"`
			ReviewerCount int    `db:"reviewer_count"`
		}{}
	}
	return stats, nil
}