package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"exp/clean_architecture/domain"
)

var _ domain.TaskRepository = (*TaskRepository)(nil)

type TaskRepository struct {
	database *sql.DB
}

func NewTaskRepository(database *sql.DB) (*TaskRepository, error) {
	repository := &TaskRepository{database: database}
	if _, err := database.Exec("CREATE SEQUENCE IF NOT EXISTS tasks_id_sequence START 1"); err != nil {
		return nil, fmt.Errorf("create task ID sequence: %w", err)
	}
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id BIGINT PRIMARY KEY DEFAULT nextval('tasks_id_sequence'),
			title VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`); err != nil {
		return nil, fmt.Errorf("create tasks table: %w", err)
	}
	return repository, nil
}

func (repository *TaskRepository) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := repository.database.QueryRowContext(ctx,
		"INSERT INTO tasks (title, created_at) VALUES (?, ?) RETURNING id", task.Title, task.CreatedAt)
	if err := row.Scan(&task.ID); err != nil {
		return domain.Task{}, fmt.Errorf("insert task: %w", err)
	}
	return task, nil
}

func (repository *TaskRepository) GetByID(ctx context.Context, id int64) (domain.Task, error) {
	var task domain.Task
	var createdAt time.Time
	err := repository.database.QueryRowContext(ctx,
		"SELECT id, title, created_at FROM tasks WHERE id = ?", id).Scan(&task.ID, &task.Title, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("select task: %w", err)
	}
	task.CreatedAt = createdAt.UTC()
	return task, nil
}
