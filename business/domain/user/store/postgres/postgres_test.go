package postgres_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/dbtest"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/business/domain/user/store/postgres"
	"golang.org/x/crypto/bcrypt"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	pgClient := dbtest.NewDatabaseClient(t, "test_create_user")
	repo := postgres.NewRepository(pgClient)
	now := time.Now()
	id := uuid.New()

	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("should be able to generate hash: %s", err)
	}

	usr := user.User{
		Id:   id,
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: pass,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = repo.Create(context.Background(), usr)
	if err != nil {
		t.Fatalf("should be able to create a user in db with valid data: %s", err)
	}
}

func TestGetById(t *testing.T) {
	t.Parallel()

	pgClient := dbtest.NewDatabaseClient(t, "test_getById_user")
	repo := postgres.NewRepository(pgClient)

	//insert one
	now := time.Now()
	id := uuid.New()
	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("should be able to generate hash: %s", err)
	}

	usr := user.User{
		Id:   id,
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: pass,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = repo.Create(context.Background(), usr)
	if err != nil {
		t.Fatalf("should be able to create a user in db with valid data: %s", err)
	}

	//fetch it
	got, err := repo.GetById(context.Background(), id)

	if err != nil {
		t.Fatalf("should be able to fetch user %q: %s", id, err)
	}

	if got.Name != usr.Name {
		t.Errorf("expected name to be %s, got %s", usr.Name, got.Name)
	}

	if got.Email.Address != usr.Email.Address {
		t.Errorf("expected email to be %s, got %s", usr.Email, got.Email.Address)
	}

	if !bytes.Equal(got.PasswordHash, usr.PasswordHash) {
		t.Errorf("expected passwords to be %q, got %q", usr.PasswordHash, got.PasswordHash)
	}

	if got.Roles[0] != user.RoleUser {
		t.Errorf("expected role to be %s, but got %s", user.RoleUser, got.Roles[0])
	}

}

func TestUpdate(t *testing.T) {
	t.Parallel()

	pgClient := dbtest.NewDatabaseClient(t, "test_update_user")
	repo := postgres.NewRepository(pgClient)

	//insert one
	now := time.Now()
	id := uuid.New()
	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("should be able to generate hash: %s", err)
	}

	usr := user.User{
		Id:   id,
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: pass,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = repo.Create(context.Background(), usr)
	if err != nil {
		t.Fatalf("should be able to create a user in db with valid data: %s", err)
	}

	updates := user.User{
		Id:   id,
		Name: "jane",
		Email: mail.Address{
			Name:    "jane",
			Address: "jane@gmail.com",
		},
		Roles:        []user.Role{user.RoleAdmin, user.RoleUser},
		PasswordHash: pass,
		Enabled:      false,
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now,
	}

	err = repo.Update(context.Background(), updates)
	if err != nil {
		t.Fatalf("expected the updates to apply: %s", err)
	}

	//fetch it
	got, err := repo.GetById(context.Background(), id)
	if err != nil {
		t.Fatalf("expected to be able fetch the updated user: %s", err)
	}

	if got.Name != updates.Name {
		t.Errorf("expected name to be %s, got %s", updates.Name, got.Name)
	}

	if got.Email.Address != updates.Email.Address {
		t.Errorf("expected email to be %s, got %s", updates.Email, got.Email.Address)
	}

	if !bytes.Equal(got.PasswordHash, updates.PasswordHash) {
		t.Errorf("expected passwords to be %q, got %q", updates.PasswordHash, got.PasswordHash)
	}

	if len(got.Roles) != 2 {
		t.Errorf("expected to have 2 roles now, got %d", len(got.Roles))
	}

	expectedRoles := []user.Role{user.RoleAdmin, user.RoleUser}
	for _, role := range expectedRoles {
		if !slices.Contains(got.Roles, role) {
			t.Errorf("expected role %s to be inside of user roles", role)
		}
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	pgClient := dbtest.NewDatabaseClient(t, "test_delete_user")
	repo := postgres.NewRepository(pgClient)

	//insert one
	now := time.Now()
	id := uuid.New()
	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("should be able to generate hash: %s", err)
	}

	usr := user.User{
		Id:   id,
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: pass,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = repo.Create(context.Background(), usr)
	if err != nil {
		t.Fatalf("should be able to create a user in db with valid data: %s", err)
	}

	//delete if
	if err := repo.Delete(context.Background(), usr); err != nil {
		t.Fatalf("should be able to delete user: %s", err)
	}

	//fetch it
	_, err = repo.GetById(context.Background(), usr.Id)
	if err == nil {
		t.Fatal("should not be able to find deleted user")
	}

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected error to be %v but got %v", sql.ErrNoRows, err)
	}
}

func TestGetByEmail(t *testing.T) {
	t.Parallel()

	pgClient := dbtest.NewDatabaseClient(t, "test_getByEmail_user")
	repo := postgres.NewRepository(pgClient)

	//insert one
	now := time.Now()
	id := uuid.New()
	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("should be able to generate hash: %s", err)
	}
	email := mail.Address{
		Name:    "john",
		Address: "john@gmail.com",
	}
	usr := user.User{
		Id:           id,
		Name:         "john",
		Email:        email,
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: pass,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = repo.Create(context.Background(), usr)
	if err != nil {
		t.Fatalf("should be able to create a user in db with valid data: %s", err)
	}

	//fetch it
	got, err := repo.GetByEmail(context.Background(), email.Address)

	if err != nil {
		t.Fatalf("should be able to fetch user by email: %s", err)
	}

	if got.Name != usr.Name {
		t.Errorf("expected name to be %s, got %s", usr.Name, got.Name)
	}

	if got.Email.Address != usr.Email.Address {
		t.Errorf("expected email to be %s, got %s", usr.Email, got.Email.Address)
	}

	if !bytes.Equal(got.PasswordHash, usr.PasswordHash) {
		t.Errorf("expected passwords to be %q, got %q", usr.PasswordHash, got.PasswordHash)
	}

	if got.Roles[0] != user.RoleUser {
		t.Errorf("expected role to be %s, but got %s", user.RoleUser, got.Roles[0])
	}

}
