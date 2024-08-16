package mid

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/hamidoujand/task-scheduler/app/api/errs"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func Panics() web.Middleware {
	m := func(h web.Handler) web.Handler {
		handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if r := recover(); r != nil {
					//need to capture and return error to "error handler mid"
					stackTrace := debug.Stack()
					err = errs.NewAppErrorf(http.StatusInternalServerError, "PANIC[%v] STACK[%s]", r, string(stackTrace))
				}
			}()

			return h(ctx, w, r)
		}

		return handler
	}
	return m
}
