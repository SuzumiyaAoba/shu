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
	"time"

	"github.com/SuzumiyaAoba/shu/app"
	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
	"github.com/spf13/cobra"
)

type coreServiceGetter func() (*core.Service, error)

func newRootCmd(injected *core.Service) *cobra.Command {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".shu", "shu.db")

	var dbPath string
	var logLevel string
	var sqliteBusyTimeout time.Duration
	var sqliteMaxOpenConns int
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

	getFeedService := asFeedServiceGetter(getService)
	getEntryService := asEntryServiceGetter(getService)
	getTagService := asTagServiceGetter(getService)
	getMaintenanceService := asMaintenanceServiceGetter(getService)
	getOPMLService := asOPMLServiceGetter(getService)

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
				DBPath:        dbPath,
				LogLevel:      logLevel,
				LogOutput:     os.Stderr,
				SQLiteOptions: sqliteOptionsFromFlags(sqliteBusyTimeout, sqliteMaxOpenConns),
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

	rootCmd.AddGroup(
		&cobra.Group{ID: "feeds", Title: "Feed Commands"},
		&cobra.Group{ID: "entries", Title: "Entry Commands"},
		&cobra.Group{ID: "tags", Title: "Tag Commands"},
		&cobra.Group{ID: "maintenance", Title: "Maintenance Commands"},
		&cobra.Group{ID: "opml", Title: "Import/Export Commands"},
	)

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().DurationVar(&sqliteBusyTimeout, "sqlite-busy-timeout", 0, "SQLite busy timeout (e.g. 5s)")
	rootCmd.PersistentFlags().IntVar(&sqliteMaxOpenConns, "sqlite-max-open-conns", 0, "SQLite max open connections")

	rootCmd.AddCommand(feedCommands(getFeedService)...)
	rootCmd.AddCommand(entryCommands(getEntryService)...)
	rootCmd.AddCommand(tagCommands(getTagService)...)
	rootCmd.AddCommand(maintenanceCommands(getMaintenanceService)...)
	rootCmd.AddCommand(opmlCommands(getOPMLService)...)

	return rootCmd
}

func withGroup(groupID string, commands ...*cobra.Command) []*cobra.Command {
	for _, cmd := range commands {
		cmd.GroupID = groupID
	}
	return commands
}

// Execute runs the root command and exits with code 1 on error.
// This is the single entry point called from main().
func Execute() {
	if err := newRootCmd(nil).Execute(); err != nil {
		os.Exit(1)
	}
}

func asFeedServiceGetter(get coreServiceGetter) feedServiceGetter {
	return func() (feedService, error) {
		return get()
	}
}

func asEntryServiceGetter(get coreServiceGetter) entryServiceGetter {
	return func() (entryService, error) {
		return get()
	}
}

func asTagServiceGetter(get coreServiceGetter) tagServiceGetter {
	return func() (tagService, error) {
		return get()
	}
}

func asMaintenanceServiceGetter(get coreServiceGetter) maintenanceServiceGetter {
	return func() (maintenanceService, error) {
		return get()
	}
}

func asOPMLServiceGetter(get coreServiceGetter) opmlServiceGetter {
	return func() (opmlService, error) {
		return get()
	}
}

func sqliteOptionsFromFlags(busyTimeout time.Duration, maxOpenConns int) *store.SQLiteOptions {
	if busyTimeout <= 0 && maxOpenConns <= 0 {
		return nil
	}
	return &store.SQLiteOptions{
		BusyTimeout:  busyTimeout,
		MaxOpenConns: maxOpenConns,
	}
}
