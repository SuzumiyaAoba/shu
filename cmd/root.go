// Package cmd implements the CLI layer for the shu RSS aggregator using the
// Cobra command framework.
//
// This package is the CLI frontend. It bootstraps a reusable runtime instance
// from the app package in PersistentPreRunE, makes the resulting service
// available to subcommands via the package-level svc variable, and closes the
// runtime in PersistentPostRunE.
package cmd

import (
	"os"
	"path/filepath"

	"github.com/SuzumiyaAoba/shu/app"
	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

var (
	// dbPath is the filesystem path to the SQLite database file.
	// Defaults to ~/.shu/shu.db and can be overridden with --db.
	dbPath string
	// logLevel controls the verbosity of structured log output to stderr.
	// Valid values: "debug", "info", "warn", "error". Defaults to "info".
	logLevel string

	// svc is the core service instance, initialized in PersistentPreRunE and
	// shared across all subcommands.
	svc *core.Service
	// closer holds the cleanup function (store.Close) to be called after the
	// subcommand finishes.
	closer func() error
)

// rootCmd is the top-level Cobra command. It defines global flags and
// lifecycle hooks that initialize and tear down the database connection.
var rootCmd = &cobra.Command{
	Use:   "shu",
	Short: "RSS Aggregator CLI",
	Long:  "shu collects RSS feeds and stores entries in SQLite.",
	// PersistentPreRunE runs before every subcommand. It initializes the
	// logger, opens the SQLite database (creating the directory if needed),
	// runs migrations, and constructs the core.Service.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		instance, err := app.Open(app.Config{
			DBPath:    dbPath,
			LogLevel:  logLevel,
			LogOutput: os.Stderr,
		})
		if err != nil {
			return err
		}
		closer = instance.Close
		svc = instance.Service
		return nil
	},
	// PersistentPostRunE runs after every subcommand to close the database.
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

// Execute runs the root command and exits with code 1 on error.
// This is the single entry point called from main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
