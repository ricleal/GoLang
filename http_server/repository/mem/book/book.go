package book

import (
	"time"

	"github.com/google/uuid"

	"exp/http_server/repository/mem/author"
)

type Book struct {
	ID        uuid.UUID
	Title     string
	Author    *author.Author
	CreatedAt time.Time
}

func NewBook(title string, a *author.Author) *Book {
	return &Book{
		ID:        uuid.New(),
		Title:     title,
		Author:    a,
		CreatedAt: time.Now(),
	}
}
