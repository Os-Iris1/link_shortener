package postgres_test

import (
	"context"
	"errors"
	"testing"

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
			if tt.dbErr != nil {
				expect.WillReturnError(tt.dbErr)
			} else {
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

func TestLinkRepositoryInitialize(t *testing.T) {
	failure := errors.New("DDL failed")
	for _, tt := range []struct {
		name  string
		dbErr error
	}{
		{"success", nil}, {"failure", failure},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			expect := db.ExpectExec("CREATE TABLE IF NOT EXISTS links")
			if tt.dbErr != nil {
				expect.WillReturnError(tt.dbErr)
			} else {
				expect.WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
			}
			err = postgres.NewLinkRepository(db).Initialize(context.Background())
			if !errors.Is(err, tt.dbErr) {
				t.Fatalf("error = %v, want %v", err, tt.dbErr)
			}
			if err := db.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
