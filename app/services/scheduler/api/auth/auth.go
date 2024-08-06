// Package auth provides support for authentication and authorization.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

// Claims represents the authorization claims.
type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// keystore represents the set of behaviours required by auth package to lookup private and public keys.
type Keystore interface {
	PrivateKey(kid string) (string, error)
	PublicKey(kid string) (string, error)
}

// Auth represents the set of APIs used for authentication and authorization.
type Auth struct {
	keystore    Keystore
	userService *user.Service
}

// New creates an auth instance with provided keystore.
func New(ks Keystore, userService *user.Service) *Auth {
	return &Auth{
		keystore:    ks,
		userService: userService,
	}
}

// GenerateToken generates a jwt token with claims
func (a *Auth) GenerateToken(kid string, claims Claims) (string, error) {
	method := jwt.GetSigningMethod(jwt.SigningMethodRS256.Name)

	tkn := jwt.NewWithClaims(method, claims)

	//save the kid
	tkn.Header["kid"] = kid

	privatePEM, err := a.keystore.PrivateKey(kid)
	if err != nil {
		return "", fmt.Errorf("fetching private key: %w", err)
	}

	// decode it
	pemBlock, _ := pem.Decode([]byte(privatePEM))
	if pemBlock == nil || pemBlock.Type != "PRIVATE KEY" {
		return "", errors.New("failed to decode private key into pem block")
	}

	//since we want to support multiple key types in future we use "PKC8"
	privateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("invalid key algorithm")
	}

	tokenString, err := tkn.SignedString(rsaPrivateKey)
	if err != nil {
		return "", fmt.Errorf("signed string: %w", err)
	}
	return tokenString, nil
}

// ValidateToken is going to validate a jwt bearer token and return the corresponding user on success and possible errors.
func (a *Auth) ValidateToken(ctx context.Context, bearerToken string) (user.User, error) {
	prefix := "Bearer "

	if !strings.HasPrefix(bearerToken, prefix) {
		return user.User{}, errors.New("invalid authorization header format: Bearer <token>")
	}

	tknString := bearerToken[len(prefix):]

	keyFn := func(t *jwt.Token) (interface{}, error) {
		key, ok := t.Header["kid"]
		if !ok {
			return nil, errors.New("kid (key id) not found in token header")
		}

		kid, ok := key.(string)
		if !ok {
			return nil, errors.New("kid (key id) kid malformed")
		}

		//search for the public key for this kid
		publicPEM, err := a.keystore.PublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("fetching public key: %w", err)
		}

		pemBlock, _ := pem.Decode([]byte(publicPEM))
		if pemBlock == nil || pemBlock.Type != "PUBLIC KEY" {
			return nil, errors.New("failed to decode public key into pem block")
		}

		publicKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}

		return publicKey, nil
	}

	var claims Claims
	tkn, err := jwt.ParseWithClaims(tknString, &claims, keyFn)

	if err != nil {
		return user.User{}, fmt.Errorf("parse with claims: %w", err)
	}

	if !tkn.Valid {
		return user.User{}, errors.New("invalid token")
	}

	//check the user in db
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return user.User{}, fmt.Errorf("parse user id: %w", err)
	}

	usr, err := a.userService.GetUserById(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("getUserById: %w", err)
	}

	if !usr.Enabled {
		return user.User{}, errors.New("user is disabled")
	}

	return usr, nil
}

// Authorized checks the claims and allowed roles and returns nil if claims have the role otherwise returns an error.
func (a *Auth) Authorized(usr user.User, roles []user.Role) error {

	for _, have := range usr.Roles {
		if slices.Contains(roles, have) {
			return nil
		}
	}
	return errors.New("not authorized")
}

type mockKeyStore struct {
	privateKey string
	publicKey  string
}

func NewMockKeyStore(t *testing.T) mockKeyStore {
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
