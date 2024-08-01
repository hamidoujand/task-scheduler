package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

// Handler represents set of APIs used
type Handler struct {
	Validator    *errs.AppValidator
	UsersService *user.Service
}

// CreateUser creates a user inside the system, returns errors on duplicated emails and invalid inputs.
func (h *Handler) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var nu NewUser

	if err := json.NewDecoder(r.Body).Decode(&nu); err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "invalid json data: %s", err.Error())
	}

	//validate
	fields, ok := h.Validator.Check(nu)
	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid data", fields)
	}

	usrService, err := nu.toServiceNewUser()
	if err != nil {
		return errs.NewAppError(http.StatusBadRequest, err.Error())
	}

	usr, err := h.UsersService.CreateUser(ctx, usrService)

	if err != nil {
		if errors.Is(err, user.ErrUniqueEmail) {
			return errs.NewAppError(http.StatusConflict, err.Error())
		}
		return errs.NewAppInternalErr(err)
	}

	return web.Respond(ctx, w, http.StatusCreated, toAppUser(usr))
}

// GetUserById gets the user from db by id and in case there is no user with that id returns an error.
func (h *Handler) GetUserById(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	userId, err := uuid.Parse(id)
	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q, invalid uuid", id)
	}

	usr, err := h.UsersService.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "%q, no user with this id", id)
		}
		return errs.NewAppInternalErr(err)
	}

	return web.Respond(ctx, w, http.StatusOK, toAppUser(usr))
}
