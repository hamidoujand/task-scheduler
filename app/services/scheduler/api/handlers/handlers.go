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
	taskPostgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	userPostgresRepo "github.com/hamidoujand/task-scheduler/business/domain/user/store/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func RegisterRoutes(shutdown chan os.Signal, logger *slog.Logger, validator *errs.AppValidator, dbClient *postgres.Client) *web.App {
	app := web.NewApp(shutdown,
		mid.Logger(logger),
		mid.Errors(logger),
		mid.Panics(),
	)

	taskRepo := taskPostgresRepo.NewRepository(dbClient)
	taskService := task.NewService(taskRepo)

	userRepo := userPostgresRepo.NewRepository(dbClient)
	userService := user.NewService(userRepo)

	taskHandler := tasks.Handler{
		Validator:   validator,
		TaskService: taskService,
		UserService: userService,
	}
	//==============================================================================
	//tasks
	app.HandleFunc(http.MethodPost, "v1", "/api/tasks/", taskHandler.CreateTask)
	app.HandleFunc(http.MethodGet, "v1", "/api/tasks/{id}", taskHandler.GetTaskById)
	app.HandleFunc(http.MethodDelete, "v1", "/api/tasks/{id}", taskHandler.DeleteTaskById)

	return app
}
