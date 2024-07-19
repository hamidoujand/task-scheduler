package tasks

import (
	"context"
	"net/http"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

type Handler struct {
	Validator *errs.AppValidator
}

func (h *Handler) CreateTask(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	model := struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}{}

	field, ok := h.Validator.Check(model)
	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid data", field)
	}

	msg := map[string]string{
		"msg": "hello world",
	}
	return web.Respond(ctx, w, http.StatusOK, msg)
}
