// Package auth provides support for authentication and authorization.
package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"strings"

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
type keystore interface {
	PrivateKey(kid string) (string, error)
	PublicKey(kid string) (string, error)
}

// Auth represents the set of APIs used for authentication and authorization.
type Auth struct {
	keystore    keystore
	userService *user.Service
}

// New creates an auth instance with provided keystore.
func New(ks keystore, userService *user.Service) *Auth {
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

// ValidateToken is going to validate a jwt bearer token and return the claims on success and possible errors.
func (a *Auth) ValidateToken(ctx context.Context, bearerToken string) (Claims, error) {
	prefix := "Bearer "

	if !strings.HasPrefix(bearerToken, prefix) {
		return Claims{}, errors.New("invalid authorization header format: Bearer <token>")
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
			return Claims{}, errors.New("failed to decode public key into pem block")
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
		return Claims{}, fmt.Errorf("parse with claims: %w", err)
	}

	if !tkn.Valid {
		return Claims{}, errors.New("invalid token")
	}

	//check the user in db
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Claims{}, fmt.Errorf("parse user id: %w", err)
	}

	usr, err := a.userService.GetUserById(ctx, id)
	if err != nil {
		return Claims{}, fmt.Errorf("getUserById: %w", err)
	}

	if !usr.Enabled {
		return Claims{}, errors.New("user is disabled")
	}

	return claims, nil
}

// Authorized checks the claims and allowed roles and returns nil if claims have the role otherwise returns an error.
func (a *Auth) Authorized(claims Claims, roles []user.Role) error {
	parsedRoles, err := user.ParseRoles(claims.Roles)
	if err != nil {
		return fmt.Errorf("parseRoles: %w", err)
	}

	for _, have := range parsedRoles {
		if slices.Contains(roles, have) {
			return nil
		}
	}
	return errors.New("not authorized")
}
