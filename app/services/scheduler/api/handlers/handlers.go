package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/mid"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	postgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func RegisterRoutes(shutdown chan os.Signal, logger *slog.Logger, validator *errs.AppValidator, dbClient *postgres.Client) *web.App {
	app := web.NewApp(shutdown,
		mid.Logger(logger),
		mid.Errors(logger),
		mid.Panics(),
	)

	taskStore := postgresRepo.NewRepository(dbClient)
	taskService := task.NewService(taskStore)

	taskHandler := tasks.Handler{
		Validator:   validator,
		TaskService: taskService,
	}
	//==============================================================================
	//tasks
	app.HandleFunc(http.MethodPost, "v1", "/api/tasks/", taskHandler.CreateTask)
	app.HandleFunc(http.MethodGet, "v1", "/api/tasks/{id}", taskHandler.GetTaskById)
	app.HandleFunc(http.MethodDelete, "v1", "/api/tasks/{id}", taskHandler.DeleteTaskById)

	return app
}
