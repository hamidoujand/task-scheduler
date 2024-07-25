// memory provides an in memory repository used for testing.
package memory

import (
	"context"
	"database/sql"
	"sync"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

// Repository represent an in-memory storage for testing.
type Repository struct {
	Tasks map[uuid.UUID]task.Task
	mu    sync.Mutex
}

// Create is going to add a new task into repo or return error.
func (ms *Repository) Create(ctx context.Context, task task.Task) error {
	ms.mu.Lock()
	ms.Tasks[task.Id] = task
	ms.mu.Unlock()
	return nil
}

// Update is going to update a task inside repo or return error.
func (ms *Repository) Update(ctx context.Context, task task.Task) error {
	ms.mu.Lock()
	ms.Tasks[task.Id] = task
	ms.mu.Unlock()
	return nil
}

// Delete is going to delete a task in repo or return error.
func (ms *Repository) Delete(ctx context.Context, task task.Task) error {
	ms.mu.Lock()
	delete(ms.Tasks, task.Id)
	ms.mu.Unlock()
	return nil
}

// GetById is going to get a task by id or return error "sql.ErrNoRows".
func (ms *Repository) GetById(ctx context.Context, taskId uuid.UUID) (task.Task, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if tsk, ok := ms.Tasks[taskId]; !ok {
		return task.Task{}, sql.ErrNoRows
	} else {
		return tsk, nil
	}
}
