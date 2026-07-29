package domain

import "context"

// TaskRepository is implemented by outer-layer persistence mechanisms.
type TaskRepository interface {
	Create(ctx context.Context, task Task) (Task, error)
	GetByID(ctx context.Context, id int64) (Task, error)
}
