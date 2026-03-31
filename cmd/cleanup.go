package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var cleanupOlderThan time.Duration

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Delete old entries (starred entries are preserved)",
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := svc.CleanupEntries(cmd.Context(), cleanupOlderThan)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d entries older than %s\n", n, cleanupOlderThan)
		return nil
	},
}

func init() {
	cleanupCmd.Flags().DurationVar(&cleanupOlderThan, "older-than", 90*24*time.Hour, "delete entries older than this duration (e.g. 90d, 720h)")
	rootCmd.AddCommand(cleanupCmd)
}
