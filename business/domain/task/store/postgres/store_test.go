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
		Command:     "date",
		Args:        nil,
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

	if tsk.Args != nil {
		t.Errorf("expected the args to be nil, got %v", tsk.Args)
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
		Command:     "ps",
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
		t.Fatalf("should return task by id after update %s", err)
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

func TestGetByUserId(t *testing.T) {
	t.Parallel()

	client := dbtest.NewDatabaseClient(t, "test_task_getByUserId")
	store := postgresRepo.NewRepository(client)

	//seed
	userId, commands := seedRandomTasks(t, store)

	usersTasks, err := store.GetByUserId(context.Background(), userId)
	if err != nil {
		t.Fatalf("expected to get all tasks created by user %q: %s", userId, err)
	}

	if len(usersTasks) != 4 {
		t.Errorf("expected the length of usersTasks to be %d, got %d", len(commands), len(usersTasks))
	}

	for _, task := range usersTasks {
		if !slices.Contains(commands, task.Command) {
			t.Errorf("%s, not exists inside of commands slice", task.Command)
		}

		if task.UserId != userId {
			t.Errorf("expected the user id for all tasks to be %s, got %s", userId, task.UserId)
		}

		if task.Command == "ls" {
			//check for parsed args
			args := []string{"-l", "-a"}
			for _, arg := range task.Args {
				if !slices.Contains(args, arg) {
					t.Errorf("expected %s arg to be in args slice", arg)
				}
			}
		} else {
			//no args should be parsed to nil
			if task.Args != nil {
				t.Errorf("args should be nil for command %s, got %v", task.Command, task.Args)
			}
		}
	}

}

func seedRandomTasks(t *testing.T, s *postgresRepo.Repository) (uuid.UUID, []string) {
	userId := uuid.New()
	commands := []string{"ls", "date", "ps", "top"}
	now := time.Now()

	for _, c := range commands {
		task := task.Task{
			Id:          uuid.New(),
			UserId:      userId,
			Command:     c,
			Status:      task.StatusCompleted,
			Result:      "data",
			CreatedAt:   now,
			ScheduledAt: now,
			UpdatedAt:   now,
			ErrMessage:  "",
		}

		if c == "ls" {
			task.Args = []string{"-l", "-a"}
		}

		err := s.Create(context.Background(), task)
		if err != nil {
			t.Fatalf("expected to seed database for getByUserId: %s", err)
		}
	}
	return userId, commands
}
