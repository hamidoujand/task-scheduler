package user

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	Id           uuid.UUID
	Name         string
	Email        mail.Address
	Roles        []Role
	PasswordHash []byte
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser represents all required data to create a new user in system.
type NewUser struct {
	Name     string
	Email    string
	Roles    []Role
	Password string
}

// UpdateUser represents all the data that can be updated on a user type.
type UpdateUser struct {
	Name     *string
	Email    *mail.Address
	Roles    []Role
	Password *string
	Enabled  *bool
}
