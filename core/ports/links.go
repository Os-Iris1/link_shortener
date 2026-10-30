package ports

import (
	"context"

	"shorter/core/models"
)

//go:generate go tool mockgen -source=links.go -destination=mocks/links.go -package=mocks

type LinkRepository interface {
	Save(context.Context, models.Link) error
	FindByCode(context.Context, string) (models.Link, error)
}

type CodeGenerator interface {
	Generate() (string, error)
}

type LinkShortener interface {
	Shorten(context.Context, string) (string, error)
}

type LinkResolver interface {
	Resolve(context.Context, string) (string, error)
}
