package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestAPI() *BooksAPI {
	api := NewBooksAPI()
	// Add test books
	for i := 1; i <= 25; i++ {
		book := NewBook(i, "Book "+string(rune('A'+i-1)), "Author "+string(rune('A'+i-1)))
		api.Library.AddBook(book)
	}
	return api
}

func TestGetBooks_DefaultPagination(t *testing.T) {
	api := setupTestAPI()

	req, _ := http.NewRequest("GET", "/books", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check default pagination values
	if response.Page != 1 {
		t.Errorf("Expected page 1, got %d", response.Page)
	}
	if response.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", response.Limit)
	}
	if response.Total != 25 {
		t.Errorf("Expected total 25, got %d", response.Total)
	}
	if response.TotalPages != 3 {
		t.Errorf("Expected 3 total pages, got %d", response.TotalPages)
	}
	if len(response.Books) != 10 {
		t.Errorf("Expected 10 books, got %d", len(response.Books))
	}
}

func TestGetBooks_CustomPagination(t *testing.T) {
	api := setupTestAPI()

	req, _ := http.NewRequest("GET", "/books?page=2&limit=5", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Page != 2 {
		t.Errorf("Expected page 2, got %d", response.Page)
	}
	if response.Limit != 5 {
		t.Errorf("Expected limit 5, got %d", response.Limit)
	}
	if response.Total != 25 {
		t.Errorf("Expected total 25, got %d", response.Total)
	}
	if response.TotalPages != 5 {
		t.Errorf("Expected 5 total pages, got %d", response.TotalPages)
	}
	if len(response.Books) != 5 {
		t.Errorf("Expected 5 books, got %d", len(response.Books))
	}

	// Check that we got the right books (IDs 6-10)
	if response.Books[0].ID != 6 {
		t.Errorf("Expected first book ID to be 6, got %d", response.Books[0].ID)
	}
}

func TestGetBooks_LastPage(t *testing.T) {
	api := setupTestAPI()

	req, _ := http.NewRequest("GET", "/books?page=3&limit=10", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Last page should have only 5 books (25 total, 10 per page)
	if len(response.Books) != 5 {
		t.Errorf("Expected 5 books on last page, got %d", len(response.Books))
	}

	// Check that we got the right books (IDs 21-25)
	if response.Books[0].ID != 21 {
		t.Errorf("Expected first book ID to be 21, got %d", response.Books[0].ID)
	}
	if response.Books[len(response.Books)-1].ID != 25 {
		t.Errorf("Expected last book ID to be 25, got %d", response.Books[len(response.Books)-1].ID)
	}
}

func TestGetBooks_OutOfBoundsPage(t *testing.T) {
	api := setupTestAPI()

	req, _ := http.NewRequest("GET", "/books?page=10&limit=10", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return empty array for out of bounds page
	if len(response.Books) != 0 {
		t.Errorf("Expected 0 books for out of bounds page, got %d", len(response.Books))
	}

	if response.Total != 25 {
		t.Errorf("Expected total 25, got %d", response.Total)
	}
}

func TestGetBooks_InvalidParameters(t *testing.T) {
	api := setupTestAPI()

	tests := []struct {
		name     string
		query    string
		wantPage int
		wantLim  int
	}{
		{
			name:     "negative page defaults to 1",
			query:    "?page=-1&limit=5",
			wantPage: 1,
			wantLim:  5,
		},
		{
			name:     "zero page defaults to 1",
			query:    "?page=0&limit=5",
			wantPage: 1,
			wantLim:  5,
		},
		{
			name:     "negative limit defaults to 10",
			query:    "?page=1&limit=-5",
			wantPage: 1,
			wantLim:  10,
		},
		{
			name:     "zero limit defaults to 10",
			query:    "?page=1&limit=0",
			wantPage: 1,
			wantLim:  10,
		},
		{
			name:     "invalid string parameters default",
			query:    "?page=abc&limit=xyz",
			wantPage: 1,
			wantLim:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/books"+tt.query, nil)
			rr := httptest.NewRecorder()

			api.GetBooks(rr, req)

			var response PaginatedResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response.Page != tt.wantPage {
				t.Errorf("Expected page %d, got %d", tt.wantPage, response.Page)
			}
			if response.Limit != tt.wantLim {
				t.Errorf("Expected limit %d, got %d", tt.wantLim, response.Limit)
			}
		})
	}
}

func TestGetBooks_EmptyLibrary(t *testing.T) {
	api := NewBooksAPI()

	req, _ := http.NewRequest("GET", "/books", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Books) != 0 {
		t.Errorf("Expected 0 books, got %d", len(response.Books))
	}
	if response.Total != 0 {
		t.Errorf("Expected total 0, got %d", response.Total)
	}
	if response.TotalPages != 0 {
		t.Errorf("Expected 0 total pages, got %d", response.TotalPages)
	}
}

func TestGetBooks_LargeLimit(t *testing.T) {
	api := setupTestAPI()

	req, _ := http.NewRequest("GET", "/books?page=1&limit=100", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return all 25 books when limit exceeds total
	if len(response.Books) != 25 {
		t.Errorf("Expected 25 books, got %d", len(response.Books))
	}
	if response.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", response.Limit)
	}
	if response.TotalPages != 1 {
		t.Errorf("Expected 1 total page, got %d", response.TotalPages)
	}
}

func TestGetBooks_SingleBook(t *testing.T) {
	api := NewBooksAPI()
	api.Library.AddBook(NewBook(1, "Only Book", "Only Author"))

	req, _ := http.NewRequest("GET", "/books", nil)
	rr := httptest.NewRecorder()

	api.GetBooks(rr, req)

	var response PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Books) != 1 {
		t.Errorf("Expected 1 book, got %d", len(response.Books))
	}
	if response.Total != 1 {
		t.Errorf("Expected total 1, got %d", response.Total)
	}
	if response.TotalPages != 1 {
		t.Errorf("Expected 1 total page, got %d", response.TotalPages)
	}
}
