package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

type Handler struct {
	Validator   *errs.AppValidator
	TaskService *task.Service
	UserService *user.Service
}

func (h *Handler) CreateTask(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var newTask NewTask

	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "invalid data: %s", err.Error())
	}

	fields, ok := h.Validator.Check(newTask)

	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid input", fields)
	}

	//valid data

	domainTask := task.NewTask{
		Command:     newTask.Command,
		Args:        newTask.Args,
		ScheduledAt: newTask.ScheduledAt,
	}

	task, err := h.TaskService.CreateTask(ctx, domainTask)
	if err != nil {
		return errs.NewAppInternalErr(err)
	}

	if err := web.Respond(ctx, w, http.StatusCreated, fromDomainTask(task)); err != nil {
		return errs.NewAppInternalErr(err)
	}

	return nil
}

func (h *Handler) GetTaskById(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	taskId := r.PathValue("id")
	taskUUID, err := uuid.Parse(taskId)

	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q not a valid uuid", taskId)
	}

	t, err := h.TaskService.GetTaskById(ctx, taskUUID)

	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "task with id %q not found", taskId)
		}
		//internal
		return errs.NewAppInternalErr(err)
	}

	if err := web.Respond(ctx, w, http.StatusOK, fromDomainTask(t)); err != nil {
		return errs.NewAppInternalErr(err)
	}

	return nil
}

func (h *Handler) DeleteTaskById(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	taskId := r.PathValue("id")
	taskUUID, err := uuid.Parse(taskId)

	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q not a valid uuid", taskId)
	}

	t, err := h.TaskService.GetTaskById(ctx, taskUUID)

	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "task with id %q not found", taskId)
		}
		//internal
		return errs.NewAppInternalErr(err)
	}

	if err := h.TaskService.DeleteTask(ctx, t); err != nil {
		return errs.NewAppInternalErr(err)
	}

	if err := web.Respond(ctx, w, http.StatusNoContent, nil); err != nil {
		return errs.NewAppInternalErr(err)
	}
	return nil
}
