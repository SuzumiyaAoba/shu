package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// fetchFeedID holds the optional --feed-id flag value. When greater than zero,
	// only the specified feed is fetched instead of all feeds.
	fetchFeedID int64
	// fetchJSON controls whether the output is formatted as JSON.
	fetchJSON bool
	// fetchYAML controls whether the output is formatted as YAML.
	fetchYAML bool
)

// fetchCmd implements "shu fetch".
//
// By default it fetches all registered feeds, downloads their latest items,
// and stores any new entries. When --feed-id is provided, only that single
// feed is fetched.
//
// This is a one-shot operation: it runs once and exits. For periodic fetching,
// use "shu run" instead.
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
			if fetchJSON {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if fetchYAML {
				return writeYAML(cmd.OutOrStdout(), entries)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries from feed #%d\n", len(entries), fetchFeedID)
			return nil
		}

		count, err := svc.FetchAll(ctx)
		if err != nil {
			return err
		}
		if fetchJSON {
			return writeJSON(cmd.OutOrStdout(), map[string]int{"count": count})
		}
		if fetchYAML {
			return writeYAML(cmd.OutOrStdout(), map[string]int{"count": count})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
		return nil
	},
}

func init() {
	fetchCmd.Flags().Int64Var(&fetchFeedID, "feed-id", 0, "fetch a specific feed by ID")
	fetchCmd.Flags().BoolVar(&fetchJSON, "json", false, "output as JSON")
	fetchCmd.Flags().BoolVar(&fetchYAML, "yaml", false, "output as YAML")
	rootCmd.AddCommand(fetchCmd)
}
