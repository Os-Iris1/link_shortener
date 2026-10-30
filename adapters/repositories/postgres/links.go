package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	cerrors "shorter/core/errors"
	"shorter/core/models"
)

type Executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
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

func (r *LinkRepository) FindByCode(ctx context.Context, code string) (models.Link, error) {
	link := models.Link{Code: code}
	err := r.db.QueryRow(ctx, "SELECT original_url FROM links WHERE code = $1", code).Scan(&link.URL)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Link{}, cerrors.ErrLinkNotFound
	}
	if err != nil {
		return models.Link{}, fmt.Errorf("select link: %w", err)
	}
	return link, nil
}
