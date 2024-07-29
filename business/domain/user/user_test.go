package user_test

import (
	"testing"

	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

func TestParseRole(t *testing.T) {
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
