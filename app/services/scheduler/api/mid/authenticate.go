package mid

import (
	"context"
	"net/http"
	"time"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func Authenticate(a *auth.Auth) web.Middleware {
	return func(h web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			authHeader := r.Header.Get("authorization")
			if authHeader == "" {
				return errs.NewAppError(http.StatusUnauthorized, "missing authorization header")
			}
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()

			user, err := a.ValidateToken(ctx, authHeader)
			if err != nil {
				return errs.NewAppError(http.StatusUnauthorized, err.Error())
			}

			//add claims into ctx
			ctx = auth.SetUser(ctx, user)
			//call the next handler
			return h(ctx, w, r)
		}
	}
}
