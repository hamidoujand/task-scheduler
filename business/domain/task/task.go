package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// store represents the decoupled store to interact with.
type store interface {
	Create(ctx context.Context, task Task) error
	Update(ctx context.Context, task Task) error
	Delete(ctx context.Context, task Task) error
	GetById(ctx context.Context, taskId uuid.UUID) (Task, error)
}

// Service represents set of APIs for accessing tasks.
type Service struct {
	store store
}

// NewService creates *Service and returns it.
func NewService(store store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateTask(ctx context.Context, nt NewTask) (Task, error) {

	now := time.Now()

	task := Task{
		Id:          uuid.New(),
		Command:     nt.Command,
		Args:        nt.Args,
		Status:      "pending",
		ScheduledAt: nt.ScheduledAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := s.store.Create(ctx, task)
	if err != nil {
		return Task{}, fmt.Errorf("task creation: %w", err)
	}
	return task, nil
}

func (s *Service) GetTaskById(ctx context.Context, taskId uuid.UUID) (Task, error) {
	task, err := s.store.GetById(ctx, taskId)
	if err != nil {
		//some issue or not found
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}

func (s *Service) DeleteTask(ctx context.Context, task Task) error {
	if err := s.store.Delete(ctx, task); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func (s *Service) UpdateTask(ctx context.Context, task Task, ut UpdateTask) (Task, error) {

	if ut.Status != nil {
		task.Status = *ut.Status
	}

	if ut.ErrMessage != nil {
		task.ErrMessage = *ut.ErrMessage
	}

	if ut.Result != nil {
		task.Result = *ut.Result
	}

	task.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, task); err != nil {
		return Task{}, fmt.Errorf("updating task: %w", err)
	}

	return task, nil
}
