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
	var statsJSON bool
	var statsYAML bool

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
			if handled, err := writeStructuredOutput(cmd.OutOrStdout(), stats, statsJSON, statsYAML); handled || err != nil {
				return err
			}
			return renderFeedStatsTable(cmd.OutOrStdout(), stats)
		},
	}

	addStructuredOutputFlags(statsCmd, &statsJSON, &statsYAML)
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
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d entries older than %s\n", n, cleanupOlderThan)
			return nil
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

			fmt.Fprintf(cmd.OutOrStdout(), "Starting daemon (interval: %s). Press Ctrl+C to stop.\n", runInterval)

			count, err := svc.FetchAll(ctx)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "fetch error: %v\n", err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
			}

			ticker := time.NewTicker(runInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					fmt.Fprintln(cmd.OutOrStdout(), "Shutting down...")
					return nil
				case <-ticker.C:
					count, err := svc.FetchAll(ctx)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "fetch error: %v\n", err)
						continue
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
				}
			}
		},
	}

	runCmd.Flags().DurationVar(&runInterval, "interval", 30*time.Minute, "fetch interval (e.g. 5m, 1h)")
	return runCmd
}
