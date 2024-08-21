package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/hamidoujand/task-scheduler/app/api/errs"
	"github.com/hamidoujand/task-scheduler/app/api/handlers"
	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/keystore"
	"github.com/hamidoujand/task-scheduler/foundation/logger"
	"github.com/redis/go-redis/v9"
)

// will be changed from build tags
var build = "0.0.1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "err: %s", err)
		os.Exit(1)
	}
}

func run() error {
	//==========================================================================
	//setup configurations
	configs := struct {
		API struct {
			Host            string        `conf:"default:0.0.0.0:8000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			Environment     string        `conf:"default:development"`
		}

		DB struct {
			User            string        `conf:"default:postgres"`
			Password        string        `conf:"default:password,mask"`
			Host            string        `conf:"default:localhost:5432"`
			Name            string        `conf:"default:postgres"`
			MaxIdleConns    int           `conf:"default:10"`
			MaxOpenConns    int           `conf:"default:10"`
			MaxIdleConnTime time.Duration `conf:"default:5m"`
			MaxConnLifeTime time.Duration `conf:"default:10m"`
			DisableTLS      bool          `conf:"default:true"`
		}

		Auth struct {
			KeysFolder string        `conf:"default:zarf/keys/"`
			ActiveKid  string        `conf:"default:a41bace0-da3c-4119-85ad-bbd293bf31ee"`
			Issuer     string        `conf:"default:task scheduler"`
			TokenAge   time.Duration `conf:"default:24h"`
		}

		Redis struct {
			Host     string        `conf:"default:localhost:6379"`
			Password string        `conf:"default:'',"`
			DBIdx    int           `conf:"default:0"`
			Timeout  time.Duration `conf:"default:5s"`
		}

		Rabbitmq struct {
			Host                 string        `conf:"default:localhost:5672"`
			User                 string        `conf:"default:guest"`
			Password             string        `conf:"default:guest"`
			MaxTimeForConnection time.Duration `conf:"default:1m"`
		}

		Scheduler struct {
			MaxFailedTasksRetries       int           `conf:"default:1"`
			MaxTimeForTaskUpdates       time.Duration `conf:"default:1m"` //slow machine maybe
			MaxTimeForGraceFullShutdown time.Duration `conf:"default:1m"`
			MaxTimeForTaskExecution     time.Duration `conf:"default:1m"`
		}
	}{}

	prefix := "TASKS"
	if help, err := conf.Parse(prefix, &configs); err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		//some error we need to handle
		return fmt.Errorf("parsing config: %w", err)
	}

	//==========================================================================
	//setup logger
	isProd := configs.API.Environment == "production"

	attrs := []slog.Attr{
		{Key: "build", Value: slog.StringValue(build)},
		{Key: "app", Value: slog.StringValue("task-scheduler")},
	}

	logger := logger.NewCustomLogger(slog.LevelInfo, isProd, attrs...)

	//==========================================================================
	//validator
	appValidator, err := errs.NewAppValidator()
	if err != nil {
		return fmt.Errorf("creating app validator: %w", err)
	}
	logger.Info("application validator", "status", "successfully initialized")
	//==========================================================================
	//database setup
	logger.Info("database setup", "status", "connecting", "host", configs.DB.Host)
	client, err := postgres.NewClient(postgres.Config{
		User:        configs.DB.User,
		Password:    configs.DB.Password,
		Host:        configs.DB.Host,
		Name:        configs.DB.Name,
		DisableTLS:  configs.DB.DisableTLS,
		MaxIdleConn: configs.DB.MaxIdleConns,
		MaxOpenConn: configs.DB.MaxOpenConns,
		MaxIdleTime: configs.DB.MaxIdleConnTime,
		MaxLifeTime: configs.DB.MaxConnLifeTime,
	})
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}
	logger.Info("database setup", "status", "checking database engine", "host", configs.DB.Host)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := client.StatusCheck(ctx); err != nil {
		return fmt.Errorf("status check: %w", err)
	}
	logger.Info("database", "status", "status check ran successfully", "host", configs.DB.Host)

	//migrations
	logger.Info("database", "status", "running migrations", "host", configs.DB.Host)
	if err := client.Migrate(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	logger.Info("database", "status", "ready to use")

	//==========================================================================
	//keystore
	logger.Info("keystore", "status", "initializing keystore support")

	ks, err := keystore.LoadFromFS(os.DirFS(configs.Auth.KeysFolder))
	if err != nil {
		return fmt.Errorf("loadFromFS: %w", err)
	}

	//==========================================================================
	//redis
	logger.Info("redis", "status", "initializing keystore support")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     configs.Redis.Host,
		Password: configs.Redis.Password,
		DB:       configs.Redis.DBIdx,
	})

	logger.Info("redis", "status", "pinging redis engine")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*configs.Redis.Timeout)
	defer cancel()

	err = redisClient.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	logger.Info("redis", "status", "successfully connected")

	//==========================================================================
	// rabbitmq setup
	logger.Info("rabbitmq", "status", "setting up the connection")
	ctx, cancel = context.WithTimeout(context.Background(), configs.Rabbitmq.MaxTimeForConnection)
	defer cancel()

	rabbitMQC, err := rabbitmq.NewClient(ctx, rabbitmq.Configs{
		Host:     configs.Rabbitmq.Host,
		User:     configs.Rabbitmq.User,
		Password: configs.Rabbitmq.Password,
	})
	if err != nil {
		return fmt.Errorf("new rabbitmq client: %w", err)
	}

	logger.Info("rabbitmq", "status", "connection successfully made to the server")

	//==========================================================================
	//server

	serverErrors := make(chan error, 1)
	shutdownCh := make(chan os.Signal, 1)

	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT)

	maxRunningTasks := runtime.GOMAXPROCS(0)

	app, err := handlers.RegisterRoutes(handlers.Config{
		Shutdown:                    shutdownCh,
		Logger:                      logger,
		Validator:                   appValidator,
		PostgresClient:              client,
		ActiveKID:                   configs.Auth.ActiveKid,
		TokenAge:                    configs.Auth.TokenAge,
		Keystore:                    ks,
		RClient:                     rabbitMQC,
		RedisClient:                 redisClient,
		MaxRunningTasks:             maxRunningTasks,
		MaxFailedTasksRetry:         configs.Scheduler.MaxFailedTasksRetries,
		MaxTimeForTaskUpdates:       configs.Scheduler.MaxTimeForTaskUpdates,
		MaxTimeForSchedulerShutdown: configs.Scheduler.MaxTimeForGraceFullShutdown,
		MaxTimeForTaskExecution:     configs.Scheduler.MaxTimeForTaskExecution,
	})

	if err != nil {
		return fmt.Errorf("register routes: %w", err)
	}

	srv := http.Server{
		Addr:        configs.API.Host,
		Handler:     http.TimeoutHandler(app, configs.API.WriteTimeout, "timed out"),
		ReadTimeout: configs.API.ReadTimeout,
		ErrorLog:    slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	logger.Info("mux", "status", "registering routes to the mux")
	//server start
	go func() {
		logger.Info("server", "status", "started", "host", configs.API.Host, "environment", configs.API.Environment)
		serverErrors <- srv.ListenAndServe()
	}()

	//block
	select {
	case serverErr := <-serverErrors:
		return fmt.Errorf("server error: %w", serverErr)
	case signal := <-shutdownCh:
		//graceful shutdown
		logger.Info("shutdown", "status", "started", "signal", signal)

		ctx, cancel := context.WithTimeout(context.Background(), configs.API.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			//force shutdown
			_ = srv.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		logger.Info("shutdown", "status", "completed")
	}
	return nil
}
