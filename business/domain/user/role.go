package user

import (
	"fmt"
	"strings"
)

type Role int

const (
	RoleAdmin Role = iota
	RoleUser
)

var roleNames = []string{"admin", "user"}

func (r Role) String() string {
	if r < RoleAdmin || r > RoleUser {
		return "UNKNOWN"
	}
	return roleNames[r]
}

// ParseRole creates a role from string.
func ParseRole(role string) (Role, error) {
	for i, r := range roleNames {
		if r == strings.ToLower(role) {
			return Role(i), nil
		}
	}
	return Role(-1), fmt.Errorf("%q is invalid role", role)
}

// ParseRoles its a utility function that creates a slice of Role from slice of strings.
func ParseRoles(roles []string) ([]Role, error) {
	results := make([]Role, len(roles))
	for i, r := range roles {
		role, err := ParseRole(r)
		if err != nil {
			return nil, fmt.Errorf("parsing role: %w", err)
		}
		results[i] = role
	}
	return results, nil
}

// EncodeRoles its a utility function that converts slice of Roles into slice of strings.
func EncodeRoles(rr []Role) []string {
	results := make([]string, len(rr))
	for i, r := range rr {
		results[i] = r.String()
	}
	return results
}
