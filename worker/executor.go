package worker

import (
	"context"
)

type Executor interface {
	Execute(ctx context.Context, a Assignment) error
}
