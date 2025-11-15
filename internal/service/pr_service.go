package service

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
	"github.com/VechkanovVV/assigner-pr/internal/storage"
)

// PRService управляет pr'ами.
type PRService struct {
	userRepo storage.UserRepository
	prRepo   storage.PullRequestRepository
}

// NewPRService создаёт новый PRService.
func NewPRService(userRepo storage.UserRepository, prRepo storage.PullRequestRepository) *PRService {
	return &PRService{userRepo: userRepo, prRepo: prRepo}
}

// CreatePR создаёт новый Pull Request, назначает ревьюеров и сохраняет его в репозитории.
func (p *PRService) CreatePR(ctx context.Context, prID, prName, authorID string) (storage.PullRequest, *apperrors.AppError) {
	auth, err := p.userRepo.Get(ctx, authorID)
	if err != nil {
		return storage.PullRequest{}, err
	}

	ts, err := p.userRepo.GetActiveTeammates(ctx, auth.TeamID, auth.ID)
	if err != nil {
		return storage.PullRequest{}, err
	}

	pick, pickErr := pickRev(ts, 2)
	if pickErr != nil {
		log.Println(fmt.Errorf("picking reviewers failed: %w", pickErr))
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}

		return storage.PullRequest{}, appErr
	}

	revID := make([]string, 0, len(pick))
	for _, u := range pick {
		revID = append(revID, u.ID)
	}

	pr := storage.PullRequest{
		ID:                prID,
		Name:              prName,
		AuthorID:          authorID,
		Status:            storage.StatusOpen,
		CreatedAt:         time.Now().UTC(),
		AssignedReviewers: revID,
	}

	if err := p.prRepo.Create(ctx, pr); err != nil {
		return storage.PullRequest{}, err
	}

	return pr, nil
}

// Merge - меняет флаг у pr на merged.
func (p *PRService) Merge(ctx context.Context, prID string) (storage.PullRequest, *apperrors.AppError) {
	pr, err := p.prRepo.MarkMerged(ctx, prID)
	if err != nil {
		return pr, err
	}
	return pr, nil
}

// ReassignReviewer - меняет ревьюера.
func (p *PRService) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (storage.PullRequest, string, *apperrors.AppError) {
	pr, err := p.prRepo.Get(ctx, prID)
	if err != nil {
		return storage.PullRequest{}, "", err
	}

	if pr.Status == storage.StatusMerged {
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrPRMerged,
			Message: apperrors.FromCode(apperrors.ErrPRMerged),
		}
		return storage.PullRequest{}, "", appErr
	}

	var check bool
	for _, u := range pr.AssignedReviewers {
		if u == oldReviewerID {
			check = true
			break
		}
	}

	if !check {
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrNotAssigned,
			Message: apperrors.FromCode(apperrors.ErrNotAssigned),
		}
		return storage.PullRequest{}, "", appErr
	}

	oldRev, err := p.userRepo.Get(ctx, oldReviewerID)
	if err != nil {
		return storage.PullRequest{}, "", err
	}

	cands, err := p.userRepo.GetActiveTeammates(ctx, oldRev.TeamID, oldReviewerID)
	if err != nil {
		return storage.PullRequest{}, "", err
	}

	visit := make(map[string]bool, len(pr.AssignedReviewers)+1)
	visit[pr.AuthorID] = true
	for _, val := range pr.AssignedReviewers {
		visit[val] = true
	}

	var filteredCands []storage.User
	for _, c := range cands {
		if !visit[c.ID] {
			filteredCands = append(filteredCands, c)
		}
	}

	if len(filteredCands) == 0 {
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrNoCandidate,
			Message: apperrors.FromCode(apperrors.ErrNoCandidate),
		}
		return storage.PullRequest{}, "", appErr
	}

	idx, pickErr := randInt(len(filteredCands))
	if pickErr != nil {
		log.Println(fmt.Errorf("rand pick failed: %w", pickErr))
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return storage.PullRequest{}, "", appErr
	}
	newCandidate := filteredCands[idx]

	if err := p.prRepo.ReplaceReviewer(ctx, prID, oldReviewerID, newCandidate.ID); err != nil {
		return storage.PullRequest{}, "", err
	}

	updatedPR, err := p.prRepo.Get(ctx, prID)
	if err != nil {
		return storage.PullRequest{}, "", err
	}

	return updatedPR, newCandidate.ID, nil
}

// pickRev выбирает случайных reviewer'ов из списка users.
func pickRev(users []storage.User, amount int) ([]storage.User, error) {
	if amount <= 0 || len(users) == 0 {
		return []storage.User{}, nil
	}

	if len(users) <= amount {
		return users, nil
	}

	pUsers := make([]storage.User, 0, amount)
	used := make(map[int]bool, amount)
	for len(pUsers) < amount {
		i, err := randInt(len(users))
		if err != nil {
			return nil, err
		}
		if _, u := used[i]; u {
			continue
		}
		used[i] = true
		pUsers = append(pUsers, users[i])
	}
	return pUsers, nil
}

// RandInt возвращает случайное число в диапазоне [0, n).
func randInt(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("invalid upper bound: %d", n)
	}
	bn := big.NewInt(int64(n))
	x, err := crand.Int(crand.Reader, bn)
	if err != nil {
		return 0, fmt.Errorf("getting rand failed: %w", err)
	}
	return int(x.Int64()), nil
}
