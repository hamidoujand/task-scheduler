package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func NewCustomLogger(level slog.Level, isProd bool, attrs ...slog.Attr) *slog.Logger {
	//setup logger
	replacer := func(groups []string, a slog.Attr) slog.Attr {
		//we do not want that long file path, just the file name and line number
		if a.Key == slog.SourceKey {
			if Source, ok := a.Value.Any().(*slog.Source); ok {
				filename := filepath.Base(Source.File)
				line := Source.Line
				functionName := Source.Function
				return slog.Attr{
					Key:   slog.SourceKey,
					Value: slog.StringValue(fmt.Sprintf("%s() line:%s:%d", functionName, filename, line)),
				}
			}
			return a
		}

		return a
	}

	opts := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replacer,
	}
	devHandler := slog.NewTextHandler(os.Stdout, opts).WithAttrs(attrs)
	prodHandler := slog.NewJSONHandler(os.Stdout, opts).WithAttrs(attrs)

	customHandler := newCustomLogHandler(prodHandler, devHandler, isProd)

	return slog.New(customHandler)

}

type customLogHandler struct {
	jsonHandler slog.Handler
	textHandler slog.Handler
	isProd      bool
}

func newCustomLogHandler(jsonHandler, textHandler slog.Handler, isProd bool) *customLogHandler {
	return &customLogHandler{
		jsonHandler: jsonHandler,
		textHandler: textHandler,
		isProd:      isProd,
	}
}

func (ch *customLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if ch.isProd {
		return ch.jsonHandler.Handle(ctx, record)
	}
	return ch.textHandler.Handle(ctx, record)
}

func (ch *customLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if ch.isProd {
		return ch.jsonHandler.Enabled(ctx, level)
	}
	return ch.textHandler.Enabled(ctx, level)
}

func (ch *customLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if ch.isProd {
		return ch.jsonHandler.WithAttrs(attrs)
	}
	return ch.textHandler.WithAttrs(attrs)
}
func (ch *customLogHandler) WithGroup(name string) slog.Handler {
	if ch.isProd {
		return ch.jsonHandler.WithGroup(name)
	}
	return ch.textHandler.WithGroup(name)
}
