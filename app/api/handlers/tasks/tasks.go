package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/api/auth"
	"github.com/hamidoujand/task-scheduler/app/api/errs"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

// Handler represents set of http handlers.
type Handler struct {
	Validator   *errs.AppValidator
	TaskService *task.Service
	UserService *user.Service
}

// CreateTask creates a task for the authenticated user or returns possible errors.
func (h *Handler) CreateTask(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	var newTask NewTask
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "invalid data: %s", err.Error())
	}

	fields, ok := h.Validator.Check(newTask)

	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid input", fields)
	}

	//valid data

	var builder strings.Builder
	for key, val := range newTask.Environment {
		builder.WriteString(key + "=" + val)
		builder.WriteByte(' ')
	}

	domainTask := task.NewTask{
		Command:     newTask.Command,
		Args:        newTask.Args,
		ScheduledAt: newTask.ScheduledAt,
		UserId:      usr.Id,
		Image:       newTask.Image,
		Environment: builder.String(),
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

// GetTaskById returns a task for the given id if the user is the creator of that task or returns possible errors.
func (h *Handler) GetTaskById(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	taskId := r.PathValue("id")

	taskUUID, err := uuid.Parse(taskId)
	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q not a valid uuid", taskId)
	}

	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	t, err := h.TaskService.GetTaskById(ctx, taskUUID)

	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "task with id %q not found", taskId)
		}
		//internal
		return errs.NewAppInternalErr(err)
	}

	if t.UserId != usr.Id {
		return errs.NewAppErrorf(http.StatusUnauthorized, "unauthorized: task with id %s, does not belong to this user", taskId)
	}

	if err := web.Respond(ctx, w, http.StatusOK, fromDomainTask(t)); err != nil {
		return errs.NewAppInternalErr(err)
	}

	return nil
}

// DeleteTaskById deletes the task by id if creator or admin request it or returns possible errors.
func (h *Handler) DeleteTaskById(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

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

	if t.UserId != usr.Id && !isItAdmin(usr.Roles) {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized: operation not permitted")
	}

	if err := h.TaskService.DeleteTask(ctx, t); err != nil {
		return errs.NewAppInternalErr(err)
	}

	if err := web.Respond(ctx, w, http.StatusNoContent, nil); err != nil {
		return errs.NewAppInternalErr(err)
	}
	return nil
}

// GetAllTasksForUser returns all the task for given user or possible error.
func (h *Handler) GetAllTasksForUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	rows, page, err := parsePagination(r)
	if err != nil {
		return errs.NewAppError(http.StatusBadRequest, err.Error())
	}

	order, err := parseOrder(r)
	if err != nil {
		return errs.NewAppError(http.StatusBadRequest, err.Error())
	}

	userTasks, err := h.TaskService.GetTasksByUserId(ctx, usr.Id, rows, page, order)
	if err != nil {
		return errs.NewAppInternalErr(err)
	}

	appTasks := make([]Task, len(userTasks))
	for i, t := range userTasks {
		appTasks[i] = fromDomainTask(t)
	}

	return web.Respond(ctx, w, http.StatusOK, appTasks)
}

func isItAdmin(roles []user.Role) bool {
	return slices.Contains(roles, user.RoleAdmin)
}
