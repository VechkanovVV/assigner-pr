package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
func (u *UserRepository) Get(ctx context.Context, userID string) (storage.User, error) {
	const query = `
		SELECT user_id, username, team_id, is_active, updated_at
        FROM users WHERE user_id = $1
	`

	var user storage.User
	err := u.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, errors.New("user not found")
		}

		return user, fmt.Errorf("query failed: %w", err)
	}
	return user, nil
}

// SetActive обновляет флаг активности пользователя.
func (u *UserRepository) SetActive(ctx context.Context, userID string, isActive bool) (storage.User, error) {
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
			return user, errors.New("user not found")
		}

		return user, fmt.Errorf("set active failed: %w", err)
	}

	return user, nil
}

// GetActiveTeammates возвращает активных участников команды, помимо пользователя с excludedID.
func (u *UserRepository) GetActiveTeammates(ctx context.Context, teamName, excludedID string) ([]storage.User, error) {
	const selectTeamIDQuery = `
		SELECT id FROM teams WHERE team_name = $1
	`
	const selectActiveTeammatesQuery = `
		SELECT user_id, username, team_id, is_active, updated_at
		FROM users
		WHERE team_id = $1 AND is_active = true AND user_id != $2
	`

	var teamID int
	err := u.pool.QueryRow(ctx, selectTeamIDQuery, teamName).Scan(&teamID)
	if err != nil {
		return nil, fmt.Errorf("get team id failed: %w", err)
	}

	rows, err := u.pool.Query(ctx, selectActiveTeammatesQuery, teamID, excludedID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	defer rows.Close()

	var users []storage.User
	for rows.Next() {
		var user storage.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamID, &user.IsActive, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Exists проверяет существует ли пользователь по его ID(userID).
func (u *UserRepository) Exists(ctx context.Context, userID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`
	var exists bool
	err := u.pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	return exists, nil
}
