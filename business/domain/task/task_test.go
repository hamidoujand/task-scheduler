package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/domain/task/store/memory"
)

func TestCreateTask(t *testing.T) {
	store := memory.Repository{
		Tasks: make(map[uuid.UUID]task.Task),
	}

	service := task.NewService(&store)

	//create task
	tsk, err := service.CreateTask(context.Background(), task.NewTask{
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		ScheduledAt: time.Now().Add(time.Hour * 2),
	})
	if err != nil {
		t.Fatalf("expected the task to be saved: %s", err)
	}

	if tsk.Status != "pending" {
		t.Errorf("expected status to be %q, but got %q", "pending", tsk.Status)
	}

	if tsk.CreatedAt.IsZero() || tsk.UpdatedAt.IsZero() {
		t.Errorf("expected createdAt and updatedAt field to not be zero time values")
	}
}

func TestGetTaskById(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	store := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			id: {
				Id:          id,
				Command:     "docker",
				Args:        []string{"ps"},
				Status:      "success",
				Result:      "data",
				ErrMessage:  "",
				ScheduledAt: now.Add(time.Hour * 2),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	service := task.NewService(&store)

	tsk, err := service.GetTaskById(context.Background(), id)
	if err != nil {
		t.Fatalf("should be able to find the task by id: %s", err)
	}

	if tsk.Command != "docker" {
		t.Errorf("expected the command to be %s but got %s", "docker", tsk.Command)
	}

	if len(tsk.Args) == 0 {
		t.Errorf("expected task to have args")
	}

	if tsk.Args[0] != "ps" {
		t.Errorf("expected the first arg to be %q, but got %q", "ps", tsk.Args[0])
	}

	newId := uuid.New()
	_, err = service.GetTaskById(context.Background(), newId)
	if err == nil {
		t.Fatal("expected to not find a task with random id")
	}

	if !errors.Is(err, task.ErrTaskNotFound) {
		t.Errorf("expected the error to be %v, but got %v", task.ErrTaskNotFound, err)
	}
}

func TestDeleteTask(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	store := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			id: {
				Id:          id,
				Command:     "docker",
				Args:        []string{"ps"},
				Status:      "success",
				Result:      "data",
				ErrMessage:  "",
				ScheduledAt: now.Add(time.Hour * 2),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	service := task.NewService(&store)

	tsk, err := service.GetTaskById(context.Background(), id)
	if err != nil {
		t.Fatalf("should be able to find the task by id: %s", err)
	}
	err = service.DeleteTask(context.Background(), tsk)
	if err != nil {
		t.Fatalf("should be able to delete the task: %s", err)
	}

	_, err = service.GetTaskById(context.Background(), tsk.Id)
	if err == nil {
		t.Fatal("expected to not find a task after deletion")
	}

	if !errors.Is(err, task.ErrTaskNotFound) {
		t.Errorf("expected the error to be %v, but got %v", task.ErrTaskNotFound, err)
	}
}

func TestUpdateTask(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	store := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			id: {
				Id:          id,
				Command:     "docker",
				Args:        []string{"ps"},
				Status:      "pending",
				ScheduledAt: now.Add(time.Hour * 2),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	service := task.NewService(&store)

	tsk, err := service.GetTaskById(context.Background(), id)
	if err != nil {
		t.Fatalf("should be able to find the task by id: %s", err)
	}

	status := "completed"
	result := "data"

	ut := task.UpdateTask{
		Status: &status,
		Result: &result,
	}

	tsk, err = service.UpdateTask(context.Background(), tsk, ut)
	if err != nil {
		t.Fatalf("should be able to update the task: %s", err)
	}

	if tsk.Result != result {
		t.Errorf("expected result to be %q but got %q", result, tsk.Result)
	}
	if tsk.Status != status {
		t.Errorf("expected status to be %q but got %q", status, tsk.Status)
	}
}
