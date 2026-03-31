package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	statsJSON bool
	statsYAML bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show feed statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := svc.FeedStatsAll(cmd.Context())
		if err != nil {
			return err
		}

		if statsJSON {
			return writeJSON(cmd.OutOrStdout(), stats)
		}
		if statsYAML {
			return writeYAML(cmd.OutOrStdout(), stats)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tTOTAL\tUNREAD\tSTARRED\tFETCHED\tSTATUS")
		for _, s := range stats {
			fetched := "-"
			if s.FetchedAt != nil {
				fetched = s.FetchedAt.Format("2006-01-02 15:04")
			}
			status := feedStatus(s.Disabled, s.ErrorCount)
			fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%d\t%s\t%s\n",
				s.FeedID, s.Title, s.TotalCount, s.UnreadCount, s.StarredCount, fetched, status)
		}
		return w.Flush()
	},
}

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output as JSON")
	statsCmd.Flags().BoolVar(&statsYAML, "yaml", false, "output as YAML")
	rootCmd.AddCommand(statsCmd)
}
