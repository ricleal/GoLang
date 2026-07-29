package application

import (
	"context"
	"errors"
	"testing"

	"exp/clean_architecture/domain"
)

type recordingTaskRepository struct {
	created bool
}

func (repository *recordingTaskRepository) Create(_ context.Context, task domain.Task) (domain.Task, error) {
	repository.created = true
	task.ID = 1
	return task, nil
}

func (repository *recordingTaskRepository) GetByID(_ context.Context, _ int64) (domain.Task, error) {
	return domain.Task{}, errors.New("not implemented")
}

func TestCreateTaskRejectsBlankTitleBeforePersistence(t *testing.T) {
	repository := &recordingTaskRepository{}
	useCase := NewCreateTask(repository)

	_, err := useCase.Execute(context.Background(), CreateTaskInput{Title: "   "})
	if !errors.Is(err, domain.ErrEmptyTaskTitle) {
		t.Fatalf("expected ErrEmptyTaskTitle, got %v", err)
	}
	if repository.created {
		t.Fatal("repository should not be called for an invalid task")
	}
}
