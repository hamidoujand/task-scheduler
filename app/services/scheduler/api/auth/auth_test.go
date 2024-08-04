package auth_test

import (
	"context"
	"net/mail"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/business/domain/user/store/memory"
	"golang.org/x/crypto/bcrypt"
)

const (
	kid = "s4sKIjD9kIRjxs2tulPqGLdxSfgPErRN1Mu3Hd9k9NQ"
)

func TestToken(t *testing.T) {
	ks := auth.NewMockKeyStore(t)
	usrId := uuid.New()
	now := time.Now()

	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("expected the password to be hashed: %s", err)
	}

	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{
			usrId: {
				Id:   usrId,
				Name: "John",
				Email: mail.Address{
					Name:    "john",
					Address: "john@gmail.com",
				},
				Roles:        []user.Role{user.RoleUser},
				Enabled:      true,
				CreatedAt:    now,
				UpdatedAt:    now,
				PasswordHash: pass,
			},
		},
	}

	userService := user.NewService(&userRepo)
	a := auth.New(ks, userService)

	c := auth.Claims{
		Roles: []string{user.RoleUser.String()},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test",
			Subject:   usrId.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 1, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tkn, err := a.GenerateToken(kid, c)
	if err != nil {
		t.Fatalf("expected the jwt token to be generated: %s", err)
	}

	bearer := "Bearer " + tkn

	usr, err := a.ValidateToken(context.Background(), bearer)
	if err != nil {
		t.Fatalf("expected the token to valid and get back user: %s", err)
	}

	if usr.Id.String() != c.Subject {
		t.Fatalf("expected the userId to match the claims")
	}

	savedUser := userRepo.Users[usrId]
	if !reflect.DeepEqual(usr, savedUser) {
		t.Logf("repo:\n\n%v\n\n", savedUser)
		t.Logf("claim:\n\n%v\n\n", usr)
		t.Fatal("expected user from validate token to be the same as saved in repo")
	}
}

func TestAuthorized(t *testing.T) {
	ks := auth.NewMockKeyStore(t)
	usr := user.User{
		Roles: []user.Role{user.RoleAdmin},
	}

	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{},
	}

	userService := user.NewService(&userRepo)
	a := auth.New(ks, userService)

	err := a.Authorized(usr, []user.Role{user.RoleAdmin})
	if err != nil {
		t.Fatalf("expected the claims to be authorized: %s", err)
	}

	err = a.Authorized(usr, []user.Role{user.RoleUser})
	if err == nil {
		t.Fatal("expected the claims to not be authorized")
	}
}
