package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool: pool,
	}
}

func (s *Store) CreateEmotion(
	ctx context.Context,
	userID int,
	emotion string,
	confidence float32,
	capturedAt time.Time,
) error {

	_, err := s.pool.Exec(
		ctx,
		`
		INSERT INTO emotion_logs (
			user_id,
			emotion,
			confidence,
			captured_at
		)
		VALUES ($1, $2, $3, $4)
		`,
		userID,
		emotion,
		confidence,
		capturedAt,
	)

	return err
}
