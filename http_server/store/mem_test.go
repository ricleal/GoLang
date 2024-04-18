package store_test

import (
	"testing"

	"exp/http_server/repository"
	"exp/http_server/store"
)

func TestExecTx(t *testing.T) {
	s := store.NewMemStore()

	authors := s.Authors()
	books := s.Books()

	aID, err := authors.Create(&repository.Author{
		Name: "George Orwell",
	})
	if err != nil {
		t.Fatalf("failed to create author: %v", err)
	}

	bID, err := books.Create(&repository.Book{
		Title: "1984",
	})
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	al, err := authors.List()
	if err != nil {
		t.Fatalf("failed to list authors: %v", err)
	}
	if len(al) != 1 {
		t.Fatalf("expected 1 author, got %d", len(al))
	}

	bl, err := books.List()
	if err != nil {
		t.Fatalf("failed to list books: %v", err)
	}
	if len(bl) != 1 {
		t.Fatalf("expected 1 book, got %d", len(bl))
	}

	if err := s.ExecTx(nil, func(scopedStore store.Store) error {
		authors := scopedStore.Authors()
		books := scopedStore.Books()

		author, err := authors.Read(aID)
		if err != nil {
			return err
		}

		book, err := books.Read(bID)
		if err != nil {
			return err
		}

		book.Author = author

		if err := books.Update(book); err != nil {
			return err
		}

		// Make sure the book was updated.
		book, err = books.Read(bID)
		if err != nil {
			return err
		}
		if book.Author.ID != author.ID {
			t.Error("expected author to be updated")
		}

		return nil
	}); err != nil {
		t.Fatalf("failed to execute transaction: %v", err)
	}
}
