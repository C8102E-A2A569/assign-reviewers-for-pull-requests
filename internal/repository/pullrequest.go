package repository

import (
	"database/sql"
	"time"
	"github.com/jmoiron/sqlx"
	"assign-reviewers-for-pull-requests/internal/model"
)

type PullRequestRepository interface {
	Create(prID, prName, authorID string) (string, error)
	GetByPRID(prID string) (*model.PullRequest, error)
	Exists(prID string) (bool, error)
	UpdateStatus(id, status string, mergedAt *time.Time) error
	
	AssignReviewer(prInternalID, userInternalID string) error
	RemoveReviewer(prInternalID, userInternalID string) error
	GetReviewerUserIDs(prInternalID string) ([]string, error)
	GetPRsByReviewerUserID(userID string) ([]model.PullRequestShort, error)
	IsReviewerAssigned(prInternalID, userInternalID string) (bool, error)
	
	AssignReviewersBatch(prInternalID string, userInternalIDs []string) error
	RemoveAllReviewers(prInternalID string) error
	
	GetReviewerAssignmentCount(userInternalID string) (int, error)
}

type pullRequestRepository struct {
	db *sqlx.DB
}

func NewPullRequestRepository(db *sqlx.DB) PullRequestRepository {
	return &pullRequestRepository{db: db}
}

func (r *pullRequestRepository) Create(prID, prName, authorID string) (string, error) {
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, 'OPEN', NOW())
		RETURNING id
	`
	var id string
	err := r.db.Get(&id, query, prID, prName, authorID)
	return id, err
}

func (r *pullRequestRepository) GetByPRID(prID string) (*model.PullRequest, error) {
	query := `
		SELECT pr.id, pr.pull_request_id, pr.pull_request_name, 
		       u.user_id as author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN users u ON pr.author_id = u.id
		WHERE pr.pull_request_id = $1
	`
	var prRow struct {
		ID                string       `db:"id"`
		PullRequestID     string       `db:"pull_request_id"`
		PullRequestName   string       `db:"pull_request_name"`
		AuthorID          string       `db:"author_id"`
		Status            string       `db:"status"`
		CreatedAt         time.Time    `db:"created_at"`
		MergedAt          sql.NullTime `db:"merged_at"`
	}
	err := r.db.Get(&prRow, query, prID)
	if err != nil {
		return nil, err
	}
 
	reviewers, err := r.GetReviewerUserIDs(prRow.ID)
	if err != nil {
		return nil, err
	}

	pr := &model.PullRequest{
		ID:                prRow.ID,
		PullRequestID:     prRow.PullRequestID,
		PullRequestName:   prRow.PullRequestName,
		AuthorID:          prRow.AuthorID,
		Status:            prRow.Status,
		CreatedAt:         prRow.CreatedAt,
		MergedAt:          prRow.MergedAt,
		AssignedReviewers: reviewers,
	}

	return pr, nil
}

func (r *pullRequestRepository) Exists(prID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	err := r.db.Get(&exists, query, prID)
	return exists, err
}

func (r *pullRequestRepository) UpdateStatus(id, status string, mergedAt *time.Time) error {
	query := `
		UPDATE pull_requests
		SET status = $2, merged_at = $3
		WHERE id = $1
	`
	result, err := r.db.Exec(query, id, status, mergedAt)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *pullRequestRepository) AssignReviewer(prInternalID, userInternalID string) error {
	query := `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (pull_request_id, user_id) DO NOTHING
	`
	_, err := r.db.Exec(query, prInternalID, userInternalID)
	return err
}

func (r *pullRequestRepository) RemoveReviewer(prInternalID, userInternalID string) error {
	query := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
	_, err := r.db.Exec(query, prInternalID, userInternalID)
	return err
}

func (r *pullRequestRepository) GetReviewerUserIDs(prInternalID string) ([]string, error) {
	query := `
		SELECT u.user_id
		FROM pr_reviewers pr
		JOIN users u ON pr.user_id = u.id
		WHERE pr.pull_request_id = $1
		ORDER BY pr.assigned_at
	`
	var reviewers []string
	err := r.db.Select(&reviewers, query, prInternalID)
	if err != nil {
		return nil, err
	}
	
	if reviewers == nil {
		reviewers = []string{}
	}
	
	return reviewers, nil
}

func (r *pullRequestRepository) GetPRsByReviewerUserID(userID string) ([]model.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, u.user_id as author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers rev ON pr.id = rev.pull_request_id
		INNER JOIN users reviewer ON rev.user_id = reviewer.id
		INNER JOIN users u ON pr.author_id = u.id
		WHERE reviewer.user_id = $1
		ORDER BY pr.created_at DESC
	`
	var prs []model.PullRequestShort
	err := r.db.Select(&prs, query, userID)
	if err != nil {
		return nil, err
	}
	
	if prs == nil {
		prs = []model.PullRequestShort{}
	}
	
	return prs, nil
}

func (r *pullRequestRepository) IsReviewerAssigned(prInternalID, userInternalID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pr_reviewers 
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`
	err := r.db.Get(&exists, query, prInternalID, userInternalID)
	return exists, err
}

func (r *pullRequestRepository) AssignReviewersBatch(prInternalID string, userInternalIDs []string) error {
	if len(userInternalIDs) == 0 {
		return nil
	}
	
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (pull_request_id, user_id) DO NOTHING
	`
	
	for _, userID := range userInternalIDs {
		if _, err := tx.Exec(query, prInternalID, userID); err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

func (r *pullRequestRepository) RemoveAllReviewers(prInternalID string) error {
	query := `DELETE FROM pr_reviewers WHERE pull_request_id = $1`
	_, err := r.db.Exec(query, prInternalID)
	return err
}

func (r *pullRequestRepository) GetReviewerAssignmentCount(userInternalID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(DISTINCT pr.id)
		FROM pr_reviewers rev
		JOIN pull_requests pr ON rev.pull_request_id = pr.id
		WHERE rev.user_id = $1 AND pr.status = 'OPEN'
	`
	err := r.db.Get(&count, query, userInternalID)
	return count, err
}