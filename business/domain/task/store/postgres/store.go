package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

var columnNames = map[task.Field]string{
	task.FieldCommand:     "command",
	task.FieldCreatedAt:   "created_at",
	task.FieldScheduledAt: "scheduled_at",
	task.FieldID:          "id",
}

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

	args, err := parseArgs(commandArgs)

	if err != nil {
		return task.Task{}, fmt.Errorf("parseArgs: %w", err)
	}

	dbTask.Args = args

	return dbTask.toDomainTask(), nil
}

func (r *Repository) GetByUserId(ctx context.Context, userId uuid.UUID, rowsPerPage int, pageNumber int, order task.OrderBy) ([]task.Task, error) {
	offset := (pageNumber - 1) * rowsPerPage

	col, exist := columnNames[order.Field]
	if !exist {
		return nil, fmt.Errorf("invalid column name %q", order.Field)
	}

	// for direction we have enum "ASC" and "DESC" and for column names also we use "Enum like" code so no risk of sql
	// injection in here.

	q := fmt.Sprintf(`
	SELECT
		id,user_id,command,array_to_json(args) as args,status,result,error_msg,scheduled_at,created_at,updated_at
	FROM tasks
	WHERE user_id = $1
	ORDER BY %s %s OFFSET $2 ROWS FETCH NEXT $3 ROWS ONLY	
	`, col, order.Direction.String())

	rows, err := r.client.DB.QueryContext(ctx, q, userId, offset, rowsPerPage)
	if err != nil {
		return nil, fmt.Errorf("queryContext: %w", err)
	}

	var results []task.Task
	for rows.Next() {
		var dbTask Task
		var commandArgs any
		err := rows.Scan(
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
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		args, err := parseArgs(commandArgs)

		if err != nil {
			return nil, fmt.Errorf("parseArgs: %w", err)
		}

		dbTask.Args = args
		results = append(results, dbTask.toDomainTask())
	}

	return results, nil
}

func parseArgs(raw any) (sql.Null[[]string], error) {
	var args sql.Null[[]string]

	switch val := raw.(type) {
	case []byte:
		// var args []string
		if err := json.Unmarshal(val, &args.V); err != nil {
			return sql.Null[[]string]{}, fmt.Errorf("unmarshalling args: %w", err)
		}

	case nil:
		return sql.Null[[]string]{
			V:     nil,
			Valid: true,
		}, nil

	default:
		return sql.Null[[]string]{}, fmt.Errorf("args scan: %T", args)
	}

	//valid
	args.Valid = args.V != nil

	return args, nil
}
