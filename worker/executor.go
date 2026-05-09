package worker

import (
	"context"
)

type Executor interface {
	Execute(ctx context.Context, a Assignment) error
}

type TaskExecutor struct{}

func NewTaskExecutor() *TaskExecutor {
	return &TaskExecutor{}
}

func (e *TaskExecutor) Execute(ctx context.Context, a Assignment) error {
	return nil
}
