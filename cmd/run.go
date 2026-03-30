package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// runInterval holds the --interval flag value controlling how frequently feeds
// are fetched in daemon mode. Defaults to 30 minutes.
var runInterval time.Duration

// runCmd implements "shu run".
//
// It starts a long-running daemon that periodically fetches all registered
// feeds at the configured interval. An initial fetch is performed immediately
// on startup, then subsequent fetches occur every --interval duration.
//
// The daemon handles graceful shutdown: it listens for SIGINT and SIGTERM
// signals (e.g. Ctrl+C) and exits cleanly after printing a shutdown message.
// Any in-progress fetch is cancelled via the context when a signal is received.
//
// Examples:
//
//	shu run                  # Fetch every 30 minutes (default)
//	shu run --interval 5m   # Fetch every 5 minutes
//	shu run --interval 1h   # Fetch every hour
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run daemon mode: fetch all feeds on an interval",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		fmt.Fprintf(cmd.OutOrStdout(), "Starting daemon (interval: %s). Press Ctrl+C to stop.\n", runInterval)

		// Fetch immediately on start so the user doesn't have to wait for
		// the first tick.
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
