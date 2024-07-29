package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const (
	uniqueViolation = "23505"
)

var (
	ErrUniqueEmail  = errors.New("email is already in use")
	ErrUserNotFound = errors.New("user not found")
)

type repository interface {
	Create(ctx context.Context, usr User) error
	Update(ctx context.Context, usr User) error
	GetById(ctx context.Context, usrId uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	Delete(ctx context.Context, usr User) error
}

// Service represents the set of APIs that needed to interact with user domain.
type Service struct {
	userRepo repository
}

func NewService(repo repository) *Service {
	return &Service{
		userRepo: repo,
	}
}

// CreateUser creates a new user into repositoy and returns possible errors in case of duplicated email will return
// ErrDuplicatedEmail.
func (s *Service) CreateUser(ctx context.Context, nu NewUser) (User, error) {
	now := time.Now()
	id := uuid.New()
	hashed, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)

	if err != nil {
		return User{}, fmt.Errorf("generate from password: %w", err)
	}

	usr := User{
		Id:           id,
		Name:         nu.Name,
		Email:        nu.Email,
		Roles:        nu.Roles,
		PasswordHash: hashed,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	//save it
	if err := s.userRepo.Create(ctx, usr); err != nil {
		//check for unique email
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == uniqueViolation {
				return User{}, ErrUniqueEmail
			}
			return User{}, fmt.Errorf("create: pgerr.code= %s: %w", pgErr.Code, err)
		}
		return User{}, fmt.Errorf("create: %w", err)
	}

	return usr, nil
}

// GetUserById queries the repo for the user with id and returns possible errors, in case of "ErrNoRows" will return
// with ErrUserNotFound error.
func (s *Service) GetUserById(ctx context.Context, id uuid.UUID) (User, error) {
	usr, err := s.userRepo.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get by id: %w", err)
	}
	return usr, nil
}

// DeleteUser deletes the given user from repo and return possible errors.
func (s *Service) DeleteUser(ctx context.Context, usr User) error {
	if err := s.userRepo.Delete(ctx, usr); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// UpdateUser updates the user based on given updates and return the updated user back and possible errors.
func (s *Service) UpdateUser(ctx context.Context, uu UpdateUser, usr User) (User, error) {

	if uu.Name != nil {
		usr.Name = *uu.Name
	}

	if uu.Email != nil {
		usr.Email = *uu.Email
	}

	if uu.Enabled != nil {
		usr.Enabled = *uu.Enabled
	}

	if uu.Password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*uu.Password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, fmt.Errorf("generate from password: %w", err)
		}
		usr.PasswordHash = hashed
	}

	if uu.Roles != nil {
		usr.Roles = uu.Roles
	}

	now := time.Now()
	usr.UpdatedAt = now
	if err := s.userRepo.Update(ctx, usr); err != nil {
		return User{}, fmt.Errorf("update: %w", err)
	}
	return usr, nil
}

// GetByEmail fetches the user from repo by email and returns it or returns "ErrUserNotFound" in case user does not exists
// of any other possible error.
func (s *Service) GetByEmail(ctx context.Context, email mail.Address) (User, error) {
	usr, err := s.userRepo.GetByEmail(ctx, email.Address)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get by email: %w", err)
	}

	return usr, nil
}
