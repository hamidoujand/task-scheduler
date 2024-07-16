package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hamidoujand/task-scheduler/foundation/logger"
)

// will be changed from build tags
var build = "development"

func main() {
	isProd := os.Getenv("ENV") == "production"
	//setup logger
	attrs := []slog.Attr{
		{Key: "build", Value: slog.StringValue(build)},
		{Key: "app", Value: slog.StringValue("task-scheduler")},
	}

	logger := logger.NewCustomLogger(slog.LevelInfo, isProd, attrs...)

	if err := run(logger); err != nil {
		fmt.Fprintf(os.Stderr, "err: %s", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	logger.Info("startup", "Hello", "World")

	return nil
}
