package store

import (
	"context"

	"exp/http_server/repository"
)

// Store is the interface that wraps the repositories.
type Store interface {
	Authors() repository.AuthorRepo
	Books() repository.BookRepo
	ExecTx(ctx context.Context, fn func(Store) error) error
}
