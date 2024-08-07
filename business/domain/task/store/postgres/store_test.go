package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
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
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
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
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
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

	if tsk.Image != tt.Image {
		t.Errorf("image=%s, got %s", tt.Image, tsk.Image)
	}

	if tsk.Environment != tt.Environment {
		t.Errorf("environment= %s, got %s", tt.Environment, tsk.Environment)
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
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
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
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
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
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
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
	repo := postgresRepo.NewRepository(client)

	userId, commands := seedTasks(t, repo)

	tests := map[string]struct {
		page           int
		rows           int
		order          task.OrderBy
		expectedResult int
	}{
		"all user's tasks": {
			page: 1,
			rows: len(commands),
			order: task.OrderBy{
				Field:     task.FieldCreatedAt,
				Direction: task.DirectionDESC,
			},
			expectedResult: len(commands),
		},

		"page 1 only 4 tasks": {
			page: 1,
			rows: 4,
			order: task.OrderBy{
				Field:     task.FieldCreatedAt,
				Direction: task.DirectionDESC,
			},
			expectedResult: 4,
		},

		"ordered by command": {
			page: 1,
			rows: len(commands),
			order: task.OrderBy{
				Field:     task.FieldCommand,
				Direction: task.DirectionDESC,
			},
			expectedResult: len(commands),
		},

		"ordered by command page 1 rows 2": {
			page: 1,
			rows: 2,
			order: task.OrderBy{
				Field:     task.FieldCommand,
				Direction: task.DirectionDESC,
			},
			expectedResult: 2,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			userTasks, err := repo.GetByUserId(context.Background(), userId, test.rows, test.page, test.order)
			if err != nil {
				t.Fatalf("expect to fetch all tasks for user %q: %s", userId, err)
			}

			if len(userTasks) != test.expectedResult {
				t.Fatalf("len(tasks)=%d, got %d", len(commands), len(userTasks))
			}
			if name == "all" {
				for _, tsk := range userTasks {
					if !slices.Contains(commands, tsk.Command) {
						t.Errorf("expect command %q to be in commands slice", tsk.Command)
					}
				}
			}

			if name == "ordered by command" {
				cmds := make([]string, len(commands))
				copy(cmds, commands)
				slices.Sort(cmds)
				slices.Reverse(cmds)

				for i, tsk := range userTasks {
					if tsk.Command != cmds[i] {
						t.Errorf("expected the data to be in order: got[%s] want [%s]", tsk.Command, cmds[i])
					}
				}
			}

			if name == "ordered by command page 1 rows 2" {
				cmds := make([]string, len(commands))
				copy(cmds, commands)
				slices.Sort(cmds)
				slices.Reverse(cmds)
				cmds = cmds[:2]

				for i, tsk := range userTasks {
					if tsk.Command != cmds[i] {
						t.Errorf("expected the data to be in order: got[%s] want [%s]", tsk.Command, cmds[i])
					}
				}
			}
		})
	}
}

func seedTasks(t *testing.T, repo *postgresRepo.Repository) (uuid.UUID, []string) {
	userId := uuid.New()
	commands := []string{"ls", "wc", "ab", "bc", "ps", "cc", "dd", "date"}

	status := []task.Status{task.StatusCompleted, task.StatusFailed, task.StatusPending}

	for i, command := range commands {
		createdAt := time.Now().Add(time.Hour * time.Duration(i))
		scheduledAt := time.Now().AddDate(0, 0, i)
		idxStatus := rand.Intn(len(status))
		tsk := task.Task{
			Id:          uuid.New(),
			UserId:      userId,
			Command:     command,
			Image:       "alpine:3.20",
			Environment: "APP_NAME=test",
			Status:      status[idxStatus],
			ScheduledAt: scheduledAt,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		}

		if tsk.Command == "ls" {
			tsk.Args = []string{"-l", "-a"}
		}

		if tsk.Status == task.StatusCompleted {
			tsk.Result = fmt.Sprintf("data%d", i)
		}

		if tsk.Status == task.StatusFailed {
			tsk.ErrMessage = fmt.Sprintf("error%d", i)
		}

		err := repo.Create(context.Background(), tsk)
		if err != nil {
			t.Fatalf("expected to create task: %s", err)
		}
	}
	return userId, commands
}
