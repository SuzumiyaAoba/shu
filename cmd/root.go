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
	"strings"
	"time"

	"github.com/SuzumiyaAoba/shu/app"
	"github.com/SuzumiyaAoba/shu/core"
	"github.com/SuzumiyaAoba/shu/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type coreServiceGetter func() (*core.Service, error)

func newRootCmd(injected *core.Service) *cobra.Command {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".shu", "shu.db")

	var configFile string
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
			initViper(configFile)
			// Apply config-file / env-var values for flags that were not
			// explicitly set on the command line.
			if !cmd.Root().PersistentFlags().Changed("db") {
				dbPath = viper.GetString("db")
			}
			if !cmd.Root().PersistentFlags().Changed("log-level") {
				logLevel = viper.GetString("log-level")
			}
			if !cmd.Root().PersistentFlags().Changed("sqlite-busy-timeout") {
				sqliteBusyTimeout = viper.GetDuration("sqlite-busy-timeout")
			}
			if !cmd.Root().PersistentFlags().Changed("sqlite-max-open-conns") {
				sqliteMaxOpenConns = viper.GetInt("sqlite-max-open-conns")
			}

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

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default ~/.config/shu/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().DurationVar(&sqliteBusyTimeout, "sqlite-busy-timeout", 0, "SQLite busy timeout (e.g. 5s)")
	rootCmd.PersistentFlags().IntVar(&sqliteMaxOpenConns, "sqlite-max-open-conns", 0, "SQLite max open connections")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress progress output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored table output")

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

// initViper configures viper to read from a config file and environment
// variables. It is called early in PersistentPreRunE so that subsequent
// viper.Get* calls reflect file / env overrides before flag defaults are used.
func initViper(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(filepath.Join(home, ".config", "shu"))
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	// SHU_DB, SHU_LOG_LEVEL, SHU_SQLITE_BUSY_TIMEOUT, …
	viper.SetEnvPrefix("SHU")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	_ = viper.ReadInConfig() // ignore missing file
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
