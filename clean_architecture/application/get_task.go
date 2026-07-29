package application

import (
	"context"

	"exp/clean_architecture/domain"
)

type GetTask struct {
	repository domain.TaskRepository
}

func NewGetTask(repository domain.TaskRepository) GetTask {
	return GetTask{repository: repository}
}

func (useCase GetTask) Execute(ctx context.Context, id int64) (domain.Task, error) {
	return useCase.repository.GetByID(ctx, id)
}
