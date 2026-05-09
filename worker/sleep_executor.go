package worker

import (
	"context"
	"time"
)

type SleepExecutor struct {
	Duration time.Duration
}

func NewSleepExecutor(duration time.Duration) *SleepExecutor {
	return &SleepExecutor{Duration: duration}
}

func (e *SleepExecutor) Execute(ctx context.Context, a Assignment) error {
	timer := time.NewTimer(e.Duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
