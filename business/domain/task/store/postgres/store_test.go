package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/dbtest"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	postgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	client := dbtest.NewDatabaseClient(t, "test_task_create")
	store := postgresRepo.NewRepository(client)

	//insert a new task
	id := uuid.New()
	userId := uuid.New()
	now := time.Now()
	tt := task.Task{
		Id:          id,
		UserId:      userId,
		Command:     "ls",
		Args:        []string{"-1", "-a"},
		Status:      task.StatusPending,
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.Create(context.Background(), tt)
	if err != nil {
		t.Fatalf("creating task: %s", err)
	}
}

func TestGetById(t *testing.T) {
	t.Parallel()

	client := dbtest.NewDatabaseClient(t, "test_task_getById")
	store := postgresRepo.NewRepository(client)

	//insert a new task
	id := uuid.New()
	userId := uuid.New()
	now := time.Now()
	tt := task.Task{
		Id:          id,
		UserId:      userId,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusPending,
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.Create(context.Background(), tt)
	if err != nil {
		t.Fatalf("creating task: %s", err)
	}

	//get by id
	tsk, err := store.GetById(context.Background(), id)
	if err != nil {
		t.Fatalf("should return task by id %s", id)
	}
	if tsk.Command != tt.Command {
		t.Errorf("expected command to be %q, got %q", tt.Command, tsk.Command)
	}
	if len(tsk.Args) != len(tt.Args) {
		t.Errorf("expected args length to b %d but got %d", len(tt.Args), len(tsk.Args))
	}

	expectedArgs := []string{"-a", "-l"}
	for _, arg := range expectedArgs {
		if !slices.Contains(tsk.Args, arg) {
			t.Errorf("expected arg %q to be in args slice", arg)
		}
	}

	diffTime := tsk.ScheduledAt.Sub(now)

	if diffTime >= time.Hour*10 {
		t.Errorf("expected diff between scheduledAt and now to be less or equal to 10 but got %s", diffTime)
	}

	if tsk.UserId != userId {
		t.Errorf("expected userId to be %s, got %s", userId, tsk.UserId)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	client := dbtest.NewDatabaseClient(t, "test_task_update")
	store := postgresRepo.NewRepository(client)

	//insert a new task
	id := uuid.New()
	userId := uuid.New()

	now := time.Now()
	tt := task.Task{
		Id:          id,
		UserId:      userId,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusPending,
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.Create(context.Background(), tt)
	if err != nil {
		t.Fatalf("creating task: %s", err)
	}

	tu := task.Task{
		Id:          id,
		UserId:      userId,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusCompleted,
		Result:      "data",
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now.Add(time.Hour),
	}

	err = store.Update(context.Background(), tu)
	if err != nil {
		t.Fatalf("should be able to update a task: %s", err)
	}
	// get the task by id
	updated, err := store.GetById(context.Background(), id)
	if err != nil {
		t.Fatalf("should return task by id after update %s", id)
	}

	if updated.Status != tu.Status {
		t.Errorf("expected the Status to be %s, but got %s", tu.Status, updated.Status)
	}

	if updated.Result != tu.Result {
		t.Errorf("expected the Result to be %s, but got %s", tu.Result, updated.Result)
	}

	if updated.ErrMessage != tu.ErrMessage {
		t.Errorf("expected the ErrMessage to be %s, but got %s", tu.ErrMessage, updated.ErrMessage)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	client := dbtest.NewDatabaseClient(t, "test_task_delete")
	store := postgresRepo.NewRepository(client)

	//insert a new task
	id := uuid.New()
	userId := uuid.New()
	now := time.Now()
	tt := task.Task{
		Id:          id,
		UserId:      userId,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusPending,
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.Create(context.Background(), tt)
	if err != nil {
		t.Fatalf("creating task: %s", err)
	}

	//delete

	if err := store.Delete(context.Background(), tt); err != nil {
		t.Fatalf("should be able to delete a task: %s", err)
	}

	_, err = store.GetById(context.Background(), id)
	if err == nil {
		t.Fatalf("should get an error when querying a deleted task")
	}

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected error to be %v but got %v", sql.ErrNoRows, err)
	}
}
