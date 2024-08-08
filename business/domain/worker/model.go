package worker

import "github.com/google/uuid"

// Worker represents a worker used to execute commands.
type Worker struct {
	ID     uuid.UUID
	Status WorkerStatus
	Load   int
}
