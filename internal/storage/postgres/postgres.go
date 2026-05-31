package postgres

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(dbURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	log.Println("Connected to PostgreSQL")

	return pool, nil
}
