package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

var (
	// entriesFeedID filters entries to a specific feed when greater than zero.
	entriesFeedID int64
	// entriesLimit caps the number of entries displayed. Defaults to 20.
	entriesLimit int
	// entriesJSON controls whether output is formatted as JSON instead of a
	// human-readable table.
	entriesJSON bool
)

// entriesCmd implements "shu entries".
//
// It displays stored feed entries in reverse chronological order (newest
// first). The output can be filtered by feed ID and limited to a maximum
// number of rows. When --json is passed, entries are output as a
// pretty-printed JSON array.
//
// Examples:
//
//	shu entries                     # Show the 20 most recent entries
//	shu entries --feed-id 3         # Show entries from feed #3 only
//	shu entries --limit 50 --json   # Show 50 entries as JSON
var entriesCmd = &cobra.Command{
	Use:   "entries",
	Short: "List stored entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := core.EntryFilter{
			Limit: entriesLimit,
		}
		if entriesFeedID > 0 {
			filter.FeedID = &entriesFeedID
		}

		entries, err := svc.ListEntries(cmd.Context(), filter)
		if err != nil {
			return err
		}

		if entriesJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK\tPUBLISHED")
		for _, e := range entries {
			pub := "-"
			if e.PublishedAt != nil {
				pub = e.PublishedAt.Format("2006-01-02 15:04")
			}
			fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link, pub)
		}
		return w.Flush()
	},
}

func init() {
	entriesCmd.Flags().Int64Var(&entriesFeedID, "feed-id", 0, "filter by feed ID")
	entriesCmd.Flags().IntVar(&entriesLimit, "limit", 20, "max entries to show")
	entriesCmd.Flags().BoolVar(&entriesJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(entriesCmd)
}
