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
	var quiet bool
	runtime := &rootRuntime{}
	getService := newCoreServiceGetter(injected, runtime)

	rootCmd := &cobra.Command{
		Use:   "shu",
		Short: "RSS Aggregator CLI",
		Long:  "shu collects RSS feeds and stores entries in SQLite.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if injected != nil {
				return nil
			}

			return runtime.open(app.Config{
				DBPath:        dbPath,
				LogLevel:      logLevel,
				LogOutput:     os.Stderr,
				SQLiteOptions: sqliteOptionsFromFlags(sqliteBusyTimeout, sqliteMaxOpenConns),
			})
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if injected != nil {
				return nil
			}
			return runtime.close()
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
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress progress output")

	rootCmd.AddCommand(feedCommands(adaptServiceGetter[feedService](getService))...)
	rootCmd.AddCommand(entryCommands(adaptServiceGetter[entryService](getService))...)
	rootCmd.AddCommand(tagCommands(adaptServiceGetter[tagService](getService))...)
	rootCmd.AddCommand(maintenanceCommands(adaptServiceGetter[maintenanceService](getService))...)
	rootCmd.AddCommand(opmlCommands(adaptServiceGetter[opmlService](getService))...)

	return rootCmd
}

type rootRuntime struct {
	instance *app.Instance
}

func (r *rootRuntime) open(cfg app.Config) error {
	instance, err := app.Open(cfg)
	if err != nil {
		return err
	}
	r.instance = instance
	return nil
}

func (r *rootRuntime) close() error {
	if r.instance == nil {
		return nil
	}
	err := r.instance.Close()
	r.instance = nil
	return err
}

func newCoreServiceGetter(injected *core.Service, runtime *rootRuntime) coreServiceGetter {
	return func() (*core.Service, error) {
		if injected != nil {
			return injected, nil
		}
		if runtime != nil && runtime.instance != nil && runtime.instance.Service != nil {
			return runtime.instance.Service, nil
		}
		return nil, errors.New("service not initialized")
	}
}

func adaptServiceGetter[T any](get coreServiceGetter) func() (T, error) {
	return func() (T, error) {
		var zero T

		svc, err := get()
		if err != nil {
			return zero, err
		}
		typed, ok := any(svc).(T)
		if !ok {
			return zero, errors.New("service type mismatch")
		}
		return typed, nil
	}
}

func withGroup(groupID string, commands ...*cobra.Command) []*cobra.Command {
	for _, cmd := range commands {
		cmd.GroupID = groupID
	}
	return commands
}

// Execute runs the root command and returns any execution error.
func Execute() error {
	return newRootCmd(nil).Execute()
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
