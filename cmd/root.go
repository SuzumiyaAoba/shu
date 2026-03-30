package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
	"github.com/spf13/cobra"
)

var (
	dbPath   string
	logLevel string

	svc    *core.Service
	closer func() error
)

var rootCmd = &cobra.Command{
	Use:   "shu",
	Short: "RSS Aggregator CLI",
	Long:  "shu collects RSS feeds and stores entries in SQLite.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level := slog.LevelInfo
		switch logLevel {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create db directory: %w", err)
		}

		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		closer = s.Close

		svc = core.New(s, logger)
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if closer != nil {
			return closer()
		}
		return nil
	},
}

func init() {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".shu", "shu.db")

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
