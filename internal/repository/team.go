package repository

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"assign-reviewers-for-pull-requests/internal/model"
)

type TeamRepository interface {
	Create(teamName string) (string, error)
	Exists(teamName string) (bool, error)
	Get(teamName string) (*model.Team, error)
	GetByID(teamID string) (*model.Team, error)
	GetIDByName(teamName string) (string, error)
}

type teamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) Create(teamName string) (string, error) {
	query := `INSERT INTO teams (team_name) VALUES ($1) RETURNING id`
	var id string
	err := r.db.Get(&id, query, teamName)
	return id, err
}

func (r *teamRepository) Exists(teamName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	err := r.db.Get(&exists, query, teamName)
	return exists, err
}

func (r *teamRepository) GetIDByName(teamName string) (string, error) {
	var id string
	query := `SELECT id FROM teams WHERE team_name = $1`
	err := r.db.Get(&id, query, teamName)
	return id, err
}

func (r *teamRepository) Get(teamName string) (*model.Team, error) {
	teamID, err := r.GetIDByName(teamName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return r.GetByID(teamID)
}

func (r *teamRepository) GetByID(teamID string) (*model.Team, error) {
	query := `SELECT id, team_name FROM teams WHERE id = $1`
	var team struct {
		ID       string `db:"id"`
		TeamName string `db:"team_name"`
	}
	err := r.db.Get(&team, query, teamID)
	if err != nil {
		return nil, err
	}

	members, err := r.getMembersByTeamID(teamID)
	if err != nil {
		return nil, err
	}

	return &model.Team{
		ID:       team.ID,
		TeamName: team.TeamName,
		Members:  members,
	}, nil
}

func (r *teamRepository) getMembersByTeamID(teamID string) ([]model.TeamMember, error) {
	query := `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_id = $1
		ORDER BY username
	`
	
	type userRow struct {
		UserID   string `db:"user_id"`
		Username string `db:"username"`
		IsActive bool   `db:"is_active"`
	}
	
	var rows []userRow
	err := r.db.Select(&rows, query, teamID)
	if err != nil {
		return nil, err
	}

	members := make([]model.TeamMember, len(rows))
	for i, row := range rows {
		members[i] = model.TeamMember{
			UserID:   row.UserID,
			Username: row.Username,
			IsActive: row.IsActive,
		}
	}

	return members, nil
}