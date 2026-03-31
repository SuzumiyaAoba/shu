package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	searchLimit int
	searchJSON  bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search across entries",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		entries, err := svc.SearchEntries(cmd.Context(), query, searchLimit)
		if err != nil {
			return err
		}

		if searchJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK")
		for _, e := range entries {
			fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
		}
		return w.Flush()
	},
}

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "max results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(searchCmd)
}
