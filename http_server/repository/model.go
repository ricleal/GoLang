package repository

import "errors"

// Author is a person who writes books.
type Author struct {
	ID   string
	Name string
}

// Book is a written work.
type Book struct {
	ID     string
	Title  string
	Author *Author
}

var ErrNotFound = errors.New("not found")
