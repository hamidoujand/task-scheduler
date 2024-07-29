package postgres

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

type User struct {
	Id           uuid.UUID
	Name         string
	Email        string
	Roles        []string
	PasswordHash []byte
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ToPostgresUser creates a User that will be saved inside of postgres.
func ToPostgresUser(u user.User) User {
	return User{
		Id:           u.Id,
		Name:         u.Name,
		Email:        u.Email.Address,
		Roles:        user.EncodeRoles(u.Roles),
		PasswordHash: u.PasswordHash,
		Enabled:      u.Enabled,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func (u User) ToServiceUser() user.User {
	//since we getting them from db which already validated
	roles, _ := user.ParseRoles(u.Roles)

	return user.User{
		Id:   u.Id,
		Name: u.Name,
		Email: mail.Address{
			Name:    u.Name,
			Address: u.Email,
		},
		Roles:        roles,
		PasswordHash: u.PasswordHash,
		Enabled:      u.Enabled,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
