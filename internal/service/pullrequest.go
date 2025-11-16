package service

import (
	"database/sql"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"assign-reviewers-for-pull-requests/internal/errors"
	"assign-reviewers-for-pull-requests/internal/model"
	"assign-reviewers-for-pull-requests/internal/repository"
)

type PullRequestService interface {
	CreatePR(req *model.CreatePRRequest) (*model.PullRequest, error)
	MergePR(prID string) (*model.PullRequest, error)
	ReassignReviewer(prID, oldUserID string) (*model.PullRequest, string, error)
}

type pullRequestService struct {
	repos  *repository.Repositories
	logger *zap.Logger
	rnd    *rand.Rand
}

func NewPullRequestService(repos *repository.Repositories, logger *zap.Logger) PullRequestService {
	source := rand.NewSource(time.Now().UnixNano())
	return &pullRequestService{
		repos:  repos,
		logger: logger,
		rnd:    rand.New(source),
	}
}

func (s *pullRequestService) CreatePR(req *model.CreatePRRequest) (*model.PullRequest, error) {
	exists, err := s.repos.PullRequest.Exists(req.PullRequestID)
	if err != nil {
		s.logger.Error("Failed to check PR existence", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}
	if exists {
		return nil, errors.ErrPRExists(req.PullRequestID)
	}

	author, err := s.repos.User.GetByUserID(req.AuthorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("author")
		}
		s.logger.Error("Failed to get author", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	if author.TeamID == "" {
		return nil, errors.ErrNotFound("author team")
	}

	if !author.IsActive {
		return nil, errors.ErrNotFound("author is inactive")
	}

	prInternalID, err := s.repos.PullRequest.Create(req.PullRequestID, req.PullRequestName, author.ID)
	if err != nil {
		s.logger.Error("Failed to create PR", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	reviewers, err := s.selectReviewersWithBalancing(author.TeamID, []string{author.ID}, 2)
	if err != nil {
		s.logger.Error("Failed to select reviewers", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	if len(reviewers) == 0 {
		s.logger.Warn("No reviewers available for PR",
			zap.String("pr_id", req.PullRequestID),
			zap.String("team_id", author.TeamID),
		)
		return nil, errors.ErrNoCandidate()
	}

	reviewerIDs := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		reviewerIDs[i] = reviewer.ID
	}

	if err := s.repos.PullRequest.AssignReviewersBatch(prInternalID, reviewerIDs); err != nil {
		s.logger.Error("Failed to assign reviewers", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	for _, reviewer := range reviewers {
		_ = s.repos.Stats.RecordAssignment(reviewer.ID, prInternalID)
	}

	reviewerUserIDs := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		reviewerUserIDs[i] = reviewer.UserID
	}

	pr := &model.PullRequest{
		ID:                prInternalID,
		PullRequestID:     req.PullRequestID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          author.UserID,
		Status:            "OPEN",
		CreatedAt:         time.Now(),
		AssignedReviewers: reviewerUserIDs,
	}

	s.logger.Info("PR created successfully",
		zap.String("pr_id", req.PullRequestID),
		zap.Strings("reviewers", reviewerUserIDs),
	)

	return pr, nil
}

func (s *pullRequestService) MergePR(prID string) (*model.PullRequest, error) {
	pr, err := s.repos.PullRequest.GetByPRID(prID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("pull request")
		}
		s.logger.Error("Failed to get PR", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	if pr.Status == "MERGED" {
		s.logger.Info("PR already merged", zap.String("pr_id", prID))
		return pr, nil
	}

	mergedAt := time.Now()
	if err := s.repos.PullRequest.UpdateStatus(pr.ID, "MERGED", &mergedAt); err != nil {
		s.logger.Error("Failed to update PR status", zap.Error(err))
		return nil, errors.ErrInternal(err)
	}

	pr.Status = "MERGED"
	pr.MergedAt = sql.NullTime{Time: mergedAt, Valid: true}

	s.logger.Info("PR merged successfully", zap.String("pr_id", prID))

	return pr, nil
}

func (s *pullRequestService) ReassignReviewer(prID, oldUserID string) (*model.PullRequest, string, error) {
	pr, err := s.repos.PullRequest.GetByPRID(prID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", errors.ErrNotFound("pull request")
		}
		s.logger.Error("Failed to get PR", zap.Error(err))
		return nil, "", errors.ErrInternal(err)
	}

	if pr.Status == "MERGED" {
		return nil, "", errors.ErrPRMerged()
	}

	oldUser, err := s.repos.User.GetByUserID(oldUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", errors.ErrNotFound("user")
		}
		return nil, "", errors.ErrInternal(err)
	}

	isAssigned, err := s.repos.PullRequest.IsReviewerAssigned(pr.ID, oldUser.ID)
	if err != nil {
		s.logger.Error("Failed to check reviewer assignment", zap.Error(err))
		return nil, "", errors.ErrInternal(err)
	}
	if !isAssigned {
		return nil, "", errors.ErrNotAssigned()
	}

	author, err := s.repos.User.GetByUserID(pr.AuthorID)
	if err != nil {
		return nil, "", errors.ErrInternal(err)
	}

	currentReviewerIDs := []string{author.ID, oldUser.ID}
	for _, rUserID := range pr.AssignedReviewers {
		if rUserID != oldUserID {
			u, _ := s.repos.User.GetByUserID(rUserID)
			if u != nil {
				currentReviewerIDs = append(currentReviewerIDs, u.ID)
			}
		}
	}

	newReviewers, err := s.selectReviewers(author.TeamID, currentReviewerIDs, 1)
	if err != nil {
		s.logger.Error("Failed to select new reviewer", zap.Error(err))
		return nil, "", errors.ErrInternal(err)
	}

	if len(newReviewers) == 0 {
		return nil, "", errors.ErrNoCandidate()
	}

	newReviewer := newReviewers[0]

	if err := s.repos.PullRequest.RemoveReviewer(pr.ID, oldUser.ID); err != nil {
		s.logger.Error("Failed to remove old reviewer", zap.Error(err))
		return nil, "", errors.ErrInternal(err)
	}

	if err := s.repos.PullRequest.AssignReviewer(pr.ID, newReviewer.ID); err != nil {
		s.logger.Error("Failed to assign new reviewer", zap.Error(err))
		return nil, "", errors.ErrInternal(err)
	}

	_ = s.repos.Stats.RecordAssignment(newReviewer.ID, pr.ID)

	for i, rUserID := range pr.AssignedReviewers {
		if rUserID == oldUserID {
			pr.AssignedReviewers[i] = newReviewer.UserID
			break
		}
	}

	s.logger.Info("Reviewer reassigned successfully",
		zap.String("pr_id", prID),
		zap.String("old_reviewer", oldUserID),
		zap.String("new_reviewer", newReviewer.UserID),
	)

	return pr, newReviewer.UserID, nil
}

func (s *pullRequestService) selectReviewers(teamID string, excludeInternalIDs []string, count int) ([]model.User, error) {
	activeUsers, err := s.repos.User.GetActiveByTeamID(teamID, excludeInternalIDs)
	if err != nil {
		return nil, err
	}

	if len(activeUsers) <= count {
		return activeUsers, nil
	}

	s.rnd.Shuffle(len(activeUsers), func(i, j int) {
		activeUsers[i], activeUsers[j] = activeUsers[j], activeUsers[i]
	})

	return activeUsers[:count], nil
}


func (s *pullRequestService) selectReviewersWithBalancing(teamID string, excludeInternalIDs []string, count int) ([]model.User, error) {
	activeUsers, err := s.repos.User.GetActiveByTeamID(teamID, excludeInternalIDs)
	if err != nil {
		return nil, err
	}

	if len(activeUsers) == 0 {
		return []model.User{}, nil
	}

	if len(activeUsers) <= count {
		return activeUsers, nil
	}

	type userWithCount struct {
		user  model.User
		count int
	}

	usersWithCounts := make([]userWithCount, len(activeUsers))
	for i, user := range activeUsers {
		assignmentCount, err := s.repos.PullRequest.GetReviewerAssignmentCount(user.ID)
		if err != nil {
			s.logger.Warn("Failed to get assignment count",
				zap.String("user_id", user.UserID),
				zap.Error(err),
			)
			assignmentCount = 0
		}
		usersWithCounts[i] = userWithCount{user: user, count: assignmentCount}
	}

	for i := 0; i < len(usersWithCounts); i++ {
		for j := i + 1; j < len(usersWithCounts); j++ {
			if usersWithCounts[j].count < usersWithCounts[i].count {
				usersWithCounts[i], usersWithCounts[j] = usersWithCounts[j], usersWithCounts[i]
			}
		}
	}

	result := make([]model.User, count)
	for i := 0; i < count; i++ {
		result[i] = usersWithCounts[i].user
	}

	return result, nil
}