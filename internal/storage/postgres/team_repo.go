// Package postgres предоставляет подключение к PostgreSQL через pqxpool.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
func (t *TeamRepository) Create(ctx context.Context, team storage.Team) error {
	const queryTeamInsert = `INSERT INTO teams (team_name) VALUES ($1) RETURNING id, created_at`
	const queryUserInsert = `
		INSERT INTO users (user_id, username, team_id, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			team_id = EXCLUDED.team_id,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()`
	exists, err := t.Exists(ctx, team.TeamName)
	if err != nil {
		return fmt.Errorf("check exists failed: %w", err)
	}

	if exists {
		return errors.New("team already exists")
	}

	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
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
		return fmt.Errorf("insert team failed: %w", err)
	}

	for _, user := range team.Members {
		_, err := tx.Exec(ctx, queryUserInsert, user.ID, user.Username, teamID, user.IsActive)
		if err != nil {
			return fmt.Errorf("failed insertion into users: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

// Get осуществляет поиск в бд команды и её участников по имени команды.
func (t *TeamRepository) Get(ctx context.Context, teamName string) (storage.Team, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return team, errors.New("team not found")
		}
		return team, fmt.Errorf("query team failed: %w", err)
	}

	rows, err := t.pool.Query(ctx, selectUsersByTeamID, team.ID)
	if err != nil {
		return team, fmt.Errorf("query users failed: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var user storage.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt); err != nil {
			return team, fmt.Errorf("scan user failed: %w", err)
		}
		team.Members = append(team.Members, user)
	}

	return team, rows.Err()
}

// Exists проверяет существует ли команда по её имени.
func (t *TeamRepository) Exists(ctx context.Context, teamName string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	var exists bool
	err := t.pool.QueryRow(ctx, query, teamName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	return exists, nil
}
