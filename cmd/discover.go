package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiscoverCmd(getService serviceGetter) *cobra.Command {
	var discoverJSON bool
	var discoverYAML bool

	discoverCmd := &cobra.Command{
		Use:   "discover <url>",
		Short: "Discover feed URLs from a web page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			feeds, err := svc.DiscoverFeeds(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if discoverJSON {
				return writeJSON(cmd.OutOrStdout(), feeds)
			}
			if discoverYAML {
				return writeYAML(cmd.OutOrStdout(), feeds)
			}

			if len(feeds) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No feeds found.")
				return nil
			}
			for _, u := range feeds {
				fmt.Fprintln(cmd.OutOrStdout(), u)
			}
			return nil
		},
	}

	discoverCmd.Flags().BoolVar(&discoverJSON, "json", false, "output as JSON")
	discoverCmd.Flags().BoolVar(&discoverYAML, "yaml", false, "output as YAML")
	discoverCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return discoverCmd
}
