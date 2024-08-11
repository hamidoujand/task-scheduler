package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
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
	GetByUserId(ctx context.Context, userId uuid.UUID, rows int, page int, order OrderBy) ([]Task, error)
}

// Service represents set of APIs for accessing tasks.
type Service struct {
	store   store
	rClient *rabbitmq.Client
}

// NewService creates *Service and returns it.
func NewService(store store, rClient *rabbitmq.Client) (*Service, error) {
	//register queue
	if err := rClient.DeclareQueue(queue); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &Service{
		store:   store,
		rClient: rClient,
	}, nil
}

func (s *Service) CreateTask(ctx context.Context, nt NewTask) (Task, error) {
	now := time.Now()

	task := Task{
		Id:          uuid.New(),
		UserId:      nt.UserId,
		Command:     nt.Command,
		Args:        nt.Args,
		Image:       nt.Image,
		Environment: nt.Environment,
		Status:      StatusPending,
		ScheduledAt: nt.ScheduledAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := s.store.Create(ctx, task)
	if err != nil {
		return Task{}, fmt.Errorf("task creation: %w", err)
	}

	//now check deadline, less than 1 min will be enqueued into rabbitmq
	difference := task.ScheduledAt.Sub(now)
	if difference < time.Minute {
		bs, err := json.Marshal(task)
		if err != nil {
			return Task{}, fmt.Errorf("marshal: %w", err)
		}

		if err := publish(s.rClient, bs); err != nil {
			return Task{}, fmt.Errorf("publish: %w", err)
		}
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

// GetTaskByUserId queries all of the taks belong to a user and retunrs them or possible error.
func (s *Service) GetTasksByUserId(ctx context.Context, userId uuid.UUID, rowsPerPage int, page int, order OrderBy) ([]Task, error) {
	tasks, err := s.store.GetByUserId(ctx, userId, rowsPerPage, page, order)
	if err != nil {
		return nil, fmt.Errorf("getByUserId: %w", err)
	}
	return tasks, nil
}
