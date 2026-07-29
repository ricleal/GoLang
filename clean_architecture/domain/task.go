package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyTaskTitle = errors.New("task title cannot be empty")
	ErrTaskNotFound   = errors.New("task not found")
)

// Task is the core business entity for the application.
type Task struct {
	ID        int64
	Title     string
	CreatedAt time.Time
}

// NewTask enforces the enterprise rule that every task must have a title.
func NewTask(title string, createdAt time.Time) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrEmptyTaskTitle
	}

	return Task{Title: title, CreatedAt: createdAt}, nil
}
