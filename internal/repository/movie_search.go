package repository

import (
	"context"

	"homework/internal/model"
)

// MovieSearcher is an additive catalog-search capability for movie repositories.
type MovieSearcher interface {
	Search(ctx context.Context, q, genre string, offset, limit int) ([]model.Movie, error)
	Genres(ctx context.Context) ([]string, error)
}
