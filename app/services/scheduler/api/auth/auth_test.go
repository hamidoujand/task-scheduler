package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/mail"
	"strings"
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

type mockKeyStore struct {
	privateKey string
	publicKey  string
}

func newMockKeyStore(t *testing.T) mockKeyStore {
	private, public := generateKeys(t)
	return mockKeyStore{
		privateKey: private,
		publicKey:  public,
	}
}

func (m mockKeyStore) PrivateKey(kid string) (string, error) {
	return m.privateKey, nil
}

func (m mockKeyStore) PublicKey(kid string) (string, error) {
	return m.publicKey, nil
}

func TestToken(t *testing.T) {
	ks := newMockKeyStore(t)
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

	parsedClaims, err := a.ValidateToken(context.Background(), bearer)
	if err != nil {
		t.Fatalf("expected the token to valid and get back parsed claims: %s", err)
	}

	if parsedClaims.Subject != c.Subject {
		t.Errorf("claims.Subject=%s , got %s", c.Subject, parsedClaims.Subject)
	}

	if parsedClaims.Issuer != c.Issuer {
		t.Errorf("claims.Issuer= %s, got %s", c.Issuer, parsedClaims.Issuer)
	}

	if parsedClaims.Roles[0] != c.Roles[0] {
		t.Errorf("claims.Roles[0]=%s , got %s", c.Roles[0], parsedClaims.Roles[0])
	}

	if !parsedClaims.ExpiresAt.Time.Equal(c.ExpiresAt.Time) {
		t.Errorf("claims.ExpiresAt= %s, got %s", c.ExpiresAt.Time, parsedClaims.ExpiresAt.Time)
	}

	if !parsedClaims.IssuedAt.Time.Equal(c.IssuedAt.Time) {
		t.Errorf("claims.IssuedAt= %s, got %s", c.IssuedAt.Time, parsedClaims.IssuedAt.Time)
	}
}

func generateKeys(t *testing.T) (string, string) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatalf("expected to generate random private key: %s", err)
	}

	//save it in PKCS8 format
	pkcs8Private, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatalf("expected to marshal key into pkcs8 format: %s", err)
	}

	PrivatePemBlock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Private,
	}

	var privatePEM strings.Builder
	err = pem.Encode(&privatePEM, &PrivatePemBlock)
	if err != nil {
		t.Fatalf("expected to encode into privatePEM: %s", err)
	}

	// public key
	var publicPEM strings.Builder
	publicBytes, err := x509.MarshalPKIXPublicKey(&private.PublicKey)

	if err != nil {
		t.Fatalf("expected to marshal public key into PKIX format: %s", err)
	}

	publicPemBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	}

	err = pem.Encode(&publicPEM, &publicPemBlock)
	if err != nil {
		t.Fatalf("expected to encode public key: %s", err)
	}

	return privatePEM.String(), publicPEM.String()
}

func TestAuthorized(t *testing.T) {
	ks := newMockKeyStore(t)

	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{},
	}

	userService := user.NewService(&userRepo)
	a := auth.New(ks, userService)

	c := auth.Claims{
		Roles: []string{user.RoleAdmin.String()},
	}

	err := a.Authorized(c, []user.Role{user.RoleAdmin})
	if err != nil {
		t.Fatalf("expected the claims to be authorized: %s", err)
	}

	err = a.Authorized(c, []user.Role{user.RoleUser})
	if err == nil {
		t.Fatal("expected the claims to not be authorized")
	}
}
