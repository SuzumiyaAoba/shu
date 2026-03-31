package cmd

import (
	"fmt"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var duplicatesCmd = &cobra.Command{
	Use:   "duplicates <entry-id>",
	Short: "Find entries from other feeds with the same link",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid entry ID: %w", err)
		}

		dupes, err := svc.FindDuplicateEntries(cmd.Context(), id)
		if err != nil {
			return err
		}

		if len(dupes) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No duplicates found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK")
		for _, e := range dupes {
			fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(duplicatesCmd)
}
