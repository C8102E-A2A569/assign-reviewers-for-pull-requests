package service

import (
	"testing"

	"assign-reviewers-for-pull-requests/internal/model"
	"assign-reviewers-for-pull-requests/internal/repository"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func TestCreateTeam_Success(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewTeamService(repos, logger)

	team := &model.Team{
		TeamName: "backend",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	createdTeam, err := service.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	if createdTeam.TeamName != "backend" {
		t.Errorf("Expected team name 'backend', got '%s'", createdTeam.TeamName)
	}

	if len(createdTeam.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(createdTeam.Members))
	}
}

func TestCreateTeam_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewTeamService(repos, logger)

	team := &model.Team{
		TeamName: "backend",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}

	_, err := service.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create first team: %v", err)
	}

	_, err = service.CreateTeam(team)
	if err == nil {
		t.Error("Expected error when creating duplicate team")
	}
}

func TestGetTeam_Success(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewTeamService(repos, logger)

	team := &model.Team{
		TeamName: "backend",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: false},
		},
	}

	_, err := service.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	retrievedTeam, err := service.GetTeam("backend")
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}

	if retrievedTeam.TeamName != "backend" {
		t.Errorf("Expected team name 'backend', got '%s'", retrievedTeam.TeamName)
	}

	if len(retrievedTeam.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(retrievedTeam.Members))
	}
}

func TestGetTeam_NotFound(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewTeamService(repos, logger)

	_, err := service.GetTeam("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent team")
	}
}