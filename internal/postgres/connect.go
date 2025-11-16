package postgres

import (
	"context"
	logger "github.com/quasttyy/pr-reviewer/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func InitPool(ctx context.Context, dsn string, minConns, maxConns int32) *pgxpool.Pool {
	logger.Info("initializing postgres connection...")

	// Парсим DSN
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("failed to parse DSN:", err)
	}

	// Настройка пула соединений
	cfg.MinConns = minConns
	cfg.MaxConns = maxConns
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	// Создаём пул
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.Fatal("failed to create postgres pool:", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("postgres ping failed:", err)
	}

	logger.Info("postgres connected successfully")

	return pool
}
