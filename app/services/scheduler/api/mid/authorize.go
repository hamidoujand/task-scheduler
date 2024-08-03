package mid

import (
	"context"
	"net/http"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func Authorized(a *auth.Auth, roles ...user.Role) web.Middleware {
	return func(h web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			//get the user
			usr, err := auth.GetUser(ctx)
			if err != nil {
				return errs.NewAppError(http.StatusUnauthorized, err.Error())
			}

			//check the user roles
			err = a.Authorized(usr, roles)
			if err != nil {
				return errs.NewAppError(http.StatusUnauthorized, "unauthorized")
			}

			return h(ctx, w, r)
		}
	}
}
