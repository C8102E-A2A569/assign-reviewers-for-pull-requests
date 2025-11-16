package model

import (
	"database/sql"
	"time"
)

type User struct {
	ID       string `db:"id" json:"-"`
	UserID   string `db:"user_id" json:"user_id"`
	Username string `db:"username" json:"username"`
	TeamID   string `db:"team_id" json:"-"`
	TeamName string `db:"team_name" json:"team_name"`
	IsActive bool   `db:"is_active" json:"is_active"`
}

type Team struct {
	ID       string       `db:"id" json:"-"`
	TeamName string       `json:"team_name" binding:"required"`
	Members  []TeamMember `json:"members" binding:"required,dive"`
}

type TeamMember struct {
	UserID   string `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type PullRequest struct {
	ID                string       `db:"id" json:"-"`
	PullRequestID     string       `db:"pull_request_id" json:"pull_request_id"`
	PullRequestName   string       `db:"pull_request_name" json:"pull_request_name"`
	AuthorID          string       `db:"author_id" json:"author_id"`
	Status            string       `db:"status" json:"status"`
	CreatedAt         time.Time    `db:"created_at" json:"createdAt,omitempty"`
	MergedAt          sql.NullTime `db:"merged_at" json:"mergedAt,omitempty"`
	AssignedReviewers []string     `json:"assigned_reviewers"`
}

type PullRequestShort struct {
	PullRequestID   string `db:"pull_request_id" json:"pull_request_id"`
	PullRequestName string `db:"pull_request_name" json:"pull_request_name"`
	AuthorID        string `db:"author_id" json:"author_id"`
	Status          string `db:"status" json:"status"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id" binding:"required"`
	PullRequestName string `json:"pull_request_name" binding:"required"`
	AuthorID        string `json:"author_id" binding:"required"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
}

type ReassignPRRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
	OldUserID     string `json:"old_user_id" binding:"required"`
}

type SetIsActiveRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	IsActive bool   `json:"is_active"`
}