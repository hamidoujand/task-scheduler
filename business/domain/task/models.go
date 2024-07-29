package task

import (
	"time"

	"github.com/google/uuid"
)

// Task represents a task inside the systems.
type Task struct {
	Id          uuid.UUID
	UserId      uuid.UUID
	Command     string
	Args        []string
	Status      Status
	Result      string
	ErrMessage  string
	ScheduledAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask represents all of the required info for creating a new task.
type NewTask struct {
	UserId      uuid.UUID
	Command     string
	Args        []string
	ScheduledAt time.Time
}

// UpdateTask represents all of the data that can be update about a task.
type UpdateTask struct {
	Status     *Status
	Result     *string
	ErrMessage *string
}
