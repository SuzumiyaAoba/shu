package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	// listJSON controls whether the output is formatted as JSON instead of a
	// human-readable table.
	listJSON bool
	// listYAML controls whether the output is formatted as YAML.
	listYAML bool
)

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
			return writeJSON(cmd.OutOrStdout(), feeds)
		}
		if listYAML {
			return writeYAML(cmd.OutOrStdout(), feeds)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tURL\tFETCHED\tSTATUS")
		for _, f := range feeds {
			fetched := "-"
			if f.FetchedAt != nil {
				fetched = f.FetchedAt.Format("2006-01-02 15:04")
			}
			status := feedStatus(f.Disabled, f.ErrorCount)
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", f.ID, f.Title, f.URL, fetched, status)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	listCmd.Flags().BoolVar(&listYAML, "yaml", false, "output as YAML")
	rootCmd.AddCommand(listCmd)
}
