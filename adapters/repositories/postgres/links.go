package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	cerrors "shorter/core/errors"
	"shorter/core/models"
)

//go:embed schema.sql
var schema string

type Executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type LinkRepository struct {
	db Executor
}

func NewLinkRepository(db Executor) *LinkRepository {
	return &LinkRepository{db: db}
}

// Initialize создаёт начальную схему MVP, включая ранее созданные тома Compose.
func (r *LinkRepository) Initialize(ctx context.Context) error {
	if _, err := r.db.Exec(ctx, schema); err != nil {
		return fmt.Errorf("initialize links schema: %w", err)
	}
	return nil
}

func (r *LinkRepository) Save(ctx context.Context, link models.Link) error {
	tag, err := r.db.Exec(ctx,
		"INSERT INTO links (code, original_url) VALUES ($1, $2) ON CONFLICT (code) DO NOTHING",
		link.Code, link.URL)
	if err != nil {
		return fmt.Errorf("insert link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return cerrors.ErrCodeConflict
	}
	return nil
}
