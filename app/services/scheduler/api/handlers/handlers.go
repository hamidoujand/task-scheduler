package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/mid"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func RegisterRoutes(shutdown chan os.Signal, logger *slog.Logger, validator *errs.AppValidator) *web.App {
	app := web.NewApp(shutdown,
		mid.Logger(logger),
		mid.Errors(logger),
		mid.Panics(),
	)

	taskHandler := tasks.Handler{
		Validator: validator,
	}

	app.HandleFunc(http.MethodGet, "v1", "/api/tasks", taskHandler.CreateTask)

	return app
}
