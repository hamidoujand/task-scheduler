package mid

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

// Errors is a middleware used to do the error handling of the routes.
func Errors(logger *slog.Logger) web.Middleware {
	m := func(h web.Handler) web.Handler {
		handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			err := h(ctx, w, r)

			if err != nil {

				appErr, ok := err.(*errs.AppError)

				if !ok {
					return errs.NewAppErrorf(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
				}

				//handle err
				logger.Error("handling error during request",
					"message", appErr.Message,
					"sourceFile", filepath.Base(appErr.FileName),
					"functionName", filepath.Base(appErr.FuncName),
				)

				//after logging change the message to internal server error
				if appErr.Code == http.StatusInternalServerError {
					appErr.Message = http.StatusText(http.StatusInternalServerError)
				}

				//send response to client
				if err := web.Respond(ctx, w, appErr.Code, appErr); err != nil {
					return errs.NewAppErrorf(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
				}
			}

			//no err
			return nil
		}
		return handler
	}
	return m
}
