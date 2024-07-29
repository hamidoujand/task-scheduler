package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

// Repository represents set of APIs used to interact with postgres.
type Repository struct {
	client *postgres.Client
}

// NewRepository provides APIs to interact with store.
func NewRepository(pgClient *postgres.Client) *Repository {
	return &Repository{
		client: pgClient,
	}
}

// Create create a user into database and returns possible error.
func (r *Repository) Create(ctx context.Context, usr user.User) error {
	const q = `
	INSERT INTO users 
		(id,name,email,roles,password_hash,enabled,created_at,updated_at)
	VALUES
		($1,$2,$3,$4,$5,$6,$7,$8)	
	`
	pgUser := ToPostgresUser(usr)

	_, err := r.client.DB.ExecContext(ctx, q,
		pgUser.Id,
		pgUser.Name,
		pgUser.Email,
		pgUser.Roles,
		pgUser.PasswordHash,
		pgUser.Enabled,
		pgUser.CreatedAt,
		pgUser.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("exec context: %w", err)
	}
	return nil
}

func (r *Repository) GetById(ctx context.Context, id uuid.UUID) (user.User, error) {
	const q = `
	SELECT 
		id,name,email,array_to_json(roles),password_hash,enabled,created_at,updated_at
	FROM users
	WHERE id = $1	
	`

	row := r.client.DB.QueryRowContext(ctx, q, id.String())
	var roles any
	var usr User

	err := row.Scan(
		&usr.Id,
		&usr.Name,
		&usr.Email,
		&roles,
		&usr.PasswordHash,
		&usr.Enabled,
		&usr.CreatedAt,
		&usr.UpdatedAt,
	)
	if err != nil {
		return user.User{}, fmt.Errorf("scanning row: %w", err)
	}

	switch t := roles.(type) {
	case []byte:
		if err := json.Unmarshal(t, &usr.Roles); err != nil {
			return user.User{}, fmt.Errorf("unmarshalling roles: %w", err)
		}
	default:
		return user.User{}, fmt.Errorf("roles scanning: %T", roles)
	}
	return usr.ToServiceUser(), nil
}

func (r *Repository) Update(ctx context.Context, usr user.User) error {
	pgUser := ToPostgresUser(usr)

	const q = `
	UPDATE 
		users 
	SET
		name = $1,
		email = $2,
		roles = $3,
		password_hash = $4,
		enabled = $5,
		updated_at = $6
	WHERE id = $7	
	`
	if _, err := r.client.DB.ExecContext(ctx, q,
		pgUser.Name,
		pgUser.Email,
		pgUser.Roles,
		pgUser.PasswordHash,
		pgUser.Enabled,
		pgUser.UpdatedAt,
		pgUser.Id,
	); err != nil {
		return fmt.Errorf("execContext: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, usr user.User) error {
	const q = `
		DELETE FROM users WHERE id = $1
	`
	if _, err := r.client.DB.ExecContext(ctx, q, usr.Id.String()); err != nil {
		return fmt.Errorf("execContext: %w", err)
	}
	return nil
}
