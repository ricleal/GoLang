package cliadapter

import (
	"context"
	"fmt"
	"io"

	"exp/clean_architecture/application"
)

type TaskPresenter struct {
	createTask application.CreateTask
	getTask    application.GetTask
	output     io.Writer
}

func NewTaskPresenter(createTask application.CreateTask, getTask application.GetTask, output io.Writer) TaskPresenter {
	return TaskPresenter{createTask: createTask, getTask: getTask, output: output}
}

func (presenter TaskPresenter) Create(ctx context.Context, title string) error {
	task, err := presenter.createTask.Execute(ctx, application.CreateTaskInput{Title: title})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(presenter.output, "created task %d: %s\n", task.ID, task.Title)
	return err
}

func (presenter TaskPresenter) Get(ctx context.Context, id int64) error {
	task, err := presenter.getTask.Execute(ctx, id)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(presenter.output, "task %d: %s\n", task.ID, task.Title)
	return err
}
