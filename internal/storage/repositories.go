package storage

import (
	"context"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
)

// UserRepository - репозиторий для упраления пользователями.
type UserRepository interface {
	Get(ctx context.Context, userID string) (User, *apperrors.AppError)
	SetActive(ctx context.Context, userID string, isActive bool) (User, *apperrors.AppError)
	GetActiveTeammates(ctx context.Context, teamID int, excludedID string) ([]User, *apperrors.AppError)
	Exists(ctx context.Context, userID string) (bool, *apperrors.AppError)
}

// TeamRepository - репозиторий для управления командами.
type TeamRepository interface {
	Create(ctx context.Context, team Team) *apperrors.AppError
	Get(ctx context.Context, teamName string) (Team, *apperrors.AppError)
	Exists(ctx context.Context, teamName string) (bool, *apperrors.AppError)
}

// PullRequestRepository - репозиторий для управления Pull Request'ами.
type PullRequestRepository interface {
	Create(ctx context.Context, pr PullRequest) *apperrors.AppError
	Get(ctx context.Context, prID string) (PullRequest, *apperrors.AppError)
	Exists(ctx context.Context, prID string) (bool, *apperrors.AppError)
	MarkMerged(ctx context.Context, prID string) (PullRequest, *apperrors.AppError)
	ReplaceReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) *apperrors.AppError
	GetByReviewer(ctx context.Context, reviewerID string) ([]PullRequest, *apperrors.AppError)
	IsReviewerAssigned(ctx context.Context, reviewerID string) (bool, *apperrors.AppError)
}
