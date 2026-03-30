package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var fetchFeedID int64

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch all feeds (or a specific feed)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if fetchFeedID > 0 {
			entries, err := svc.FetchFeed(ctx, fetchFeedID)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries from feed #%d\n", len(entries), fetchFeedID)
			return nil
		}

		count, err := svc.FetchAll(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
		return nil
	},
}

func init() {
	fetchCmd.Flags().Int64Var(&fetchFeedID, "feed-id", 0, "fetch a specific feed by ID")
	rootCmd.AddCommand(fetchCmd)
}
