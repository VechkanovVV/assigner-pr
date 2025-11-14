// Package postgres предоставляет подключение к PostgreSQL через pqxpool.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// PoolMaxConns - максимальное кол-во соединения в pool.
	PoolMaxConns = int32(25)
	// PoolMinConns - минимальное кол-во поддерживаемых соединений в pool.
	PoolMinConns = int32(5)
	// PoolMaxConnLifetime - максимальное время жизни соединения в pool.
	PoolMaxConnLifetime = 30 * time.Minute
	// PoolMaxConnIdleTime - максимальное время простоя соединения.
	PoolMaxConnIdleTime = 5 * time.Minute
	// PoolHealthCheckPeriod - переодичность проверки соединения.
	PoolHealthCheckPeriod = 1 * time.Minute
)

// NewPool создаёт новый pool соединений к PostgreSQL
func NewPool(ctx context.Context, port int, host, user, password, dbName, sslmode string) (*pgxpool.Pool, error) {
	conf := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", user, password, host, port, dbName, sslmode)

	cfg, err := pgxpool.ParseConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("parsing config failed: %w", err)
	}

	cfg.MaxConns = PoolMaxConns
	cfg.MinConns = PoolMinConns
	cfg.MaxConnLifetime = PoolMaxConnLifetime
	cfg.MaxConnIdleTime = PoolMaxConnIdleTime
	cfg.HealthCheckPeriod = PoolHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool failed: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pool ping failed: %w", err)
	}

	return pool, nil
}
