package store

import (
	"context"
	"fmt"
	"sync"

	"exp/http_server/repository"
	"exp/http_server/repository/mem"
)

type memStore struct {
	mu      sync.RWMutex
	storage *mem.MemStorage
}

func NewMemStore() *memStore {
	return &memStore{
		storage: mem.NewMemStorage(),
	}
}

func (s *memStore) Authors() repository.AuthorRepo {
	return &mem.AuthorMemStore{
		MemStorage: *s.storage,
	}
}

func (s *memStore) Books() repository.BookRepo {
	return &mem.BookMemStore{
		MemStorage: *s.storage,
	}
}

// ExecTx executes the given function within a database transaction.
func (s *memStore) ExecTx(ctx context.Context, fn func(Store) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := fn(s)
	if err != nil {
		return fmt.Errorf("ExecTx: %w", err)
	}
	return nil
}
