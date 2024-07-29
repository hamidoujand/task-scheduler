// Package memory provides a in memory repository used for testing.
package memory

import (
	"context"
	"database/sql"
	"sync"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

type Repository struct {
	Users map[uuid.UUID]user.User
	mu    sync.Mutex
}

// Create adds a new user into the repo and return possible error.
func (r *Repository) Create(ctx context.Context, usr user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Users[usr.Id] = usr
	return nil
}

// GetById queries the repo for user with id and returns sql.ErrNoRows if there is not user.
func (r *Repository) GetById(ctx context.Context, id uuid.UUID) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if usr, ok := r.Users[id]; ok {
		return usr, nil
	} else {
		return user.User{}, sql.ErrNoRows
	}
}

// Delete deletes the user from repo and return possible error.
func (r *Repository) Delete(ctx context.Context, usr user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Users, usr.Id)
	return nil
}

// Update replaces the user with the update one and return possible error.
func (r *Repository) Update(ctx context.Context, usr user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Users[usr.Id] = usr
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, usr := range r.Users {
		if usr.Email.Address == email {
			return usr, nil
		}
	}
	return user.User{}, sql.ErrNoRows
}
