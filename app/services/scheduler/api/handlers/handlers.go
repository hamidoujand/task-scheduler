package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/users"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/mid"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	taskPostgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	userPostgresRepo "github.com/hamidoujand/task-scheduler/business/domain/user/store/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

type Config struct {
	Shutdown       chan os.Signal
	Logger         *slog.Logger
	Validator      *errs.AppValidator
	PostgresClient *postgres.Client
	ActiveKID      string
	TokenAge       time.Duration
	Keystore       auth.Keystore
}

func RegisterRoutes(conf Config) *web.App {
	//==============================================================================
	//setup
	const version = "v1"
	app := web.NewApp(conf.Shutdown,
		mid.Logger(conf.Logger),
		mid.Errors(conf.Logger),
		mid.Panics(),
	)

	taskRepo := taskPostgresRepo.NewRepository(conf.PostgresClient)
	taskService := task.NewService(taskRepo)

	userRepo := userPostgresRepo.NewRepository(conf.PostgresClient)
	userService := user.NewService(userRepo)

	taskHandler := tasks.Handler{
		Validator:   conf.Validator,
		TaskService: taskService,
		UserService: userService,
	}

	//setup auth
	auth := auth.New(conf.Keystore, userService)

	userHandler := users.Handler{
		Validator:    conf.Validator,
		UsersService: userService,
		Auth:         auth,
		ActiveKID:    conf.ActiveKID,
		TokenAge:     conf.TokenAge,
	}

	//==============================================================================
	//tasks
	app.HandleFunc(http.MethodPost, version, "/api/tasks/", taskHandler.CreateTask, mid.Authenticate(auth))
	app.HandleFunc(http.MethodGet, version, "/api/tasks/{id}", taskHandler.GetTaskById, mid.Authenticate(auth))
	app.HandleFunc(http.MethodDelete, version, "/api/tasks/{id}", taskHandler.DeleteTaskById, mid.Authenticate(auth))

	//==============================================================================
	//users
	app.HandleFunc(http.MethodPost, version, "/api/users", userHandler.CreateUser,
		mid.Authenticate(auth),
		mid.Authorized(auth, user.RoleAdmin),
	)

	app.HandleFunc(http.MethodGet, version, "/api/users/{id}", userHandler.GetUserById)
	app.HandleFunc(http.MethodPut, version, "/api/users/{id}", userHandler.UpdateUser, mid.Authenticate(auth))
	app.HandleFunc(http.MethodDelete, version, "/api/users/{id}", userHandler.DeleteUserById, mid.Authenticate(auth))
	app.HandleFunc(http.MethodPost, version, "/api/users/signup", userHandler.Signup)

	app.HandleFunc(http.MethodPut, version, "/api/users/role", userHandler.UpdateRole,
		mid.Authenticate(auth),
		mid.Authorized(auth, user.RoleAdmin))

	app.HandleFunc(http.MethodPost, version, "/api/users/login", userHandler.Login)

	return app
}
