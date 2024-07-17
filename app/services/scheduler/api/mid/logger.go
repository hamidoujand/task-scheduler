package mid

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func Logger(logger *slog.Logger) web.Middleware {
	m := func(h web.Handler) web.Handler {
		handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			path := r.URL.Path

			if r.URL.RawQuery != "" {
				path = fmt.Sprintf("%s?%s", r.URL.Path, r.URL.RawQuery)
			}

			reqId := web.GetRequestId(ctx)

			logger.Info("request started", "id", reqId, "method", r.Method, "path", path, "remoteAddr", r.RemoteAddr)
			err := h(ctx, w, r)

			startedAt := web.GetStartedAt(ctx)
			statusCode := web.GetStatusCode(ctx)

			logger.Info("request completed", "id", reqId, "method", r.Method, "path", path, "remoteAddr", r.RemoteAddr,
				"took", time.Since(startedAt),
				"statusCode", statusCode,
			)

			return err
		}
		return handler
	}
	return m
}
