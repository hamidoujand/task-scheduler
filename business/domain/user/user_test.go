package user_test

import (
	"context"
	"errors"
	"net/mail"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/business/domain/user/store/memory"
	"golang.org/x/crypto/bcrypt"
)

func TestParseRole(t *testing.T) {
	t.Parallel()
	r := "user"
	role, err := user.ParseRole(r)
	if err != nil {
		t.Fatalf("expected the role to be parsed: %s", err)
	}
	if role != user.RoleUser {
		t.Errorf("expected the role to be %q, but got %q", user.RoleUser, role)
	}
}

func TestParseRoles(t *testing.T) {
	t.Parallel()
	rr := []string{"usEr", "ADmin"}
	roles, err := user.ParseRoles(rr)
	if err != nil {
		t.Fatalf("should be able to create roles from slice of strings: %s", err)
	}

	if roles[0] != user.RoleUser {
		t.Errorf("expect the first role to be %q, but got %q", user.RoleUser, roles[0])
	}

	if roles[1] != user.RoleAdmin {
		t.Errorf("expect the first role to be %q, but got %q", user.RoleAdmin, roles[1])
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()
	pass := "test1234"
	email := mail.Address{
		Name:    "john",
		Address: "john@gmail.com",
	}
	nu := user.NewUser{
		Name:     "john",
		Email:    email,
		Roles:    []user.Role{user.RoleUser},
		Password: pass,
	}

	repo := memory.Repository{
		Users: make(map[uuid.UUID]user.User),
	}

	service := user.NewService(&repo)

	got, err := service.CreateUser(context.Background(), nu)
	if err != nil {
		t.Fatalf("expected the user to be saved with valid data: %s", err)
	}

	if got.Name != nu.Name {
		t.Errorf("user.Name= %s, got %s", nu.Name, got.Name)
	}

	if got.Email.Address != nu.Email.Address {
		t.Errorf("user.Email= %s, got %s", nu.Email.Address, got.Email.Address)
	}

	if !got.Enabled {
		t.Errorf("user.Enabled=%t, got %t", true, got.Enabled)
	}

	if len(got.Roles) != 1 {
		t.Errorf("len(user.Role)=%d, got %d", 1, len(got.Roles))
	}
	err = bcrypt.CompareHashAndPassword(got.PasswordHash, []byte(pass))
	if err != nil {
		t.Errorf("expected the password to be correctly hashes: %s", err)
	}

	//dplicated email
	_, err = service.CreateUser(context.Background(), nu)
	if err == nil {
		t.Fatalf("should not be able to create a user with duplicated email")
	}
	if !errors.Is(err, user.ErrUniqueEmail) {
		t.Errorf("error= %v, want %v", err, user.ErrUniqueEmail)
	}
}

func TestGetUserById(t *testing.T) {
	t.Parallel()
	pass := "test1234"
	nu := user.NewUser{
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:    []user.Role{user.RoleUser},
		Password: pass,
	}

	repo := memory.Repository{
		Users: make(map[uuid.UUID]user.User),
	}

	service := user.NewService(&repo)

	usr, err := service.CreateUser(context.Background(), nu)
	if err != nil {
		t.Fatalf("expected the user to be saved with valid data: %s", err)
	}

	// fetch by id
	got, err := service.GetUserById(context.Background(), usr.Id)
	if err != nil {
		t.Errorf("expected the user to be fetched with id: %s", err)
	}

	if !reflect.DeepEqual(got, usr) {
		t.Logf("got: %+v\n", got)
		t.Logf("expected: %+v\n", usr)
		t.Fatalf("expected the fetched user to be the same as saved")
	}

	//not found
	randomId := uuid.New()
	_, err = service.GetUserById(context.Background(), randomId)
	if err == nil {
		t.Fatal("expected to get an error when asking a user with random id")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("error= %v, got %v", user.ErrUserNotFound, err)
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()
	pass := "test1234"
	nu := user.NewUser{
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:    []user.Role{user.RoleUser},
		Password: pass,
	}

	repo := memory.Repository{
		Users: make(map[uuid.UUID]user.User),
	}

	service := user.NewService(&repo)

	usr, err := service.CreateUser(context.Background(), nu)
	if err != nil {
		t.Fatalf("expected the user to be saved with valid data: %s", err)
	}

	//delete
	if err := service.DeleteUser(context.Background(), usr); err != nil {
		t.Fatalf("expected to delete the given from repo: %s", err)
	}

	//fetch it
	_, err = service.GetUserById(context.Background(), usr.Id)
	if err == nil {
		t.Fatal("expected to get an error when asking for a deleted user")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("error= %v, got %v", user.ErrUserNotFound, err)
	}
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()
	pass := "test1234"
	nu := user.NewUser{
		Name: "john",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:    []user.Role{user.RoleUser},
		Password: pass,
	}

	repo := memory.Repository{
		Users: make(map[uuid.UUID]user.User),
	}

	service := user.NewService(&repo)

	usr, err := service.CreateUser(context.Background(), nu)
	if err != nil {
		t.Fatalf("expected the user to be saved with valid data: %s", err)
	}

	// fetch by id
	fetched, err := service.GetUserById(context.Background(), usr.Id)
	if err != nil {
		t.Errorf("expected the user to be fetched with id: %s", err)
	}

	//updates
	name := "jane"
	email := mail.Address{
		Name:    name,
		Address: "jane@gmail.com",
	}
	newPassword := "1234test"
	enabled := false
	roles := []user.Role{user.RoleAdmin, user.RoleUser}

	uu := user.UpdateUser{
		Name:     &name,
		Email:    &email,
		Roles:    roles,
		Password: &newPassword,
		Enabled:  &enabled,
	}

	updated, err := service.UpdateUser(context.Background(), uu, fetched)

	if err != nil {
		t.Fatalf("expected the updates to be applied: %s", err)
	}

	if updated.Name != name {
		t.Errorf("updated.Name= %s, got %s", name, updated.Name)
	}

	if updated.Email.Address != email.Address {
		t.Errorf("updated.Email= %s, got %s", email.Address, updated.Email.Address)
	}

	if len(updated.Roles) != 2 {
		t.Errorf("len(role)= %d, got %d", len(roles), len(updated.Roles))
	}

	if updated.Enabled {
		t.Errorf("updated.Enabled= %t, got %t", enabled, updated.Enabled)
	}

	err = bcrypt.CompareHashAndPassword(updated.PasswordHash, []byte(newPassword))
	if err != nil {
		t.Errorf("expected the password to be correctly hashes and update: %s", err)
	}
}

func TestGetByEmail(t *testing.T) {
	t.Parallel()
	pass := "test1234"
	email := mail.Address{
		Name:    "john",
		Address: "john@gmail.com",
	}

	nu := user.NewUser{
		Name:     "john",
		Email:    email,
		Roles:    []user.Role{user.RoleUser},
		Password: pass,
	}

	repo := memory.Repository{
		Users: make(map[uuid.UUID]user.User),
	}

	service := user.NewService(&repo)

	usr, err := service.CreateUser(context.Background(), nu)

	if err != nil {
		t.Fatalf("expected the user to be saved with valid data: %s", err)
	}

	//fetch by eamil
	got, err := service.GetByEmail(context.Background(), usr.Email)

	if err != nil {
		t.Fatalf("expected to fetch the user by email: %s", err)
	}

	if got.Name != nu.Name {
		t.Errorf("user.Name= %s, got %s", nu.Name, got.Name)
	}

	if got.Email.Address != nu.Email.Address {
		t.Errorf("user.Email= %s, got %s", nu.Email.Address, got.Email.Address)
	}

	if !got.Enabled {
		t.Errorf("user.Enabled=%t, got %t", true, got.Enabled)
	}

	if len(got.Roles) != 1 {
		t.Errorf("len(user.Role)=%d, got %d", 1, len(got.Roles))
	}
	err = bcrypt.CompareHashAndPassword(got.PasswordHash, []byte(pass))
	if err != nil {
		t.Errorf("expected the password to be correctly hashes: %s", err)
	}

	//random email
	email = mail.Address{
		Name:    "Jane",
		Address: "jane@gmail.com",
	}

	_, err = service.GetByEmail(context.Background(), email)
	if err == nil {
		t.Fatalf("expected to get an error while querying with a random email: %s", err)
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected error to be %v, got %v", user.ErrUserNotFound, err)
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()
	pass := "test1234"
	email := mail.Address{
		Name:    "john",
		Address: "john@gmail.com",
	}

	nu := user.NewUser{
		Name:     "john",
		Email:    email,
		Roles:    []user.Role{user.RoleUser},
		Password: pass,
	}

	repo := memory.Repository{
		Users: make(map[uuid.UUID]user.User),
	}

	service := user.NewService(&repo)

	usr, err := service.CreateUser(context.Background(), nu)
	if err != nil {
		t.Fatalf("expected the user to be saved with valid data: %s", err)
	}

	got, err := service.Login(context.Background(), email, pass)
	if err != nil {
		t.Fatalf("expected to login with the valid credentials: %s", err)
	}

	if !reflect.DeepEqual(usr, got) {
		t.Fatal("expected the user from login to be the same we got from create")
	}

	_, err = service.Login(context.Background(), email, "pass")
	if err == nil {
		t.Fatal("expected to get an error while using invalid credentials")
	}

	if !errors.Is(err, user.ErrLoginFailed) {
		t.Errorf("error = %v, want %v", err, user.ErrLoginFailed)
	}

	email = mail.Address{
		Name:    "jane",
		Address: "jane@hotmail.com",
	}
	_, err = service.Login(context.Background(), email, pass)
	if err == nil {
		t.Fatalf("expected the login to fail when using random email")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("error= %v, want %v", err, user.ErrUserNotFound)
	}
}
