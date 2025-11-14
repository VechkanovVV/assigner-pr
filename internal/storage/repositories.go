package storage

import "context"

// UserRepository - репозиторий для упраления пользователями.
type UserRepository interface {
	Get(ctx context.Context, userID string) (User, error)
	SetActive(ctx context.Context, userID string, isActive bool) (User, error)
	GetActiveTeammates(ctx context.Context, teamName, excludedID string) ([]User, error)
	Exists(ctx context.Context, userID string) (bool, error)
}

// TeamRepository - репозиторий для управления командами.
type TeamRepository interface {
	Create(ctx context.Context, team Team) error
	Get(ctx context.Context, teamName string) (Team, error)
	Exists(ctx context.Context, teamName string) (bool, error)
}

// PullRequestRepository - репозиторий для управления Pull Request'ами.
type PullRequestRepository interface {
	Create(ctx context.Context, pr PullRequest) error
	Get(ctx context.Context, prID string) (PullRequest, error)
	Exists(ctx context.Context, prID string) (bool, error)
	MarkMerged(ctx context.Context, prID string) (PullRequest, error)
	ReplaceReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error
	GetByReviewer(ctx context.Context, reviewerID string) ([]PullRequest, error)
	IsReviewerAssigned(ctx context.Context, reviewerID string) (bool, error)
}
