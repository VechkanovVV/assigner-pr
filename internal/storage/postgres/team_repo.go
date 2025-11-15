// Package postgres предоставляет подключение к PostgreSQL через pqxpool.
package postgres

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
	"github.com/VechkanovVV/assigner-pr/internal/storage"
)

// TeamRepository - репозиторий, для управлением командами в Postgres.
type TeamRepository struct {
	pool *pgxpool.Pool
}

// NewTeamRepository создаёт экземпляр *TeamRepository.
func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

// Create создаёт новую команду.
func (t *TeamRepository) Create(ctx context.Context, team storage.Team) *apperrors.AppError {
	const queryTeamInsert = `INSERT INTO teams (team_name) VALUES ($1) RETURNING id, created_at`
	const queryUserInsert = `
        INSERT INTO users (user_id, username, team_id, is_active)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (user_id) DO UPDATE SET
            username = EXCLUDED.username,
            team_id = EXCLUDED.team_id,
            is_active = EXCLUDED.is_active,
            updated_at = NOW()`

	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("begin tx failed: %v", err)
		return &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}

	defer func() {
		if rerr := tx.Rollback(ctx); rerr != nil && !errors.Is(rerr, pgx.ErrTxClosed) {
			log.Printf("tx rollback error: %v", rerr)
		}
	}()

	var teamID int
	var createdAt time.Time
	err = tx.QueryRow(ctx, queryTeamInsert, team.TeamName).Scan(&teamID, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &apperrors.AppError{
				Code:    apperrors.ErrTeamExists,
				Message: apperrors.FromCode(apperrors.ErrTeamExists),
			}
		}
		log.Printf("insert team failed: %v", err)
		return &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}

	for _, user := range team.Members {
		_, err := tx.Exec(ctx, queryUserInsert, user.ID, user.Username, teamID, user.IsActive)
		if err != nil {
			log.Printf("failed insertion into users: %v", err)
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

// Get осуществляет поиск в бд команды и её участников по имени команды.
func (t *TeamRepository) Get(ctx context.Context, teamName string) (storage.Team, *apperrors.AppError) {
	const selectTeamByName = `
	SELECT t.id, t.team_name, t.created_at
	FROM teams t
	WHERE t.team_name = $1
	`

	const selectUsersByTeamID = `
		SELECT u.user_id, u.username, u.team_id, u.is_active, u.updated_at
		FROM users u
		WHERE u.team_id = $1
	`
	var team storage.Team
	err := t.pool.QueryRow(ctx, selectTeamByName, teamName).Scan(&team.ID, &team.TeamName, &team.CreatedAt)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.Is(err, pgx.ErrNoRows) {
			appErr = &apperrors.AppError{
				Code:    apperrors.ErrNotFound,
				Message: apperrors.FromCode(apperrors.ErrNotFound),
			}
			return storage.Team{}, appErr
		}
		log.Printf("query team failed: %v", err)
		appErr = &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return team, appErr
	}

	rows, err := t.pool.Query(ctx, selectUsersByTeamID, team.ID)
	if err != nil {
		log.Printf("query users failed: %v", err)
		appErr := &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
		return team, appErr
	}

	defer rows.Close()

	for rows.Next() {
		var user storage.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt); err != nil {
			log.Printf("scan user failed: %v", err)
			appErr := &apperrors.AppError{
				Code:    apperrors.ErrInternalIssue,
				Message: apperrors.FromCode(apperrors.ErrInternalIssue),
			}
			return team, appErr
		}
		team.Members = append(team.Members, user)
	}

	if err := rows.Err(); err != nil {
		log.Printf("%v", err)
		return storage.Team{}, &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}

	return team, nil
}

// Exists проверяет существует ли команда по её имени.
func (t *TeamRepository) Exists(ctx context.Context, teamName string) (bool, *apperrors.AppError) {
	const query = `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	var exists bool
	err := t.pool.QueryRow(ctx, query, teamName).Scan(&exists)
	if err != nil {
		log.Printf("query failed: %v", err)
		return false, &apperrors.AppError{
			Code:    apperrors.ErrInternalIssue,
			Message: apperrors.FromCode(apperrors.ErrInternalIssue),
		}
	}

	return exists, nil
}
