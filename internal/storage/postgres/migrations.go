package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE EXTENSION IF NOT EXISTS pgcrypto;

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255),
		email VARCHAR(255),
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS emotion_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id INT,
		emotion VARCHAR(50),
		confidence FLOAT,
		captured_at TIMESTAMP
	);
	`

	_, err := pool.Exec(ctx, query)
	return err
}
