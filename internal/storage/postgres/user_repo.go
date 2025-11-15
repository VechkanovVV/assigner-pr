package postgres

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
	"github.com/VechkanovVV/assigner-pr/internal/storage"
)

// UserRepository - репозиторий, для управлением пользователями(участниками команд) в Postgres.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создаёт экземпляр *UserRepository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Get осуществляет поиск в бд пользователя(участника команды) по его id.
func (u *UserRepository) Get(ctx context.Context, userID string) (storage.User, *apperrors.AppError) {
	const query = `
		SELECT user_id, username, team_id, is_active, updated_at
        FROM users WHERE user_id = $1
	`

	var user storage.User
	err := u.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			appErr := &apperrors.AppError{
				Code:    apperrors.ErrNotFound,
				Message: apperrors.FromCode(apperrors.ErrNotFound),
			}
			return user, appErr
		}

		log.Printf("query failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return user, appErr
	}
	return user, nil
}

// SetActive обновляет флаг активности пользователя.
func (u *UserRepository) SetActive(ctx context.Context, userID string, isActive bool) (storage.User, *apperrors.AppError) {
	const query = `
		UPDATE users
		SET is_active = $2, updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id, username, team_id, is_active, updated_at
	`

	var user storage.User
	err := u.pool.QueryRow(ctx, query, userID, isActive).Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			appErr := &apperrors.AppError{
				Code:    apperrors.ErrNotFound,
				Message: apperrors.FromCode(apperrors.ErrNotFound),
			}
			return user, appErr
		}
		log.Printf("set active failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return user, appErr
	}

	return user, nil
}

// GetActiveTeammates возвращает активных участников команды по teamID, исключая excludedID.
func (u *UserRepository) GetActiveTeammates(ctx context.Context, teamID int, excludedID string) ([]storage.User, *apperrors.AppError) {
	const query = `
		SELECT user_id, username, team_id, is_active, updated_at
		FROM users
		WHERE team_id = $1 AND is_active = true AND user_id != $2
	`

	rows, err := u.pool.Query(ctx, query, teamID, excludedID)
	if err != nil {
		log.Printf("query failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return nil, appErr
	}
	defer rows.Close()

	var users []storage.User
	for rows.Next() {
		var user storage.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt); err != nil {
			log.Printf("scan failed: %v", err)
			appErr := &apperrors.AppError{
				Code:    apperrors.ErrInternalIssue,
				Message: apperrors.FromCode(apperrors.ErrInternalIssue),
			}
			return nil, appErr
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.Printf("%v", err)
		return nil, &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}
	return users, nil
}

// Exists проверяет существует ли пользователь по его ID(userID).
func (u *UserRepository) Exists(ctx context.Context, userID string) (bool, *apperrors.AppError) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`
	var exists bool
	err := u.pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		log.Printf("query failed: %v", err)
		return false, &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}
	return exists, nil
}
