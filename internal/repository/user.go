package repository

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"assign-reviewers-for-pull-requests/internal/model"
)

type UserRepository interface {
	Upsert(userID, username, teamID string, isActive bool) error
	GetByUserID(userID string) (*model.User, error)
	GetByID(id string) (*model.User, error)
	GetActiveByTeamID(teamID string, excludeIDs []string) ([]model.User, error)
	SetIsActive(userID string, isActive bool) error
	
	// Get user internal ID by user_id
	GetIDByUserID(userID string) (string, error)
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Upsert(userID, username, teamID string, isActive bool) error {
	query := `
		INSERT INTO users (user_id, username, team_id, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			team_id = EXCLUDED.team_id,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`
	_, err := r.db.Exec(query, userID, username, teamID, isActive)
	return err
}

func (r *userRepository) GetByUserID(userID string) (*model.User, error) {
	query := `
		SELECT u.id, u.user_id, u.username, u.team_id, t.team_name, u.is_active
		FROM users u
		LEFT JOIN teams t ON u.team_id = t.id
		WHERE u.user_id = $1
	`
	var user model.User
	err := r.db.Get(&user, query, userID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(id string) (*model.User, error) {
	query := `
		SELECT u.id, u.user_id, u.username, u.team_id, t.team_name, u.is_active
		FROM users u
		LEFT JOIN teams t ON u.team_id = t.id
		WHERE u.id = $1
	`
	var user model.User
	err := r.db.Get(&user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetActiveByTeamID(teamID string, excludeIDs []string) ([]model.User, error) {
	query := `
		SELECT u.id, u.user_id, u.username, u.team_id, t.team_name, u.is_active
		FROM users u
		LEFT JOIN teams t ON u.team_id = t.id
		WHERE u.team_id = $1 AND u.is_active = true
	`
	
	args := []interface{}{teamID}
	
	if len(excludeIDs) > 0 {
		query += ` AND u.id != ALL($2)`
		args = append(args, pq.Array(excludeIDs))
	}
	
	query += ` ORDER BY u.username`
	
	var users []model.User
	err := r.db.Select(&users, query, args...)
	if err != nil {
		return nil, err
	}
	
	if users == nil {
		users = []model.User{}
	}
	
	return users, nil
}

func (r *userRepository) SetIsActive(userID string, isActive bool) error {
	query := `
		UPDATE users
		SET is_active = $2, updated_at = NOW()
		WHERE user_id = $1
	`
	result, err := r.db.Exec(query, userID, isActive)
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

func (r *userRepository) GetIDByUserID(userID string) (string, error) {
	var id string
	query := `SELECT id FROM users WHERE user_id = $1`
	err := r.db.Get(&id, query, userID)
	return id, err
}