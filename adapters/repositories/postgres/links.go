package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	cerrors "shorter/core/errors"
	"shorter/core/models"
)

type Executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type LinkRepository struct {
	db Executor
}

func NewLinkRepository(db Executor) *LinkRepository {
	return &LinkRepository{db: db}
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
