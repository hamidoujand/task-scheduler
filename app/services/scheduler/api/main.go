package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/hamidoujand/task-scheduler/foundation/logger"
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
	}{}

	prefix := "SCHEDULER"
	if help, err := conf.Parse(prefix, &configs); err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		//some error we need to handle
		return fmt.Errorf("parsing config: %w", err)
	}

	cfg, err := conf.String(&configs)
	if err != nil {
		return fmt.Errorf("stringify configs: %w", err)
	}
	//==========================================================================
	//setup logger
	isProd := configs.API.Environment == "production"

	attrs := []slog.Attr{
		{Key: "build", Value: slog.StringValue(build)},
		{Key: "app", Value: slog.StringValue("task-scheduler")},
	}

	logger := logger.NewCustomLogger(slog.LevelInfo, isProd, attrs...)

	logger.Info("configurations", "configs", cfg)

	//==========================================================================
	//server

	srv := http.Server{
		Addr:        configs.API.Host,
		Handler:     http.TimeoutHandler(nil, configs.API.WriteTimeout, "timed out"),
		ReadTimeout: configs.API.ReadTimeout,
		ErrorLog:    slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	serverErrors := make(chan error, 1)
	shutdownCh := make(chan os.Signal, 1)

	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT)

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
