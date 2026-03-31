package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// listJSON controls whether the output is formatted as JSON instead of a
// human-readable table.
var listJSON bool

// listCmd implements "shu list".
//
// It displays all registered feeds in a tabular format showing ID, title, URL,
// and the last-fetched timestamp. When --json is passed, it outputs the full
// feed data as a pretty-printed JSON array.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all feeds",
	RunE: func(cmd *cobra.Command, args []string) error {
		feeds, err := svc.ListFeeds(cmd.Context())
		if err != nil {
			return err
		}

		if listJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(feeds)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tURL\tFETCHED\tSTATUS")
		for _, f := range feeds {
			fetched := "-"
			if f.FetchedAt != nil {
				fetched = f.FetchedAt.Format("2006-01-02 15:04")
			}
			status := "ok"
			if f.Disabled {
				status = "disabled"
			} else if f.ErrorCount > 0 {
				status = fmt.Sprintf("err(%d)", f.ErrorCount)
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", f.ID, f.Title, f.URL, fetched, status)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}
