// Package cmd implements the CLI layer for the shu RSS aggregator using the
// Cobra command framework.
//
// This package is the CLI frontend. It builds a fresh command tree per
// execution and bootstraps a reusable runtime instance from the app package in
// PersistentPreRunE when no service is injected explicitly.
package cmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/SuzumiyaAoba/shu/app"
	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

type serviceGetter func() (*core.Service, error)

func newRootCmd(injected *core.Service) *cobra.Command {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".shu", "shu.db")

	var dbPath string
	var logLevel string
	var instance *app.Instance

	getService := func() (*core.Service, error) {
		if injected != nil {
			return injected, nil
		}
		if instance != nil && instance.Service != nil {
			return instance.Service, nil
		}
		return nil, errors.New("service not initialized")
	}

	rootCmd := &cobra.Command{
		Use:   "shu",
		Short: "RSS Aggregator CLI",
		Long:  "shu collects RSS feeds and stores entries in SQLite.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if injected != nil {
				return nil
			}

			var err error
			instance, err = app.Open(app.Config{
				DBPath:    dbPath,
				LogLevel:  logLevel,
				LogOutput: os.Stderr,
			})
			return err
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if injected != nil || instance == nil {
				return nil
			}

			err := instance.Close()
			instance = nil
			return err
		},
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	rootCmd.AddCommand(
		newAddCmd(getService),
		newCleanupCmd(getService),
		newDiscoverCmd(getService),
		newDisableCmd(getService),
		newDuplicatesCmd(getService),
		newEnableCmd(getService),
		newEntriesCmd(getService),
		newExportCmd(getService),
		newFetchCmd(getService),
		newImportCmd(getService),
		newListCmd(getService),
		newOpenCmd(getService),
		newReadCmd(getService),
		newRemoveCmd(getService),
		newRunCmd(getService),
		newSearchCmd(getService),
		newStarCmd(getService),
		newStatsCmd(getService),
		newTagCmd(getService),
		newTagsCmd(getService),
		newUnstarCmd(getService),
		newUnreadCmd(getService),
		newUntagCmd(getService),
		newUpdateCmd(getService),
	)

	return rootCmd
}

// Execute runs the root command and exits with code 1 on error.
// This is the single entry point called from main().
func Execute() {
	if err := newRootCmd(nil).Execute(); err != nil {
		os.Exit(1)
	}
}
