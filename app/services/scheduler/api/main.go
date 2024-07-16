package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/hamidoujand/task-scheduler/foundation/logger"
)

// will be changed from build tags
var build = "development"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "err: %s", err)
		os.Exit(1)
	}
}

func run() error {
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
	//setup logger
	isProd := configs.API.Environment == "production"

	attrs := []slog.Attr{
		{Key: "build", Value: slog.StringValue(build)},
		{Key: "app", Value: slog.StringValue("task-scheduler")},
	}

	logger := logger.NewCustomLogger(slog.LevelInfo, isProd, attrs...)

	logger.Info("current app configurations", "configs", cfg)

	return nil
}
