package application

import (
	"context"
	"time"

	"exp/clean_architecture/domain"
)

type CreateTaskInput struct {
	Title string
}

type CreateTask struct {
	repository domain.TaskRepository
	now        func() time.Time
}

func NewCreateTask(repository domain.TaskRepository) CreateTask {
	return CreateTask{repository: repository, now: time.Now}
}

func (useCase CreateTask) Execute(ctx context.Context, input CreateTaskInput) (domain.Task, error) {
	task, err := domain.NewTask(input.Title, useCase.now().UTC())
	if err != nil {
		return domain.Task{}, err
	}

	return useCase.repository.Create(ctx, task)
}
