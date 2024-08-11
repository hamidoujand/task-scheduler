package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/brokertest"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/domain/task/store/memory"
)

func TestCreateTask(t *testing.T) {
	t.Parallel()
	store := memory.Repository{
		Tasks: make(map[uuid.UUID]task.Task),
	}

	rClient := brokertest.NewTestClient(t, context.Background(), "test_create_task")

	service, err := task.NewService(&store, rClient)
	if err != nil {
		t.Fatalf("expected to create service: %s", err)
	}

	nt := task.NewTask{
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		ScheduledAt: time.Now().Add(time.Minute),
		Image:       "alpine",
		UserId:      uuid.New(),
		Environment: "APP_NAME=test",
	}

	//create task
	tsk, err := service.CreateTask(context.Background(), nt)
	if err != nil {
		t.Fatalf("expected the task to be saved: %s", err)
	}

	if tsk.Status != task.StatusPending {
		t.Errorf("expected status to be %q, but got %q", task.StatusPending, tsk.Status)
	}

	if tsk.Image != nt.Image {
		t.Errorf("image= %s, got %s", nt.Image, tsk.Image)
	}

	if tsk.Command != nt.Command {
		t.Errorf("command= %s, got %s", nt.Command, tsk.Command)
	}

	if tsk.Environment != nt.Environment {
		t.Errorf("environment= %s, got %s", nt.Environment, tsk.Environment)
	}

	if tsk.CreatedAt.IsZero() || tsk.UpdatedAt.IsZero() {
		t.Errorf("expected createdAt and updatedAt field to not be zero time values")
	}

	//check the message inside rabbitmq
	queue := "tasks"
	msgs, err := rClient.Consumer(queue)
	if err != nil {
		t.Fatalf("expected to get back a rabbitmq delivery: %s", err)
	}

	//consume
	d := <-msgs

	if d.ContentType != "application/json" {
		t.Fatalf("content-type= %s, got %s", "application/json", d.ContentType)
	}

	var queueTask task.Task
	if err := json.Unmarshal(d.Body, &queueTask); err != nil {
		t.Fatalf("expected to unmarshal task: %s", err)
	}

	if queueTask.Command != tsk.Command {
		t.Errorf("command= %s, got %s", tsk.Command, queueTask.Command)
	}
	//should be able to ack the message
	if err := d.Ack(false); err != nil {
		t.Fatalf("expected to ack the message: %s", err)
	}
}

func TestGetTaskById(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	now := time.Now()
	dt := task.Task{
		Id:          id,
		Command:     "docker",
		Args:        []string{"ps"},
		Status:      task.StatusCompleted,
		UserId:      uuid.New(),
		Image:       "alpine",
		Environment: "APP_NAME=test",
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

	rClient := brokertest.NewTestClient(t, context.Background(), "test_get_task_by_id")

	service, err := task.NewService(&store, rClient)
	if err != nil {
		t.Fatalf("expected to create service: %s", err)
	}

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

	if tsk.Image != dt.Image {
		t.Errorf("image= %s, got %s", dt.Image, tsk.Image)
	}

	if tsk.Environment != dt.Environment {
		t.Errorf("environment= %s, got %s", dt.Environment, tsk.Environment)
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
	t.Parallel()

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

	rClient := brokertest.NewTestClient(t, context.Background(), "test_delete_task")

	service, err := task.NewService(&store, rClient)
	if err != nil {
		t.Fatalf("expected to create service: %s", err)
	}

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
	t.Parallel()

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

	rClient := brokertest.NewTestClient(t, context.Background(), "test_update_task")

	service, err := task.NewService(&store, rClient)
	if err != nil {
		t.Fatalf("expected to create service: %s", err)
	}

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
	t.Parallel()

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

	rClient := brokertest.NewTestClient(t, context.Background(), "test_tasks_by_user_id")

	service, err := task.NewService(&store, rClient)
	if err != nil {
		t.Fatalf("expected to create service: %s", err)
	}

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
	t.Parallel()

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
	t.Parallel()

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
