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
	DBPath        string
	LogLevel      string
	LogOutput     io.Writer
	Logger        *slog.Logger
	Store         core.Store
	OpenStore     StoreOpener
	HTTPClient    *http.Client
	SQLiteOptions *store.SQLiteOptions
	Cleanup       func() error
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
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	logger, err := buildLogger(cfg)
	if err != nil {
		return nil, err
	}

	dataStore, ownedClose, err := openStore(cfg)
	if err != nil {
		return nil, err
	}

	return buildInstance(cfg, logger, dataStore, ownedClose), nil
}

func validateConfig(cfg Config) error {
	if cfg.Store != nil && cfg.OpenStore != nil {
		return fmt.Errorf("config conflict: Store and OpenStore are mutually exclusive")
	}
	if cfg.SQLiteOptions != nil && (cfg.Store != nil || cfg.OpenStore != nil) {
		return fmt.Errorf("config conflict: SQLiteOptions cannot be combined with Store or OpenStore")
	}
	if cfg.Logger != nil && (cfg.LogLevel != "" || cfg.LogOutput != nil) {
		return fmt.Errorf("config conflict: Logger cannot be combined with LogLevel or LogOutput")
	}
	if cfg.Store == nil && cfg.DBPath == "" {
		return fmt.Errorf("config error: DBPath is required when Store is not provided")
	}
	return nil
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
			return store.NewSQLiteStoreWithOptions(dsn, cfg.SQLiteOptions)
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

func buildInstance(cfg Config, logger *slog.Logger, dataStore core.Store, ownedClose func() error) *Instance {
	return &Instance{
		Service: core.New(dataStore, logger, serviceOptions(cfg)...),
		Store:   dataStore,
		Logger:  logger,
		Close:   composeCleanup(ownedClose, cfg.Cleanup),
	}
}

func serviceOptions(cfg Config) []core.Option {
	options := make([]core.Option, 0, 1)
	if cfg.HTTPClient != nil {
		options = append(options, core.WithHTTPClientWithUserAgent(cfg.HTTPClient))
	}
	return options
}

// composeCleanup runs cleanup callbacks in order and stops at the first error.
// Later cleanup functions are skipped once an earlier step fails so callers can
// rely on a deterministic first-failure result.
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
