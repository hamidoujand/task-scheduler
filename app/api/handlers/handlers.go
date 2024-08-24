package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/hamidoujand/task-scheduler/app/api/auth"
	"github.com/hamidoujand/task-scheduler/app/api/errs"
	"github.com/hamidoujand/task-scheduler/app/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/app/api/handlers/users"
	"github.com/hamidoujand/task-scheduler/app/api/mid"
	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/scheduler"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/scheduler/store/redis"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	taskPostgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	userPostgresRepo "github.com/hamidoujand/task-scheduler/business/domain/user/store/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/web"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Shutdown                    chan os.Signal
	Logger                      *slog.Logger
	Validator                   *errs.AppValidator
	PostgresClient              *postgres.Client
	ActiveKID                   string
	TokenAge                    time.Duration
	Keystore                    auth.Keystore
	RClient                     *rabbitmq.Client
	RedisClient                 *redis.Client
	MaxRunningTasks             int
	MaxFailedTasksRetry         int
	MaxTimeForTaskUpdates       time.Duration
	MaxTimeForSchedulerShutdown time.Duration
	MaxTimeForTaskExecution     time.Duration
}

func RegisterRoutes(conf Config) (*web.App, error) {
	//==============================================================================
	//setup
	const version = "v1"
	app := web.NewApp(conf.Shutdown,
		mid.Logger(conf.Logger),
		mid.Errors(conf.Logger),
		mid.Panics(),
	)

	taskRepo := taskPostgresRepo.NewRepository(conf.PostgresClient)
	taskService, err := task.NewService(taskRepo, conf.RClient)
	if err != nil {
		return nil, fmt.Errorf("new service: %w", err)
	}

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
	//redisRepo
	redisR := redisRepo.NewRepository(conf.RedisClient)

	//setup scheduler
	scheduler, err := scheduler.New(scheduler.Config{
		RabbitClient:            conf.RClient,
		Logger:                  conf.Logger,
		TaskService:             taskService,
		RedisRepo:               redisR,
		MaxRunningTask:          conf.MaxRunningTasks,
		MaxRetries:              conf.MaxFailedTasksRetry,
		MaxTimeForUpdateOps:     conf.MaxTimeForTaskUpdates,
		MaxTimeForTaskExecution: conf.MaxTimeForTaskExecution,
	})

	if conf.MaxTimeForSchedulerShutdown <= 0 {
		conf.MaxTimeForSchedulerShutdown = time.Minute
	}

	if err != nil {
		return nil, fmt.Errorf("creating scheduler: %w", err)
	}

	//setup all consumers
	if err := scheduler.ConsumeTasks(); err != nil {
		return nil, fmt.Errorf("task consumer: %w", err)
	}

	if err := scheduler.OnTaskSuccess(); err != nil {
		return nil, fmt.Errorf("on task success consumer: %w", err)
	}

	if err := scheduler.OnTaskRetry(); err != nil {
		return nil, fmt.Errorf("on task retry consumer: %w", err)
	}

	if err := scheduler.OnTaskFailure(); err != nil {
		return nil, fmt.Errorf("on task failure: %w", err)
	}

	if err := scheduler.MonitorScheduledTasks(); err != nil {
		return nil, fmt.Errorf("monitor scheduled tasks: %w", err)
	}

	//scheduler monitor
	//long-lived goroutine
	go func() {
		//wait for server shutdown
		<-conf.Shutdown
		conf.Logger.Info("scheduler", "status", "received shutdown signal", "msg", "shutting down scheduler")
		ctx, cancel := context.WithTimeout(context.Background(), conf.MaxTimeForSchedulerShutdown)
		defer cancel()

		if err := scheduler.Shutdown(ctx); err != nil {
			conf.Logger.Error("scheduler", "status", "failed to gracfully shutdown", "msg", err)
			return
		}
	}()

	//==============================================================================
	//tasks
	app.HandleFunc(http.MethodPost, version, "/api/tasks/", taskHandler.CreateTask, mid.Authenticate(auth))
	app.HandleFunc(http.MethodGet, version, "/api/tasks/{id}", taskHandler.GetTaskById, mid.Authenticate(auth))
	app.HandleFunc(http.MethodDelete, version, "/api/tasks/{id}", taskHandler.DeleteTaskById, mid.Authenticate(auth))

	//==============================================================================
	//users
	app.HandleFunc(http.MethodPost, version, "/api/users/", userHandler.CreateUser,
		mid.Authenticate(auth),
		mid.Authorized(auth, user.RoleAdmin),
	)

	app.HandleFunc(http.MethodPost, version, "/api/users/login", userHandler.Login)
	app.HandleFunc(http.MethodPost, version, "/api/users/signup", userHandler.Signup)
	app.HandleFunc(http.MethodPut, version, "/api/users/role/{id}", userHandler.UpdateRole,
		mid.Authenticate(auth),
		mid.Authorized(auth, user.RoleAdmin))

	app.HandleFunc(http.MethodGet, version, "/api/users/{id}", userHandler.GetUserById)
	app.HandleFunc(http.MethodPut, version, "/api/users/{id}", userHandler.UpdateUser, mid.Authenticate(auth))
	app.HandleFunc(http.MethodDelete, version, "/api/users/{id}", userHandler.DeleteUserById, mid.Authenticate(auth))

	return app, nil
}
