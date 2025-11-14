package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
func (p *PullRequestRepository) Create(ctx context.Context, pr storage.PullRequest) error {
	const prInsertQuery = `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
        VALUES ($1, $2, $3, $4, $5)
	`
	const reviewInsertQuery = `INSERT INTO reviews (pull_request_id, reviewer_id) VALUES ($1, $2)`

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}

	defer func() {
		if rerr := tx.Rollback(ctx); rerr != nil && !errors.Is(rerr, pgx.ErrTxClosed) {
			log.Printf("tx rollback failed: %v", rerr)
		}
	}()

	_, err = tx.Exec(ctx, prInsertQuery, pr.ID, pr.Name, pr.AuthorID, pr.Status, pr.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert pr failed: %w", err)
	}

	for _, rev := range pr.AssignedReviewers {
		_, err := tx.Exec(ctx, reviewInsertQuery, pr.ID, rev)
		if err != nil {
			return fmt.Errorf("insert reviewer failed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

// Get возвращает pr по id.
func (p *PullRequestRepository) Get(ctx context.Context, prID string) (storage.PullRequest, error) {
	const prQuery = `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
        FROM pull_requests WHERE pull_request_id = $1
	`
	const revQuery = `SELECT reviewer_id FROM reviews WHERE pull_request_id = $1`

	var pr storage.PullRequest

	err := p.pool.QueryRow(ctx, prQuery, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pr, errors.New("pr not found")
		}
		return pr, fmt.Errorf("query pr failed: %w", err)
	}

	rows, err := p.pool.Query(ctx, revQuery, prID)
	if err != nil {
		return pr, fmt.Errorf("query reviewer failed: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var rev string
		if err := rows.Scan(&rev); err != nil {
			return pr, fmt.Errorf("reviewer scan failed: %w", err)
		}

		pr.AssignedReviewers = append(pr.AssignedReviewers, rev)
	}

	return pr, rows.Err()
}

// Exists проверяет существование pr.
func (p *PullRequestRepository) Exists(ctx context.Context, prID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, query, prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	return exists, nil
}

// MarkMerged проверяет pr как MERGED.
func (p *PullRequestRepository) MarkMerged(ctx context.Context, prID string) (storage.PullRequest, error) {
	const query = `
		UPDATE pull_requests
		SET status = 'MERGED', merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
	`
	_, err := p.pool.Exec(ctx, query, prID)
	if err != nil {
		return storage.PullRequest{}, fmt.Errorf("update failed: %w", err)
	}

	return p.Get(ctx, prID)
}

// ReplaceReviewer заменяет одного ревьюера на другого.
func (p *PullRequestRepository) ReplaceReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error {
	const query = `
		UPDATE reviews SET reviewer_id = $3, assigned_at = NOW()
		WHERE pull_request_id = $1 AND reviewer_id = $2
	`

	ct, err := p.pool.Exec(ctx, query, prID, oldReviewerID, newReviewerID)
	if err != nil {
		return fmt.Errorf("update rev failed: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return errors.New("reviewers not assigned")
	}

	return nil
}

// GetByReviewer возвращет все pr пользоавтель. где он ревьюер.
func (p *PullRequestRepository) GetByReviewer(ctx context.Context, reviewerID string) ([]storage.PullRequest, error) {
	const query = `
		SELECT DISTINCT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
        FROM pull_requests pr
        INNER JOIN reviews r ON r.pull_request_id = pr.pull_request_id
        WHERE r.reviewer_id = $1 
	`
	rows, err := p.pool.Query(ctx, query, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	defer rows.Close()

	var prs []storage.PullRequest
	for rows.Next() {
		var pr storage.PullRequest
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

// IsReviewerAssigned проверяет, является ли пользователь ревьюером.
func (p *PullRequestRepository) IsReviewerAssigned(ctx context.Context, reviewerID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM reviews WHERE reviewer_id = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, query, reviewerID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	return exists, nil
}
