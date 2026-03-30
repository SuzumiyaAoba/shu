package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var runInterval time.Duration

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run daemon mode: fetch all feeds on an interval",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		fmt.Fprintf(cmd.OutOrStdout(), "Starting daemon (interval: %s). Press Ctrl+C to stop.\n", runInterval)

		// Fetch immediately on start
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

func init() {
	runCmd.Flags().DurationVar(&runInterval, "interval", 30*time.Minute, "fetch interval (e.g. 5m, 1h)")
	rootCmd.AddCommand(runCmd)
}
