package users

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

// User represents a user value that will be send to client.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	PasswordHash []byte    `json:"-"`
	Token        string    `json:"token,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func toAppUser(usr user.User) User {
	return User{
		ID:           usr.Id.String(),
		Name:         usr.Name,
		Email:        usr.Email.Address,
		Roles:        user.EncodeRoles(usr.Roles),
		PasswordHash: usr.PasswordHash,
		Enabled:      usr.Enabled,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
	}
}
func toAppUserWithToken(usr user.User, token string) User {
	return User{
		ID:           usr.Id.String(),
		Name:         usr.Name,
		Email:        usr.Email.Address,
		Roles:        user.EncodeRoles(usr.Roles),
		PasswordHash: usr.PasswordHash,
		Token:        token,
		Enabled:      usr.Enabled,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
	}
}

// NewUser represents all of the required data to create a new user.
type NewUser struct {
	Name            string   `json:"name" validate:"required"`
	Email           string   `json:"email" validate:"required,email"`
	Password        string   `json:"password" validate:"required,min=8"`
	Roles           []string `json:"roles" validate:"required"`
	PasswordConfirm string   `json:"passwordConfirm" validate:"required,eqfield=Password"`
}

func (nu NewUser) toServiceNewUser() (user.NewUser, error) {
	roles, err := user.ParseRoles(nu.Roles)
	if err != nil {
		return user.NewUser{}, fmt.Errorf("parsing roles: %w", err)
	}

	return user.NewUser{
		Name: nu.Name,
		Email: mail.Address{
			Name:    nu.Name,
			Address: nu.Email,
		},
		Roles:    roles,
		Password: nu.Password,
	}, nil
}

// UpdateUser represents all of the data that a role user can update.
type UpdateUser struct {
	Name            *string `json:"name" validate:"omitempty,min=4"`
	Email           *string `json:"email" validate:"omitempty,email"`
	Enabled         *bool   `json:"enabled"`
	Password        *string `json:"password" validate:"omitempty,min=8"`
	PasswordConfirm *string `json:"passowordConfirm" validate:"omitempty,eqfield=Password"`
}

func (u UpdateUser) toServiceUpdateUser() user.UpdateUser {
	return user.UpdateUser{
		Name: u.Name,
		Email: &mail.Address{
			Name:    *u.Name,
			Address: *u.Email,
		},
		Password: u.Password,
		Enabled:  u.Enabled,
	}
}

// UpdateRole represents required data for updating roles
type UpdateRole struct {
	//TODO search validator package for better enum like validation.
	Roles []string `json:"roles" validate:"required"`
}

// SignUp represents all of required data for signup.
type SignUp struct {
	Name            string `json:"name" validate:"required"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,eqfield=Password"`
}

func (si SignUp) toServiceNewUser() user.NewUser {

	return user.NewUser{
		Name: si.Name,
		Email: mail.Address{
			Name:    si.Name,
			Address: si.Email,
		},
		Roles:    []user.Role{user.RoleUser},
		Password: si.Password,
	}
}

// Login represents all required data for login.
type Login struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}
