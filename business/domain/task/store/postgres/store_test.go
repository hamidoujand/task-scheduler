package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/dbtest"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	postgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
)

func Test_Task(t *testing.T) {
	client := dbtest.NewDatabaseClient(t)
	store := postgresRepo.NewRepository(client)

	//insert a new task
	id := uuid.New()
	now := time.Now()
	tt := task.Task{
		Id:          id,
		Command:     "ls",
		Args:        []string{"-1", "-a"},
		Status:      "pending",
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.Create(context.Background(), tt)
	if err != nil {
		t.Fatalf("creating task: %s", err)
	}

	//get the task by id
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

	diffTime := tsk.ScheduledAt.Sub(now)

	if diffTime >= time.Hour*10 {
		t.Errorf("expected diff between scheduledAt and now to be less or equal to 10 but got %s", diffTime)
	}

	//update
	tsk.Result = "data"
	tsk.ErrMessage = "no-err"
	tsk.Status = "success"

	err = store.Update(context.Background(), tsk)
	if err != nil {
		t.Fatalf("should be able to update a task: %s", err)
	}
	// get the task by id
	tsk, err = store.GetById(context.Background(), id)
	if err != nil {
		t.Fatalf("should return task by id after update %s", id)
	}

	if tsk.Status != "success" {
		t.Errorf("expected the Status to be success, but got %s", tsk.Status)
	}

	if tsk.Result != "data" {
		t.Errorf("expected the Result to be data, but got %s", tsk.Result)
	}

	if tsk.ErrMessage != "no-err" {
		t.Errorf("expected the ErrMessage to be np-err, but got %s", tsk.ErrMessage)
	}

	//delete

	if err := store.Delete(context.Background(), tsk); err != nil {
		t.Fatalf("should be able to delete a task: %s", err)
	}

	_, err = store.GetById(context.Background(), id)
	if err == nil {
		t.Fatalf("should get an error when querying a deleted task")
	}
}
