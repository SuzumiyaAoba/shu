// Package app provides reusable application bootstrap logic shared by
// frontends such as the CLI and a future TUI.
package app

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
)

// StoreOpener opens a [core.Store] from a DSN or path.
type StoreOpener func(dsn string) (core.Store, error)

// Config defines the runtime dependencies needed to bootstrap the application.
type Config struct {
	DBPath     string
	LogLevel   string
	LogOutput  io.Writer
	Logger     *slog.Logger
	Store      core.Store
	OpenStore  StoreOpener
	HTTPClient *http.Client
	Cleanup    func() error
}

// Instance contains the bootstrapped runtime objects shared by frontends.
type Instance struct {
	Service *core.Service
	Store   core.Store
	Logger  *slog.Logger
	Close   func() error
}

// Open creates a reusable application instance from the given config.
func Open(cfg Config) (*Instance, error) {
	logger, err := buildLogger(cfg)
	if err != nil {
		return nil, err
	}

	dataStore, ownedClose, err := openStore(cfg)
	if err != nil {
		return nil, err
	}

	closeFn := composeCleanup(ownedClose, cfg.Cleanup)
	svc := core.New(dataStore, logger)
	if cfg.HTTPClient != nil {
		svc.SetHTTPClientWithUserAgent(cfg.HTTPClient)
	}

	return &Instance{
		Service: svc,
		Store:   dataStore,
		Logger:  logger,
		Close:   closeFn,
	}, nil
}

func buildLogger(cfg Config) (*slog.Logger, error) {
	if cfg.Logger != nil {
		return cfg.Logger, nil
	}

	level, err := parseLogLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	logOutput := cfg.LogOutput
	if logOutput == nil {
		logOutput = os.Stderr
	}

	return slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: level})), nil
}

func openStore(cfg Config) (core.Store, func() error, error) {
	if cfg.Store != nil {
		return cfg.Store, nil, nil
	}

	opener := cfg.OpenStore
	if opener == nil {
		dir := filepath.Dir(cfg.DBPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create db directory: %w", err)
		}
		opener = func(dsn string) (core.Store, error) {
			return store.NewSQLiteStore(dsn)
		}
	}

	dataStore, err := opener(cfg.DBPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}

	return dataStore, func() error {
		return dataStore.Close()
	}, nil
}

func composeCleanup(closers ...func() error) func() error {
	return func() error {
		for _, closer := range closers {
			if closer == nil {
				continue
			}
			if err := closer(); err != nil {
				return err
			}
		}
		return nil
	}
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
