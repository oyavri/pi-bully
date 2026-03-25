package task

import (
	"time"

	"github.com/google/uuid"
)

type State string

const (
	StateQueued     State = "QUEUED"
	StateAssigned   State = "ASSIGNED"
	StateRunning    State = "RUNNING"
	StateFailed     State = "FAILED"
	StateWorkerLost State = "WORKER_LOST"
	StateCompleted  State = "COMPLETED"
)

type Task struct {
	ID         uuid.UUID
	Executable string
	InputURI   string
	OutputURI  string
	Args       []string
	MaxRetries int
	CreatedAt  time.Time
}
