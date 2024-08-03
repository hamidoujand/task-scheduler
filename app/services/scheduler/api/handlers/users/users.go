package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

// Handler represents set of APIs used
type Handler struct {
	Validator      *errs.AppValidator
	UsersService   *user.Service
	Auth           *auth.Auth
	ActiveKID      string
	TokenExpiresAt time.Time
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

	//TODO add a custom validator for "Roles" field
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

// DeleteUserById is going to delete user when the the role is admin or user itself and returns possible errors.
func (h *Handler) DeleteUserById(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	userId, err := uuid.Parse(id)
	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q not a valid uuid", id)
	}

	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	if userId != usr.Id && !isItAdmin(usr.Roles) {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	fetched, err := h.UsersService.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "user with id %s not found", userId)
		}
		return errs.NewAppInternalErr(err)
	}

	if err := h.UsersService.DeleteUser(ctx, fetched); err != nil {
		return errs.NewAppInternalErr(err)
	}

	return web.Respond(ctx, w, http.StatusNoContent, nil)
}

// UpdateUser updates a user and returns the possible errors.
func (h *Handler) UpdateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	userId, err := uuid.Parse(id)
	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q is invalid uuid", userId)
	}

	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	if usr.Id != userId && !isItAdmin(usr.Roles) {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}
	var uu UpdateUser
	if err := json.NewDecoder(r.Body).Decode(&uu); err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "invalid json: %s", err.Error())
	}

	fields, ok := h.Validator.Check(uu)
	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid input", fields)
	}

	//fetch the user to see if exists
	fetched, err := h.UsersService.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "user with id %q not found", userId)
		}
		return errs.NewAppInternalErr(err)
	}

	//update it
	updated, err := h.UsersService.UpdateUser(ctx, uu.toServiceUpdateUser(), fetched)
	if err != nil {
		errs.NewAppInternalErr(err)
	}

	return web.Respond(ctx, w, http.StatusOK, toAppUser(updated))
}

func (h *Handler) UpdateRole(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	userId, err := uuid.Parse(id)
	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "%q not a valid uuid", id)
	}

	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	//admin only
	if !isItAdmin(usr.Roles) {
		return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
	}

	var ur UpdateRole
	if err := json.NewDecoder(r.Body).Decode(&ur); err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "invalid json: %s", err)
	}

	fields, ok := h.Validator.Check(ur)
	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid input", fields)
	}

	parsedRoles, err := user.ParseRoles(ur.Roles)
	if err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "parsing roles: %s", err)
	}

	uu := user.UpdateUser{
		Roles: parsedRoles,
	}

	//fetch user from db to make sure exists
	fetched, err := h.UsersService.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return errs.NewAppErrorf(http.StatusNotFound, "user with id %q not found", id)
		}
		return errs.NewAppInternalErr(err)
	}

	updated, err := h.UsersService.UpdateUser(ctx, uu, fetched)
	if err != nil {
		return errs.NewAppInternalErr(err)
	}

	return web.Respond(ctx, w, http.StatusOK, toAppUser(updated))
}

// Signup is going to signup a user and generate a token.
func (h *Handler) Signup(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var signup SignUp
	if err := json.NewDecoder(r.Body).Decode(&signup); err != nil {
		return errs.NewAppErrorf(http.StatusBadRequest, "invalid json data: %s", err)
	}

	fields, ok := h.Validator.Check(signup)
	if !ok {
		return errs.NewAppValidationError(http.StatusBadRequest, "invalid input", fields)
	}

	newUser, err := h.UsersService.CreateUser(ctx, signup.toServiceNewUser())
	if err != nil {
		if errors.Is(err, user.ErrUniqueEmail) {
			return errs.NewAppErrorf(http.StatusConflict, "%q already in use", signup.Email)
		}
		return errs.NewAppInternalErr(err)
	}

	//generate token
	c := auth.Claims{
		Roles: user.EncodeRoles(newUser.Roles),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "task-scheduler",
			Subject:   newUser.Id.String(),
			ExpiresAt: jwt.NewNumericDate(h.TokenExpiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tkn, err := h.Auth.GenerateToken(h.ActiveKID, c)
	if err != nil {
		return errs.NewAppInternalErr(err)
	}

	return web.Respond(ctx, w, http.StatusCreated, toAppUserWithToken(newUser, tkn))
}

func (h *Handler) Signin(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return nil
}

func isItAdmin(roles []user.Role) bool {
	return slices.Contains(roles, user.RoleAdmin)
}
