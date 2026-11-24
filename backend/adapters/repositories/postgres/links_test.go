package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"shorter/adapters/repositories/postgres"
	cerrors "shorter/core/errors"
	"shorter/core/models"
)

func TestLinkRepositorySave(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, tt := range []struct {
		name           string
		rows           int64
		dbErr, wantErr error
	}{
		{name: "insert", rows: 1},
		{name: "collision", rows: 0, wantErr: cerrors.ErrCodeConflict},
		{name: "database error", dbErr: failure, wantErr: failure},
		{name: "context canceled", dbErr: context.Canceled, wantErr: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			expect := db.ExpectExec("INSERT INTO links (code, original_url) VALUES ($1, $2) ON CONFLICT (code) DO NOTHING").WithArgs("code", "https://example.com/'?q=$1")
			switch {
			case tt.dbErr != nil:
				expect.WillReturnError(tt.dbErr)
			default:
				expect.WillReturnResult(pgxmock.NewResult("INSERT", tt.rows))
			}
			err = postgres.NewLinkRepository(db).Save(context.Background(), models.Link{Code: "code", URL: "https://example.com/'?q=$1"})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if err := db.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLinkRepositoryFindByCode(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, tt := range []struct {
		name           string
		dbErr, wantErr error
		badRow         bool
	}{
		{name: "found"},
		{name: "missing", dbErr: pgx.ErrNoRows, wantErr: cerrors.ErrLinkNotFound},
		{name: "database error", dbErr: failure, wantErr: failure},
		{name: "canceled", dbErr: context.Canceled, wantErr: context.Canceled},
		{name: "deadline", dbErr: context.DeadlineExceeded, wantErr: context.DeadlineExceeded},
		{name: "scan error", badRow: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			code := "abc' OR 1=1"
			expect := db.ExpectQuery("SELECT original_url FROM links WHERE code = $1").WithArgs(code)
			switch {
			case tt.dbErr != nil:
				expect.WillReturnError(tt.dbErr)
			case tt.badRow:
				expect.WillReturnRows(pgxmock.NewRows([]string{"original_url"}).AddRow("https://example.com").RowError(0, failure))
			default:
				expect.WillReturnRows(pgxmock.NewRows([]string{"original_url"}).AddRow("https://example.com/?q=1#part"))
			}
			got, err := postgres.NewLinkRepository(db).FindByCode(context.Background(), code)
			if tt.badRow {
				if err == nil {
					t.Fatal("expected scan error")
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			want := models.Link{}
			if tt.dbErr == nil && !tt.badRow {
				want = models.Link{Code: code, URL: "https://example.com/?q=1#part"}
			}
			if got != want {
				t.Fatalf("link = %+v, want %+v", got, want)
			}
			if err := db.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
