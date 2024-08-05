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

	if tsk.Status != task.StatusPending {
		t.Errorf("expected status to be %q, but got %q", task.StatusPending, tsk.Status)
	}

	if tsk.CreatedAt.IsZero() || tsk.UpdatedAt.IsZero() {
		t.Errorf("expected createdAt and updatedAt field to not be zero time values")
	}
}

func TestGetTaskById(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	dt := task.Task{
		Id:          id,
		Command:     "docker",
		Args:        []string{"ps"},
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: now.Add(time.Hour * 2),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	store := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			id: dt,
		},
	}

	service := task.NewService(&store)

	tsk, err := service.GetTaskById(context.Background(), id)
	if err != nil {
		t.Fatalf("should be able to find the task by id: %s", err)
	}

	if tsk.Command != dt.Command {
		t.Errorf("expected the command to be %s but got %s", dt.Command, tsk.Command)
	}

	if len(tsk.Args) == 0 {
		t.Errorf("expected task to have args")
	}

	if tsk.Args[0] != dt.Args[0] {
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
				Status:      task.StatusCompleted,
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
				Status:      task.StatusPending,
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

	status := task.StatusCompleted
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

func TestGetTasksByUserId(t *testing.T) {
	userId := uuid.New()

	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	store := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			id1: {
				Id:      id1,
				UserId:  userId,
				Command: "ls",
				Status:  task.StatusCompleted,
			},
			id2: {
				Id:      id2,
				UserId:  userId,
				Command: "date",
				Status:  task.StatusPending,
			},
			id3: {
				Id:      id3,
				UserId:  userId,
				Command: "ps",
				Status:  task.StatusFailed,
			},
		},
	}
	service := task.NewService(&store)

	page := 1
	rows := 3
	order := task.OrderBy{
		Field:     task.FieldCommand,
		Direction: task.DirectionDESC,
	}

	tasks, err := service.GetTasksByUserId(context.Background(), userId, rows, page, order)
	if err != nil {
		t.Fatalf("expected to get tasks related to user %s: %s", userId, err)
	}

	if len(tasks) != 3 {
		t.Errorf("len(tasks)=%d, got %d", 3, len(tasks))
	}

	for _, tsk := range tasks {
		if tsk.UserId != userId {
			t.Errorf("task.UserId= %s, got %s", userId, tsk.UserId)
		}
	}
}

func TestParseField(t *testing.T) {
	tests := map[string]struct {
		input         string
		expectError   bool
		successResult task.Field
	}{
		"command field": {
			input:         "command",
			expectError:   false,
			successResult: task.FieldCommand,
		},
		"unknown field": {
			input:       ";SELECT",
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			field, err := task.ParseField(test.input)

			if !test.expectError {
				if err != nil {
					t.Fatalf("expected to parse the direction: %s", err)
				}

				if field != task.Field(test.successResult) {
					t.Errorf("field= %s, got %s", test.successResult, field)
				}
			} else {
				if err == nil {
					t.Fatalf("expected to fail when parsin unknown fields")
				}
			}
		})

	}

}

func TestParseDirection(t *testing.T) {
	tests := map[string]struct {
		input         string
		expectError   bool
		successResult task.Direction
	}{
		"ASC": {
			input:         "asc",
			expectError:   false,
			successResult: task.DirectionASC,
		},
		"unknown dir": {
			input:       ";DELETE",
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			field, err := task.ParseDirection(test.input)

			if !test.expectError {
				if err != nil {
					t.Fatalf("expected to parse the direction: %s", err)
				}

				if field != test.successResult {
					t.Errorf("field= %s, got %s", test.successResult, field)
				}
			} else {
				t.Log(err)
				if err == nil {
					t.Fatalf("expected to fail when parsin unknown dir")
				}
			}
		})

	}

}
