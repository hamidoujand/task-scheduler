package tasks_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/domain/task/store/memory"
)

func TestCreateTask200(t *testing.T) {

	memRepo := memory.Repository{
		Tasks: make(map[uuid.UUID]task.Task),
	}
	taskService := task.NewService(&memRepo)

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to create a validator: %s", err)
	}

	h := tasks.Handler{
		Validator:   v,
		TaskService: taskService,
	}

	jsn := `
	{
		"command":"ls",
		"args": ["-l","-a"],
		"scheduledAt":"2024-08-01T12:00:00-07:00" 
	}
	`
	r := strings.NewReader(jsn)

	req := httptest.NewRequest(http.MethodPost, "/v1/api/tasks/", r)
	w := httptest.NewRecorder()
	err = h.CreateTask(context.Background(), w, req)
	if err != nil {
		t.Fatalf("should be able to create a task with valid input: %s", err)
	}
	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expect to get status %d but got %d", http.StatusCreated, w.Result().StatusCode)
	}
	// var resp response
	var resp tasks.Task
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("should be able to decode response body: %s", err)
	}

	if resp.Command != "ls" {
		t.Errorf("expected the command to be %q, but got %q", "ls", resp.Command)
	}

	if resp.Status != "pending" {
		t.Errorf("expected the status to be %q, but got %q", "pending", resp.Status)
	}

	now := time.Now()
	if !resp.ScheduledAt.After(now) {
		t.Logf("now: %s scheduledAt: %s", now, resp.ScheduledAt)
		t.Errorf("expected the scheduledAt to be in future")
	}

	if resp.CreatedAt.IsZero() {
		t.Error("expected the createdAt field to not be zero value")
	}

	if resp.UpdatedAt.IsZero() {
		t.Error("expected the updatedAt field to not be zero value")
	}

}

func TestGetTaskById200(t *testing.T) {
	taskId := uuid.New()
	savedTsk := task.Task{
		Id:          taskId,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      "completed",
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour).Truncate(0),
		CreatedAt:   time.Now().Add(-time.Hour * 2).Truncate(0),
		UpdatedAt:   time.Now().Add(-time.Hour * 1).Truncate(0),
	}
	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			taskId: savedTsk,
		},
	}

	taskService := task.NewService(&memRepo)

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to create a validator: %s", err)
	}

	h := tasks.Handler{
		Validator:   v,
		TaskService: taskService,
	}
	path := "/v1/api/tasks/" + taskId.String()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.SetPathValue("id", taskId.String())

	w := httptest.NewRecorder()

	err = h.GetTaskById(context.Background(), w, req)

	if err != nil {
		t.Fatalf("should be able to get the task with id %s: %s", taskId, err)
	}

	status := w.Result().StatusCode
	if status != http.StatusOK {
		t.Fatalf("expected to get status %d, but got %d", http.StatusOK, status)
	}

	var task tasks.Task
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("should be able to decode into a task value: %s", err)
	}

	if task.Command != savedTsk.Command {
		t.Errorf("expected the command to be %q, but got %q", savedTsk.Command, task.Command)
	}

	if len(task.Args) != len(savedTsk.Args) {
		t.Errorf("expected the length of args to be %d, but got %d", len(savedTsk.Args), len(task.Args))
	}

	if task.Status != savedTsk.Status {
		t.Errorf("expected the status to be %q, but got %q", savedTsk.Status, task.Status)
	}

}

func TestGetTaskById404(t *testing.T) {
	taskId := uuid.New()
	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{},
	}

	taskService := task.NewService(&memRepo)

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to create a validator: %s", err)
	}

	h := tasks.Handler{
		Validator:   v,
		TaskService: taskService,
	}
	path := "/v1/api/tasks/" + taskId.String()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.SetPathValue("id", taskId.String())

	w := httptest.NewRecorder()

	err = h.GetTaskById(context.Background(), w, req)
	if err == nil {
		t.Fatal("should not be able to find the task")
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expect the type of err to be *errs.AppErr but got %T", err)
	}

	if appErr.Code != http.StatusNotFound {
		t.Errorf("expect the status of error to be %d, but got %d", http.StatusNotFound, appErr.Code)
	}

}

func TestDeleteTaskById200(t *testing.T) {
	taskId := uuid.New()
	savedTsk := task.Task{
		Id:          taskId,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      "completed",
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour).Truncate(0),
		CreatedAt:   time.Now().Add(-time.Hour * 2).Truncate(0),
		UpdatedAt:   time.Now().Add(-time.Hour * 1).Truncate(0),
	}
	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			taskId: savedTsk,
		},
	}

	taskService := task.NewService(&memRepo)

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to create a validator: %s", err)
	}

	h := tasks.Handler{
		Validator:   v,
		TaskService: taskService,
	}
	path := "/v1/api/tasks/" + taskId.String()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.SetPathValue("id", taskId.String())

	w := httptest.NewRecorder()

	err = h.DeleteTaskById(context.Background(), w, req)
	if err != nil {
		t.Fatalf("should be able to delete the task with id %s", taskId)
	}

	statusCode := w.Result().StatusCode
	if statusCode != http.StatusNoContent {
		t.Errorf("expected the status code to be %d but got %d", http.StatusNoContent, statusCode)
	}
}

func TestDeleteTaskById404(t *testing.T) {
	taskId := uuid.New()

	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{},
	}

	taskService := task.NewService(&memRepo)

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to create a validator: %s", err)
	}

	h := tasks.Handler{
		Validator:   v,
		TaskService: taskService,
	}
	path := "/v1/api/tasks/" + taskId.String()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.SetPathValue("id", taskId.String())
	w := httptest.NewRecorder()

	err = h.DeleteTaskById(context.Background(), w, req)

	if err == nil {
		t.Fatal("should return an error when the task not found")
	}

	var appErr *errs.AppError

	if !errors.As(err, &appErr) {
		t.Fatalf("expected the error type to be *appError but got %T", err)
	}

	if appErr.Code != http.StatusNotFound {
		t.Errorf("expected the status in appError to be %d, but got %d", http.StatusNotFound, appErr.Code)
	}
}
