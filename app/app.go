// Package app provides reusable application bootstrap logic shared by
// frontends such as the CLI and a future TUI.
package app

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

// Config defines the runtime dependencies needed to bootstrap the application.
type Config struct {
	DBPath    string
	LogLevel  string
	LogOutput io.Writer
}

// Instance contains the bootstrapped runtime objects shared by frontends.
type Instance struct {
	Service *core.Service
	Close   func() error
}

// Open creates a reusable application instance from the given config.
func Open(cfg Config) (*Instance, error) {
	level, err := parseLogLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	logOutput := cfg.LogOutput
	if logOutput == nil {
		logOutput = os.Stderr
	}

	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	sqliteStore, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: level}))
	return &Instance{
		Service: core.New(sqliteStore, logger),
		Close:   sqliteStore.Close,
	}, nil
}

func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %q (must be debug, info, warn, error)", level)
	}
}
