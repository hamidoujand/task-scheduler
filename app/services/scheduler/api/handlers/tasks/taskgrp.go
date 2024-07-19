package tasks

import (
	"context"
	"errors"
	"net/http"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

type Handler struct {
}

func (h *Handler) CreateTask(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	err := errors.New("task validation failed")
	if err != nil {
		return errs.NewAppError(http.StatusBadRequest, err.Error())
	}

	msg := map[string]string{
		"msg": "hello world",
	}
	return web.Respond(ctx, w, http.StatusOK, msg)
}
