package service

import (
	"testing"
	"assign-reviewers-for-pull-requests/internal/model"
	"assign-reviewers-for-pull-requests/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
		return nil
	}

	createSchema(t, db)

	_, _ = db.Exec("TRUNCATE TABLE assignment_stats CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE pr_reviewers CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE pull_requests CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE users CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE teams CASCADE")

	return db
}

func createSchema(t *testing.T, db *sqlx.DB) {

	var exists bool
	err := db.Get(&exists, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'teams')")
	if err == nil && exists {
		return 
	}


	schema := `
		CREATE TABLE IF NOT EXISTS teams (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			team_name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL UNIQUE,
			username VARCHAR(255) NOT NULL,
			team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);
		CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
		CREATE INDEX IF NOT EXISTS idx_users_user_id ON users(user_id);

		CREATE TABLE IF NOT EXISTS pull_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			pull_request_id VARCHAR(255) NOT NULL UNIQUE,
			pull_request_name VARCHAR(500) NOT NULL,
			author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			merged_at TIMESTAMP,
			CONSTRAINT chk_merged_at CHECK (
				(status = 'MERGED' AND merged_at IS NOT NULL) OR
				(status = 'OPEN' AND merged_at IS NULL)
			)
		);

		CREATE INDEX IF NOT EXISTS idx_pr_author_id ON pull_requests(author_id);
		CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);
		CREATE INDEX IF NOT EXISTS idx_pr_created_at ON pull_requests(created_at);
		CREATE INDEX IF NOT EXISTS idx_pr_pull_request_id ON pull_requests(pull_request_id);

		CREATE TABLE IF NOT EXISTS pr_reviewers (
			pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (pull_request_id, user_id)
		);

		CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pr_reviewers(user_id);

		CREATE TABLE IF NOT EXISTS assignment_stats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			pr_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
			assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, pr_id)
		);

		CREATE INDEX IF NOT EXISTS idx_stats_user_id ON assignment_stats(user_id);
		CREATE INDEX IF NOT EXISTS idx_stats_pr_id ON assignment_stats(pr_id);
		CREATE INDEX IF NOT EXISTS idx_stats_assigned_at ON assignment_stats(assigned_at);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
}

func createTestTeam(t *testing.T, repos *repository.Repositories, teamName string, users []model.TeamMember) string {
	teamID, err := repos.Team.Create(teamName)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	for _, user := range users {
		err := repos.User.Upsert(user.UserID, user.Username, teamID, user.IsActive)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	return teamID
}

func TestCreatePR_Success(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
		{UserID: "u3", Username: "Charlie", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	pr, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	if pr.PullRequestID != "pr-001" {
		t.Errorf("Expected PR ID 'pr-001', got '%s'", pr.PullRequestID)
	}

	if pr.Status != "OPEN" {
		t.Errorf("Expected status 'OPEN', got '%s'", pr.Status)
	}

	if len(pr.AssignedReviewers) == 0 {
		t.Error("Expected at least 1 reviewer assigned")
	}

	if len(pr.AssignedReviewers) > 2 {
		t.Errorf("Expected max 2 reviewers, got %d", len(pr.AssignedReviewers))
	}


	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == "u1" {
			t.Error("Author should not be assigned as reviewer")
		}
	}
}

func TestCreatePR_DuplicateID(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	_, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create first PR: %v", err)
	}

	_, err = service.CreatePR(req)
	if err == nil {
		t.Error("Expected error when creating duplicate PR")
	}
}

func TestCreatePR_NoActiveReviewers(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: false},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	_, err := service.CreatePR(req)
	if err == nil {
		t.Error("Expected error when no active reviewers available")
	}
}

func TestMergePR_Success(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	_, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	pr, err := service.MergePR("pr-001")
	if err != nil {
		t.Fatalf("Failed to merge PR: %v", err)
	}

	if pr.Status != "MERGED" {
		t.Errorf("Expected status 'MERGED', got '%s'", pr.Status)
	}

	if !pr.MergedAt.Valid {
		t.Error("Expected MergedAt to be set")
	}
}

func TestMergePR_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	_, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	pr1, err := service.MergePR("pr-001")
	if err != nil {
		t.Fatalf("Failed to merge PR first time: %v", err)
	}

	pr2, err := service.MergePR("pr-001")
	if err != nil {
		t.Fatalf("Failed to merge PR second time: %v", err)
	}

	if pr1.Status != pr2.Status {
		t.Errorf("Status should be the same after repeated merge")
	}
}

func TestReassignReviewer_Success(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
		{UserID: "u3", Username: "Charlie", IsActive: true},
		{UserID: "u4", Username: "David", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	pr, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	if len(pr.AssignedReviewers) == 0 {
		t.Fatal("No reviewers assigned")
	}

	oldReviewer := pr.AssignedReviewers[0]

	newPR, replacedBy, err := service.ReassignReviewer("pr-001", oldReviewer)
	if err != nil {
		t.Fatalf("Failed to reassign reviewer: %v", err)
	}

	if replacedBy == oldReviewer {
		t.Error("New reviewer should be different from old reviewer")
	}

	found := false
	for _, r := range newPR.AssignedReviewers {
		if r == replacedBy {
			found = true
			break
		}
	}
	if !found {
		t.Error("New reviewer not found in assigned reviewers")
	}
}

func TestReassignReviewer_AfterMerge(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
		{UserID: "u3", Username: "Charlie", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	pr, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	_, err = service.MergePR("pr-001")
	if err != nil {
		t.Fatalf("Failed to merge PR: %v", err)
	}

	if len(pr.AssignedReviewers) == 0 {
		t.Fatal("No reviewers assigned")
	}

	_, _, err = service.ReassignReviewer("pr-001", pr.AssignedReviewers[0])
	if err == nil {
		t.Error("Expected error when reassigning after merge")
	}
}

func TestReassignReviewer_NotAssigned(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	logger, _ := zap.NewDevelopment()
	service := NewPullRequestService(repos, logger)

	users := []model.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
		{UserID: "u3", Username: "Charlie", IsActive: true},
	}
	createTestTeam(t, repos, "backend", users)

	req := &model.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}

	_, err := service.CreatePR(req)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}


	_, _, err = service.ReassignReviewer("pr-001", "u1")
	if err == nil {
		t.Error("Expected error when reassigning user not assigned as reviewer")
	}
}