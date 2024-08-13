// memory provides an in memory repository used for testing.
package memory

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

// Repository represent an in-memory storage for testing.
type Repository struct {
	Tasks map[uuid.UUID]task.Task
	mu    sync.Mutex
}

// Create is going to add a new task into repo or return error.
func (r *Repository) Create(ctx context.Context, task task.Task) error {
	r.mu.Lock()
	r.Tasks[task.Id] = task
	r.mu.Unlock()
	return nil
}

// Update is going to update a task inside repo or return error.
func (r *Repository) Update(ctx context.Context, task task.Task) error {
	r.mu.Lock()
	r.Tasks[task.Id] = task
	r.mu.Unlock()
	return nil
}

// Delete is going to delete a task in repo or return error.
func (r *Repository) Delete(ctx context.Context, task task.Task) error {
	r.mu.Lock()
	delete(r.Tasks, task.Id)
	r.mu.Unlock()
	return nil
}

// GetById is going to get a task by id or return error "sql.ErrNoRows".
func (r *Repository) GetById(ctx context.Context, taskId uuid.UUID) (task.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tsk, ok := r.Tasks[taskId]; !ok {
		return task.Task{}, sql.ErrNoRows
	} else {
		return tsk, nil
	}
}

func (r *Repository) GetByUserId(ctx context.Context, userId uuid.UUID, rows int, page int, order task.OrderBy) ([]task.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var results []task.Task

	for _, task := range r.Tasks {
		if task.UserId == userId {
			results = append(results, task)
		}
	}

	return results, nil
}

// GetDueTasks returns all tasks that has 1 min to execute.
func (r *Repository) GetDueTasks(ctx context.Context, from time.Time) ([]task.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var results []task.Task
	for _, tsk := range r.Tasks {
		diff := tsk.ScheduledAt.Sub(from)
		if diff <= time.Minute {
			results = append(results, tsk)
		}
	}
	return results, nil
}
