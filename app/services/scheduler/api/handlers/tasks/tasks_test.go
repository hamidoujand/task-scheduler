package tasks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/domain/task/store/memory"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

func TestCreateTask(t *testing.T) {

	memRepo := memory.Repository{
		Tasks: make(map[uuid.UUID]task.Task),
	}
	taskService := task.NewService(&memRepo)

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to create a validator: %s", err)
	}

	taskCreator := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h := tasks.Handler{
		Validator:   v,
		TaskService: taskService,
	}

	tests := map[string]struct {
		input        tasks.NewTask
		expectError  bool
		status       int
		fields       []string
		unauthorized bool
	}{
		"success": {
			input: tasks.NewTask{
				Command:     "ls",
				Args:        []string{"-l", "-a"},
				ScheduledAt: time.Now().Add(time.Hour),
			},
			expectError: false,
			status:      http.StatusCreated,
		},
		"invalid input": {
			input: tasks.NewTask{
				Command:     "rm",
				Args:        []string{";", "-l"},
				ScheduledAt: time.Now().Add(-time.Hour),
			},
			expectError: true,
			status:      http.StatusBadRequest,
			fields:      []string{"command", "args", "scheduledAt"},
		},

		"unauthorized user": {
			input: tasks.NewTask{
				Command:     "date",
				ScheduledAt: time.Now().Add(time.Hour),
			},
			expectError:  true,
			status:       http.StatusUnauthorized,
			unauthorized: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			var buff bytes.Buffer
			if err := json.NewEncoder(&buff).Encode(test.input); err != nil {
				t.Fatalf("expected to encode input: %s", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/api/tasks/", &buff)
			w := httptest.NewRecorder()

			var ctx context.Context
			if test.unauthorized {
				ctx = context.Background()
			} else {
				ctx = auth.SetUser(req.Context(), taskCreator)
			}

			err := h.CreateTask(ctx, w, req)
			if !test.expectError {
				//success path
				if err != nil {
					t.Fatalf("should be able to create a task with valid input: %s", err)
				}
				if w.Result().StatusCode != test.status {
					t.Fatalf("expect to get status %d but got %d", test.status, w.Result().StatusCode)
				}
				// var resp response
				var resp tasks.Task
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("should be able to decode response body: %s", err)
				}

				if resp.Command != test.input.Command {
					t.Errorf("expected the command to be %q, but got %q", test.input.Command, resp.Command)
				}

				if resp.Status != task.StatusPending.String() {
					t.Errorf("expected the status to be %q, but got %q", task.StatusPending, resp.Status)
				}

				if resp.UserId != taskCreator.Id.String() {
					t.Errorf("task.UserId=%s, got %s", taskCreator.Id, resp.UserId)
				}

				if resp.CreatedAt.IsZero() {
					t.Error("expected the createdAt field to not be zero value")
				}

				if resp.UpdatedAt.IsZero() {
					t.Error("expected the updatedAt field to not be zero value")
				}

			} else {
				//failure path
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got %T", err)
				}

				if appErr.Code != test.status {
					t.Errorf("appError.Code=%d, got %d", test.status, appErr.Code)
				}

				if appErr.Fields != nil {
					for name := range appErr.Fields {
						if !slices.Contains(test.fields, name) {
							t.Errorf("expected field %s to be invalid", name)
						}
					}
				}
			}
		})
	}

}

func TestGetTaskById(t *testing.T) {

	taskCreator := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	randomUser := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	taskId := uuid.New()

	tsk := task.Task{
		Id:          taskId,
		UserId:      taskCreator.Id,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			taskId: tsk,
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

	tests := map[string]struct {
		input       uuid.UUID
		status      int
		expectError bool
	}{
		"success": {
			input:       taskId,
			status:      http.StatusOK,
			expectError: false,
		},
		"random_id": {
			input:       uuid.New(),
			status:      http.StatusNotFound,
			expectError: true,
		},

		"unauthorized_user": {
			input:       taskId,
			status:      http.StatusUnauthorized,
			expectError: true,
		},
		"not_owner": {
			input:       taskId,
			status:      http.StatusUnauthorized,
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			path := "/v1/api/tasks/" + test.input.String()
			r := httptest.NewRequest(http.MethodGet, path, nil)
			r.SetPathValue("id", test.input.String())

			w := httptest.NewRecorder()

			var ctx context.Context

			switch name {
			case "unauthorized_user":
				ctx = r.Context()
			case "not_owner":
				ctx = auth.SetUser(r.Context(), randomUser)
			default:
				ctx = auth.SetUser(r.Context(), taskCreator)

			}

			err := h.GetTaskById(ctx, w, r)
			if !test.expectError {

				if err != nil {
					t.Errorf("expected to get task by id %q: %s", test.input, err)
				}
				//### dynamic status
				status := w.Result().StatusCode
				if status != test.status {
					t.Fatalf("expect to get status %d but got %d", test.status, status)
				}
				// var resp response
				var resp tasks.Task
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("should be able to decode response body: %s", err)
				}

				if resp.Command != tsk.Command {
					t.Errorf("expected the command to be %q, but got %q", tsk.Command, resp.Command)
				}

				if resp.Status != task.StatusCompleted.String() {
					t.Errorf("expected the status to be %q, but got %q", task.StatusCompleted, resp.Status)
				}

				if resp.UserId != taskCreator.Id.String() {
					t.Errorf("task.UserId=%s, got %s", taskCreator.Id, resp.UserId)
				}

			} else {
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got %T", err)
				}

				if appErr.Code != test.status {
					t.Errorf("appError.Code=%d, got %d", test.status, appErr.Code)
				}
			}
		})
	}

}

