package mid

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/hamidoujand/task-scheduler/app/api/errs"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

// Errors is a middleware used to do the error handling of the routes.
func Errors(logger *slog.Logger) web.Middleware {
	m := func(h web.Handler) web.Handler {
		handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			err := h(ctx, w, r)
			if err != nil {
				var appErr *errs.AppError
				if errors.As(err, &appErr) {
					// handle err
					logger.Error("handling error during request",
						"message", appErr.Message,
						"sourceFile", filepath.Base(appErr.FileName),
						"functionName", filepath.Base(appErr.FuncName),
					)

					// after logging change the message to internal server error
					if appErr.Code == http.StatusInternalServerError {
						appErr.Message = http.StatusText(http.StatusInternalServerError)
					}
					// send response to client
					if err := web.Respond(ctx, w, appErr.Code, appErr); err != nil {
						return errs.NewAppInternalErr(err)
					}
					return nil //stop err propagation
				}

				return errs.NewAppInternalErr(err)

			}

			//no err
			return nil
		}
		return handler
	}
	return m
}
