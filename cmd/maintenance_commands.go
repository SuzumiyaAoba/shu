package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func maintenanceCommands(getService serviceGetter) []*cobra.Command {
	return withGroup("maintenance",
		newStatsCmd(getService),
		newCleanupCmd(getService),
		newRunCmd(getService),
	)
}

func newStatsCmd(getService serviceGetter) *cobra.Command {
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

			if statsJSON {
				return writeJSON(cmd.OutOrStdout(), stats)
			}
			if statsYAML {
				return writeYAML(cmd.OutOrStdout(), stats)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tTOTAL\tUNREAD\tSTARRED\tFETCHED\tSTATUS")
			for _, s := range stats {
				fetched := "-"
				if s.FetchedAt != nil {
					fetched = s.FetchedAt.Format("2006-01-02 15:04")
				}
				status := feedStatus(s.Disabled, s.ErrorCount)
				fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%d\t%s\t%s\n",
					s.FeedID, s.Title, s.TotalCount, s.UnreadCount, s.StarredCount, fetched, status)
			}
			return w.Flush()
		},
	}

	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output as JSON")
	statsCmd.Flags().BoolVar(&statsYAML, "yaml", false, "output as YAML")
	statsCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return statsCmd
}

func newCleanupCmd(getService serviceGetter) *cobra.Command {
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

func newRunCmd(getService serviceGetter) *cobra.Command {
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
