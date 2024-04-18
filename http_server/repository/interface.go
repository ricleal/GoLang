package repository

// CRUD operations for authors.
type AuthorRepo interface {
	// Create a new author.
	Create(author *Author) (string, error)
	// Read a author by ID.
	Read(id string) (*Author, error)
	// Update a author.
	Update(author *Author) error
	// Delete a author by ID.
	Delete(id string) error
	// List all authors.
	List() ([]*Author, error)
}

// CRUD operations for books.
type BookRepo interface {
	// Create a new book.
	Create(book *Book) (string, error)
	// Read a book by ID.
	Read(id string) (*Book, error)
	// Update a book.
	Update(book *Book) error
	// Delete a book by ID.
	Delete(id string) error
	// List all books.
	List() ([]*Book, error)
}
