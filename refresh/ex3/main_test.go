package main

import (
	"os"
	"testing"
)

func TestScanner(t *testing.T) {
	// Create a temporary test file
	content := `line 1
line 2

line 3
line 4`

	tmpFile, err := os.CreateTemp("", "scanner_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create a new channel for this test (since Scanner uses the global one)
	lines = make(chan string, 10) // Buffered to avoid blocking

	// Run Scanner in a goroutine
	go Scanner(tmpFile.Name())

	// Collect all lines from the channel
	var receivedLines []string
	for line := range lines {
		receivedLines = append(receivedLines, line)
	}

	// Verify results
	expectedLines := []string{"line 1", "line 2", "line 3", "line 4"}
	if len(receivedLines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d", len(expectedLines), len(receivedLines))
	}

	for i, expected := range expectedLines {
		if i >= len(receivedLines) {
			t.Errorf("Missing line %d: expected %q", i, expected)
			continue
		}
		if receivedLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, receivedLines[i])
		}
	}
}
