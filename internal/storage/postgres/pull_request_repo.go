package postgres

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
	"github.com/VechkanovVV/assigner-pr/internal/storage"
)

// PullRequestRepository - репозиторий, для управлением pr'ами в Postgres.
type PullRequestRepository struct {
	pool *pgxpool.Pool
}

// NewPullRequestRepository создаёт экземпляр *PullRequestRepository
func NewPullRequestRepository(pool *pgxpool.Pool) *PullRequestRepository {
	return &PullRequestRepository{pool: pool}
}

// Create создаёт pr с ревьюверами.
func (p *PullRequestRepository) Create(ctx context.Context, pr storage.PullRequest) *apperrors.AppError {
	const prInsertQuery = `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
        VALUES ($1, $2, $3, $4, $5)
	`
	const reviewInsertQuery = `INSERT INTO reviews (pull_request_id, reviewer_id) VALUES ($1, $2)`

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("begin tx failed: %v", err)
		return &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}

	defer func() {
		if rerr := tx.Rollback(ctx); rerr != nil && !errors.Is(rerr, pgx.ErrTxClosed) {
			log.Printf("tx rollback failed: %v", rerr)
		}
	}()

	_, err = tx.Exec(ctx, prInsertQuery, pr.ID, pr.Name, pr.AuthorID, pr.Status, pr.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &apperrors.AppError{
				Code:    apperrors.ErrPRExists,
				Message: apperrors.FromCode(apperrors.ErrPRExists),
			}
		}
		log.Printf("inserting pr failed: %v", err)
		return &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}

	for _, rev := range pr.AssignedReviewers {
		_, err := tx.Exec(ctx, reviewInsertQuery, pr.ID, rev)
		if err != nil {
			log.Printf("insert reviewer failed: %v", err)
			return &apperrors.AppError{
				Code:    apperrors.ErrInternalIssue,
				Message: apperrors.FromCode(apperrors.ErrInternalIssue),
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("commit failed: %v", err)
		return &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}
	return nil
}

// Get возвращает pr по id.
func (p *PullRequestRepository) Get(ctx context.Context, prID string) (storage.PullRequest, *apperrors.AppError) {
	const prQuery = `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
        FROM pull_requests WHERE pull_request_id = $1
	`
	const revQuery = `SELECT reviewer_id FROM reviews WHERE pull_request_id = $1`

	var pr storage.PullRequest

	err := p.pool.QueryRow(ctx, prQuery, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.Is(err, pgx.ErrNoRows) {
			appErr = &apperrors.AppError{
				Code:    apperrors.ErrNotFound,
				Message: apperrors.FromCode(apperrors.ErrNotFound),
			}
			return pr, appErr
		}
		log.Printf("query pr failed: %v", err)
		appErr = &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return pr, appErr
	}

	rows, err := p.pool.Query(ctx, revQuery, prID)
	if err != nil {
		log.Printf("query reviewer failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return pr, appErr
	}

	defer rows.Close()

	for rows.Next() {
		var rev string
		if err := rows.Scan(&rev); err != nil {
			log.Printf("reviewer scan failed: %v", err)
			appErr := &apperrors.AppError{
				Code:    apperrors.ErrInternalIssue,
				Message: apperrors.FromCode(apperrors.ErrInternalIssue),
			}
			return pr, appErr
		}

		pr.AssignedReviewers = append(pr.AssignedReviewers, rev)
	}

	if err := rows.Err(); err != nil {
		log.Printf("%v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return pr, appErr
	}
	return pr, nil
}

// Exists проверяет существование pr.
func (p *PullRequestRepository) Exists(ctx context.Context, prID string) (bool, *apperrors.AppError) {
	const query = `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, query, prID).Scan(&exists)
	if err != nil {
		log.Printf("query failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return false, appErr
	}

	return exists, nil
}

// MarkMerged проверяет pr как MERGED.
func (p *PullRequestRepository) MarkMerged(ctx context.Context, prID string) (storage.PullRequest, *apperrors.AppError) {
	const query = `
		UPDATE pull_requests
		SET status = 'MERGED', merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
	`
	ct, err := p.pool.Exec(ctx, query, prID)
	if err != nil {
		log.Printf("update failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return storage.PullRequest{}, appErr
	}

	if ct.RowsAffected() == 0 {
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrNotFound,
			Message: apperrors.FromCode(apperrors.ErrNotFound),
		}
		return storage.PullRequest{}, appErr
	}

	return p.Get(ctx, prID)
}

// ReplaceReviewer заменяет одного ревьюера на другого.
func (p *PullRequestRepository) ReplaceReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) *apperrors.AppError {
	const query = `
		UPDATE reviews SET reviewer_id = $3, assigned_at = NOW()
		WHERE pull_request_id = $1 AND reviewer_id = $2
	`

	ct, err := p.pool.Exec(ctx, query, prID, oldReviewerID, newReviewerID)
	if err != nil {
		log.Printf("update rev failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return appErr
	}

	if ct.RowsAffected() == 0 {
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrNotAssigned,
			Message: apperrors.FromCode(apperrors.ErrNotAssigned),
		}
		return appErr
	}

	return nil
}

// GetByReviewer возвращет все pr пользоавтель. где он ревьюер.
func (p *PullRequestRepository) GetByReviewer(ctx context.Context, reviewerID string) ([]storage.PullRequest, *apperrors.AppError) {
	const query = `
		SELECT DISTINCT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
        FROM pull_requests pr
        INNER JOIN reviews r ON r.pull_request_id = pr.pull_request_id
        WHERE r.reviewer_id = $1 
	`
	rows, err := p.pool.Query(ctx, query, reviewerID)
	if err != nil {
		log.Printf("query failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return nil, appErr
	}

	defer rows.Close()

	var prs []storage.PullRequest
	for rows.Next() {
		var pr storage.PullRequest
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt); err != nil {
			log.Printf("scan failed: %v", err)
			appErr := &apperrors.AppError{
				Code:    apperrors.ErrInternalIssue,
				Message: apperrors.FromCode(apperrors.ErrInternalIssue),
			}
			return nil, appErr
		}

		prs = append(prs, pr)
	}

	if err := rows.Err(); err != nil {
		log.Printf("%v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}

		return nil, appErr
	}
	return prs, nil
}

// IsReviewerAssigned проверяет, является ли пользователь ревьюером.
func (p *PullRequestRepository) IsReviewerAssigned(ctx context.Context, reviewerID string) (bool, *apperrors.AppError) {
	const query = `SELECT EXISTS(SELECT 1 FROM reviews WHERE reviewer_id = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, query, reviewerID).Scan(&exists)
	if err != nil {
		log.Printf("query failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return false, appErr
	}

	return exists, nil
}
