package mem

import (
	"exp/http_server/repository"
	"exp/http_server/repository/mem/author"
	"exp/http_server/repository/mem/book"
)

// MemStorage is an in-memory store for authors and books.
type MemStorage struct {
	authors map[string]*author.Author
	books   map[string]*book.Book
}

// NewMemStore creates a new MemStore.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		authors: make(map[string]*author.Author),
		books:   make(map[string]*book.Book),
	}
}

func (s *MemStorage) createAuthor(a *author.Author) (string, error) {
	s.authors[a.ID.String()] = a
	return a.ID.String(), nil
}

func (s *MemStorage) readAuthor(id string) (*author.Author, error) {
	a, ok := s.authors[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return a, nil
}

func (s *MemStorage) updateAuthor(a *author.Author) error {
	s.authors[a.ID.String()] = a
	return nil
}

func (s *MemStorage) deleteAuthor(id string) error {
	delete(s.authors, id)
	return nil
}

func (s *MemStorage) listAuthors() ([]*author.Author, error) {
	authors := make([]*author.Author, 0, len(s.authors))
	for _, a := range s.authors {
		authors = append(authors, a)
	}
	return authors, nil
}

func (s *MemStorage) createBook(b *book.Book) (string, error) {
	s.books[b.ID.String()] = b
	return b.ID.String(), nil
}

func (s *MemStorage) readBook(id string) (*book.Book, error) {
	b, ok := s.books[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return b, nil
}

func (s *MemStorage) updateBook(b *book.Book) error {
	s.books[b.ID.String()] = b
	return nil
}

func (s *MemStorage) deleteBook(id string) error {
	delete(s.books, id)
	return nil
}

func (s *MemStorage) listBooks() ([]*book.Book, error) {
	books := make([]*book.Book, 0, len(s.books))
	for _, b := range s.books {
		books = append(books, b)
	}
	return books, nil
}

// AuthorMemStore is an in-memory store for authors.
type AuthorMemStore struct {
	MemStorage
}

func (s *AuthorMemStore) Create(at *repository.Author) (string, error) {
	return s.createAuthor(author.NewAuthor(at.Name))
}

func (s *AuthorMemStore) Read(id string) (*repository.Author, error) {
	a, err := s.readAuthor(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, nil
	}
	return &repository.Author{
		ID:   a.ID.String(),
		Name: a.Name,
	}, nil
}

func (s *AuthorMemStore) Update(at *repository.Author) error {
	a, err := s.readAuthor(at.ID)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	a.Name = at.Name
	return s.updateAuthor(a)
}

func (s *AuthorMemStore) Delete(id string) error {
	return s.deleteAuthor(id)
}

func (s *AuthorMemStore) List() ([]*repository.Author, error) {
	authors, err := s.listAuthors()
	if err != nil {
		return nil, err
	}
	ats := make([]*repository.Author, 0, len(authors))
	for _, a := range authors {
		ats = append(ats, &repository.Author{
			ID:   a.ID.String(),
			Name: a.Name,
		})
	}
	return ats, nil
}

// BookMemStore is an in-memory store for books.
type BookMemStore struct {
	MemStorage
}

func (s *BookMemStore) Create(bk *repository.Book) (string, error) {
	if bk.Author == nil {
		return s.createBook(book.NewBook(bk.Title, nil))
	}
	a, err := s.readAuthor(bk.Author.ID)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", repository.ErrNotFound
	}
	return s.createBook(book.NewBook(bk.Title, a))
}

func (s *BookMemStore) Read(id string) (*repository.Book, error) {
	b, err := s.readBook(id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	if b.Author == nil {
		return &repository.Book{
			ID:    b.ID.String(),
			Title: b.Title,
		}, nil
	}
	a := s.authors[b.Author.ID.String()]
	return &repository.Book{
		ID:    b.ID.String(),
		Title: b.Title,
		Author: &repository.Author{
			ID:   a.ID.String(),
			Name: a.Name,
		},
	}, nil
}

func (s *BookMemStore) Update(bk *repository.Book) error {
	b, err := s.readBook(bk.ID)
	if err != nil {
		return err
	}
	if b == nil {
		return nil
	}
	a, err := s.readAuthor(bk.Author.ID)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	b.Title = bk.Title
	b.Author = a
	return s.updateBook(b)
}

func (s *BookMemStore) Delete(id string) error {
	return s.deleteBook(id)
}

func (s *BookMemStore) List() ([]*repository.Book, error) {
	books, err := s.listBooks()
	if err != nil {
		return nil, err
	}
	bks := make([]*repository.Book, 0, len(books))
	for _, b := range books {
		if b.Author == nil {
			bks = append(bks, &repository.Book{
				ID:    b.ID.String(),
				Title: b.Title,
			})
			continue
		}
		a := s.authors[b.Author.ID.String()]
		bks = append(bks, &repository.Book{
			ID:    b.ID.String(),
			Title: b.Title,
			Author: &repository.Author{
				ID:   a.ID.String(),
				Name: a.Name,
			},
		})
	}
	return bks, nil
}
