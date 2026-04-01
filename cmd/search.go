package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newSearchCmd(getService serviceGetter) *cobra.Command {
	var searchLimit int
	var searchJSON bool
	var searchYAML bool

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search across entries",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")

			entries, err := svc.SearchEntries(cmd.Context(), query, searchLimit)
			if err != nil {
				return err
			}

			if searchJSON {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if searchYAML {
				return writeYAML(cmd.OutOrStdout(), entries)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK")
			for _, e := range entries {
				fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
			}
			return w.Flush()
		},
	}

	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "max results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	searchCmd.Flags().BoolVar(&searchYAML, "yaml", false, "output as YAML")
	searchCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return searchCmd
}