func TestDeleteTaskById(t *testing.T) {
	taskCreator := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	randomUser := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	admin := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleAdmin},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	task1Id := uuid.New()

	tsk1 := task.Task{
		Id:          task1Id,
		UserId:      taskCreator.Id,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	task2Id := uuid.New()
	tsk2 := task.Task{
		Id:          task1Id,
		UserId:      taskCreator.Id,
		Command:     "date",
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	task3Id := uuid.New()
	tsk3 := task.Task{
		Id:          task3Id,
		UserId:      uuid.New(),
		Command:     "date",
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			task1Id: tsk1,
			task2Id: tsk2,
			task3Id: tsk3,
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

	tests := map[string]struct {
		input       uuid.UUID
		status      int
		expectError bool
	}{
		"owner": {
			input:       task1Id,
			status:      http.StatusNoContent,
			expectError: false,
		},
		"admin": {
			input:       task2Id,
			status:      http.StatusNoContent,
			expectError: false,
		},
		"not_found": {
			input:       uuid.New(),
			status:      http.StatusNotFound,
			expectError: true,
		},

		"random_user": {
			input:       task3Id,
			status:      http.StatusUnauthorized,
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := "/v1/api/tasks/" + test.input.String()

			r := httptest.NewRequest(http.MethodDelete, path, nil)
			r.SetPathValue("id", test.input.String())

			w := httptest.NewRecorder()

			var ctx context.Context
			switch name {
			case "admin":
				ctx = auth.SetUser(r.Context(), admin)
			case "random_user":
				ctx = auth.SetUser(r.Context(), randomUser)
			default:
				ctx = auth.SetUser(r.Context(), taskCreator)
			}

			err := h.DeleteTaskById(ctx, w, r)
			if !test.expectError {
				if err != nil {
					t.Fatalf("expected to delete task with id %s: %s", test.input, err)
				}
				// ### dynamic status
				status := w.Result().StatusCode
				if status != test.status {
					t.Fatalf("expect to get status %d but got %d", test.status, status)
				}
			} else {
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got %T", err)
				}

				if appErr.Code != test.status {
					t.Errorf("appError.Code=%d, got %d", test.status, appErr.Code)
				}
			}
		})
	}
}

// pagination and order by have solid tests against real database so we do not need to test against
// memory repository just the parsing part of "order by" and "pagination"
func TestGetAllTasks(t *testing.T) {
	taskCreator := user.User{
		Id:   uuid.New(),
		Name: "John Doe",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: []byte("[hashed_pass]"),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	task1Id := uuid.New()

	tsk1 := task.Task{
		Id:          task1Id,
		UserId:      taskCreator.Id,
		Command:     "ls",
		Args:        []string{"-l", "-a"},
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	task2Id := uuid.New()
	tsk2 := task.Task{
		Id:          task1Id,
		UserId:      taskCreator.Id,
		Command:     "date",
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	task3Id := uuid.New()
	tsk3 := task.Task{
		Id:          task3Id,
		UserId:      uuid.New(),
		Command:     "date",
		Status:      task.StatusCompleted,
		Result:      "data",
		ErrMessage:  "",
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour * 2),
		UpdatedAt:   time.Now().Add(-time.Hour * 2),
	}

	memRepo := memory.Repository{
		Tasks: map[uuid.UUID]task.Task{
			task1Id: tsk1,
			task2Id: tsk2,
			task3Id: tsk3,
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

	tests := map[string]struct {
		page        int
		rows        int
		field       string
		direction   string
		expectError bool
		status      int
	}{

		"success": {
			page:        1,
			rows:        2,
			field:       "command",
			direction:   "desc",
			expectError: false,
			status:      http.StatusOK,
		},
		"invalid field": {
			page:        1,
			rows:        2,
			field:       "SELECT",
			direction:   "ASc",
			expectError: true,
			status:      http.StatusBadRequest,
		},
		"invalid direction": {
			page:        1,
			rows:        2,
			field:       "command",
			direction:   "Up",
			expectError: true,
			status:      http.StatusBadRequest,
		},
		"invalid rows": {
			page:        1,
			rows:        0,
			field:       "command",
			direction:   "desc",
			expectError: true,
			status:      http.StatusBadRequest,
		},
		"invalid page": {
			page:        0,
			rows:        2,
			field:       "command",
			direction:   "Asc",
			expectError: true,
			status:      http.StatusBadRequest,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			path := fmt.Sprintf("/v1/api/tasks?orderby=%s,%s&page=%d&rows=%d",
				test.field,
				test.direction,
				test.page,
				test.rows,
			)
			r := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			ctx := auth.SetUser(r.Context(), taskCreator)

			err := h.GetAllTasksForUser(ctx, w, r)
			if !test.expectError {
				if err != nil {
					t.Fatalf("expected to get all user's tasks: %s", err)
				}

				status := w.Result().StatusCode
				if status != http.StatusOK {
					t.Errorf("status=%d got %d", http.StatusOK, status)
				}

				var resp []tasks.Task
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("should be able to decode response body: %s", err)
				}

				if len(resp) != 2 {
					t.Errorf("expected the results to be 3 tasks got %d", len(resp))
				}
			} else {
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got %T", err)
				}

				if appErr.Code != test.status {
					t.Errorf("appError.Code=%d, got %d", test.status, appErr.Code)
				}
			}
		})
	}

}
