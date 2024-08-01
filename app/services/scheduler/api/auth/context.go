package auth

import (
	"context"
	"errors"

	"github.com/hamidoujand/task-scheduler/business/domain/user"
)

type ctxKey int

const userKey ctxKey = 1

// SetUser injects the user into ctx to be passed to next handler.
func SetUser(ctx context.Context, usr user.User) context.Context {
	return context.WithValue(ctx, userKey, usr)
}

// GetUser fetches the user from context or returns possible error if not user exists.
func GetUser(ctx context.Context) (user.User, error) {
	usr, ok := ctx.Value(userKey).(user.User)
	if !ok {
		return user.User{}, errors.New("no user in context")
	}
	return usr, nil
}
