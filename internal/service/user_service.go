package service

import (
	"context"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
	"github.com/VechkanovVV/assigner-pr/internal/storage"
)

// UserService - сервис для управления пользователями.
type UserService struct {
	userRepo storage.UserRepository
	prRepo   storage.PullRequestRepository
}

// NewUserService возвращает новый UserService.
func NewUserService(userRepo storage.UserRepository, prRepo storage.PullRequestRepository) *UserService {
	return &UserService{userRepo: userRepo, prRepo: prRepo}
}

// SetActiveStatus устанавливает флаг активности у пользователя.
func (u *UserService) SetActiveStatus(ctx context.Context, userID string, isActive bool) (storage.User, *apperrors.AppError) {
	return u.userRepo.SetActive(ctx, userID, isActive)
}

// GetUserReviews возвращает pr'ы, где пользователь ревьюер.
func (u *UserService) GetUserReviews(ctx context.Context, userID string) ([]storage.PullRequest, *apperrors.AppError) {
	exists, err := u.userRepo.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &apperrors.AppError{
			Code:    apperrors.ErrNotFound,
			Message: apperrors.FromCode(apperrors.ErrNotFound),
		}
	}

	prs, err := u.prRepo.GetByReviewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	return prs, nil
}
