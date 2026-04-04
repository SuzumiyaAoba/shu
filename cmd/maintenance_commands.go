package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func maintenanceCommands(getService maintenanceServiceGetter) []*cobra.Command {
	return withGroup("maintenance",
		newStatsCmd(getService),
		newCleanupCmd(getService),
		newRunCmd(getService),
	)
}

func newStatsCmd(getService maintenanceServiceGetter) *cobra.Command {
	var output structuredOutputOptions

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show feed statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}

			stats, err := svc.FeedStatsAll(cmd.Context())
			if err != nil {
				return err
			}
			return output.renderOrWrite(cmd.OutOrStdout(), stats, func() error {
				return renderFeedStatsTable(cmd.OutOrStdout(), stats)
			})
		},
	}

	addStructuredOutputFlags(statsCmd, &output.JSON, &output.YAML)
	return statsCmd
}

func newCleanupCmd(getService maintenanceServiceGetter) *cobra.Command {
	var cleanupOlderThan time.Duration

	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete old entries (starred entries are preserved)",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			n, err := svc.CleanupEntries(cmd.Context(), cleanupOlderThan)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d entries older than %s\n", n, cleanupOlderThan)
			return err
		},
	}

	cleanupCmd.Flags().DurationVar(&cleanupOlderThan, "older-than", 90*24*time.Hour, "delete entries older than this duration (e.g. 90d, 720h)")
	return cleanupCmd
}

func newRunCmd(getService maintenanceServiceGetter) *cobra.Command {
	var runInterval time.Duration

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run daemon mode: fetch all feeds on an interval",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Starting daemon (interval: %s). Press Ctrl+C to stop.\n", runInterval); err != nil {
				return err
			}

			count, err := svc.FetchAll(ctx)
			if err != nil {
				if writeErr := writeStderrf(cmd.ErrOrStderr(), "fetch error: %v", err); writeErr != nil {
					return writeErr
				}
			} else {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count); err != nil {
					return err
				}
			}

			ticker := time.NewTicker(runInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Shutting down...")
					return err
				case <-ticker.C:
					count, err := svc.FetchAll(ctx)
					if err != nil {
						if writeErr := writeStderrf(cmd.ErrOrStderr(), "fetch error: %v", err); writeErr != nil {
							return writeErr
						}
						continue
					}
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count); err != nil {
						return err
					}
				}
			}
		},
	}

	runCmd.Flags().DurationVar(&runInterval, "interval", 30*time.Minute, "fetch interval (e.g. 5m, 1h)")
	return runCmd
}
