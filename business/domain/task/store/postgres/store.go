package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

// Store represents apis used to interact with database.
type Repository struct {
	client *postgres.Client
}

// New creates a new store that uses *postgres.Client as its db client
func NewRepository(clint *postgres.Client) *Repository {
	return &Repository{
		client: clint,
	}
}

func (s *Repository) Create(ctx context.Context, task task.Task) error {
	const q = `
	INSERT INTO tasks
		(id,user_id,command,args,status,result,error_msg,scheduled_at,created_at,updated_at)
	VALUES
		($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);
	`

	dbTask := toDBTask(task)
	_, err := s.client.DB.ExecContext(ctx, q,
		dbTask.Id,
		dbTask.UserId,
		dbTask.Command,
		dbTask.Args,
		dbTask.Status,
		dbTask.Result,
		dbTask.ErrorMessage,
		dbTask.ScheduledAt,
		dbTask.CreatedAt,
		dbTask.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("exec context: %w", err)
	}
	return nil
}

func (s *Repository) Update(ctx context.Context, task task.Task) error {
	const q = `
	UPDATE 
		tasks
	SET
		status =    $1,
		result =    $2,
		error_msg = $3
	WHERE
		id = $4		
	`
	dbTask := toDBTask(task)

	_, err := s.client.DB.ExecContext(ctx, q,
		dbTask.Status,
		dbTask.Result,
		dbTask.ErrorMessage,
		dbTask.Id,
	)
	if err != nil {
		return fmt.Errorf("exec context: %w", err)
	}

	return nil
}
func (s *Repository) Delete(ctx context.Context, task task.Task) error {
	const q = `
	DELETE FROM
		tasks
	WHERE 
		id = $1	
	
	`
	_, err := s.client.DB.ExecContext(ctx, q, task.Id)
	if err != nil {
		return fmt.Errorf("exec context: %w", err)
	}
	return nil
}
func (s *Repository) GetById(ctx context.Context, taskId uuid.UUID) (task.Task, error) {
	var dbTask Task
	const q = `
	SELECT 
		id,user_id,command,array_to_json(args) as args,status,result,error_msg,scheduled_at,created_at,updated_at
	FROM 
		tasks
	WHERE 
		id = $1		
	`

	row := s.client.DB.QueryRowContext(ctx, q, taskId.String())

	var commandArgs any

	if err := row.Scan(
		&dbTask.Id,
		&dbTask.UserId,
		&dbTask.Command,
		&commandArgs,
		&dbTask.Status,
		&dbTask.Result,
		&dbTask.ErrorMessage,
		&dbTask.ScheduledAt,
		&dbTask.CreatedAt,
		&dbTask.UpdatedAt,
	); err != nil {
		return task.Task{}, fmt.Errorf("row scan: %w", err)
	}

	switch val := commandArgs.(type) {
	case []byte:
		if err := json.Unmarshal(val, &dbTask.Args); err != nil {
			return task.Task{}, fmt.Errorf("unmarshalling args: %w", err)
		}
	default:
		return task.Task{}, fmt.Errorf("args scan: %T", commandArgs)
	}

	return dbTask.toDomainTask(), nil
}
