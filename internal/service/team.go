package service

import (
	"database/sql"

	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/errors"
	"assign-reviewers-for-pull-requests/internal/model"
	"assign-reviewers-for-pull-requests/internal/repository"
)

type TeamService interface {
	CreateTeam(team *model.Team) (*model.Team, error)
	GetTeam(teamName string) (*model.Team, error)
}

type teamService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

func NewTeamService(repos *repository.Repositories, logger *zap.Logger) TeamService {
	return &teamService{
		repos:  repos,
		logger: logger,
	}
}

func (s *teamService) CreateTeam(team *model.Team) (*model.Team, error) {
	exists, err := s.repos.Team.Exists(team.TeamName)
	if err != nil {
		s.logger.Error("Failed to check team existence", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	if exists {
		return nil, errors.ErrTeamExists(team.TeamName)
	}

	teamID, err := s.repos.Team.Create(team.TeamName)
	if err != nil {
		s.logger.Error("Failed to create team", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	for _, member := range team.Members {
		if err := s.repos.User.Upsert(member.UserID, member.Username, teamID, member.IsActive); err != nil {
			s.logger.Error("Failed to upsert user",
				zap.String("user_id", member.UserID),
				zap.Error(err),
			)
			return nil, errors.ErrInternal(err)
		}
	}

	s.logger.Info("Team created successfully", zap.String("team_name", team.TeamName))

	return s.GetTeam(team.TeamName)
}

func (s *teamService) GetTeam(teamName string) (*model.Team, error) {
	team, err := s.repos.Team.Get(teamName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("team")
		}
		s.logger.Error("Failed to get team", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	return team, nil
}