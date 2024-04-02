package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type Library struct {
	Books []Book `json:"books"`
}

func (l *Library) AddBook(b Book) {
	l.Books = append(l.Books, b)
}

func (l *Library) RemoveBook(id int) {
	for i, b := range l.Books {
		if b.ID == id {
			l.Books = append(l.Books[:i], l.Books[i+1:]...)
			break
		}
	}
}

func (l *Library) GetBook(id int) *Book {
	for _, b := range l.Books {
		if b.ID == id {
			return &b
		}
	}
	return nil
}

func (l *Library) GetBooks() []Book {
	return l.Books
}

func (l *Library) UpdateBook(id int, b Book) *Book {
	for i, book := range l.Books {
		if book.ID == id {
			// check which fields are set and update only those
			if b.Title != "" {
				l.Books[i].Title = b.Title
			}
			if b.Author != "" {
				l.Books[i].Author = b.Author
			}
			return &l.Books[i]
		}
	}
	return nil
}

func (l *Library) Reset() {
	l.Books = []Book{}
}

func NewLibrary() *Library {
	return &Library{}
}

func NewBook(id int, title, author string) Book {
	return Book{
		ID:     id,
		Title:  title,
		Author: author,
	}
}

// REST API to manage a library of books using the Go standard library
//
// This is a simple REST API to manage a library of books. It uses the Go standard library to create a simple HTTP server that listens on port 8080. The API supports the following operations:
//
// - GET /books: Get all books in the library
// - GET /books/{id}: Get a book by ID
// - POST /books: Add a new book to the library
// - PUT /books/{id}: Update a book by ID
// - DELETE /books/{id}: Delete a book by ID
//

type BooksAPI struct {
	Library *Library
	router  *mux.Router
}

func (api *BooksAPI) GetBooks(w http.ResponseWriter, r *http.Request) {
	books := api.Library.GetBooks()
	respondWithJSON(w, http.StatusOK, books)
}

func (api *BooksAPI) GetBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := parseInt(vars["id"])
	book := api.Library.GetBook(id)
	if book == nil {
		respondWithError(w, http.StatusNotFound, "Book not found")
		return
	}
	respondWithJSON(w, http.StatusOK, book)
}

func (api *BooksAPI) AddBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	if err := decodeJSON(r.Body, &book); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	api.Library.AddBook(book)
	respondWithJSON(w, http.StatusCreated, book)
}

func (api *BooksAPI) UpdateBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := parseInt(vars["id"])
	var book Book
	if err := decodeJSON(r.Body, &book); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	newBook := api.Library.UpdateBook(id, book)
	if newBook == nil {
		respondWithError(w, http.StatusNotFound, "Book not found")
		return
	}
	respondWithJSON(w, http.StatusOK, newBook)
}

func (api *BooksAPI) DeleteBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := parseInt(vars["id"])
	api.Library.RemoveBook(id)
	respondWithJSON(w, http.StatusNoContent, map[string]string{"result": "success"})
}

func (api *BooksAPI) ResetLibrary(w http.ResponseWriter, r *http.Request) {
	api.Library.Reset()
	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (api *BooksAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}

func NewBooksAPI() *BooksAPI {
	api := &BooksAPI{
		Library: NewLibrary(),
	}
	r := mux.NewRouter()
	r.HandleFunc("/books", api.GetBooks).Methods("GET")
	r.HandleFunc("/books/{id}", api.GetBook).Methods("GET")
	r.HandleFunc("/books", api.AddBook).Methods("POST")
	r.HandleFunc("/books/{id}", api.UpdateBook).Methods("PUT")
	r.HandleFunc("/books/{id}", api.DeleteBook).Methods("DELETE")
	r.HandleFunc("/books/reset", api.ResetLibrary).Methods("POST")
	api.router = r
	return api
}

// Helper functions

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := encodeJSON(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encodeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Main function
func main() {
	fmt.Println("Starting server on :8080")
	api := NewBooksAPI()
	http.ListenAndServe(":8080", api)
}
