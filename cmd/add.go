package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// addTitle holds the optional --title flag value. When non-empty, it overrides
	// the title extracted from the feed document.
	addTitle string
	// addJSON controls whether the output is formatted as JSON.
	addJSON bool
	// addYAML controls whether the output is formatted as YAML.
	addYAML bool
)

// addCmd implements "shu add <url>".
//
// It fetches the feed at the given URL to validate it, extracts metadata
// (title, site URL), and persists the feed record. If --title is provided,
// the user-supplied title is stored instead of the one from the feed document.
//
// Example:
//
//	shu add https://example.com/feed.xml
//	shu add https://example.com/feed.xml --title "My Blog"
var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a feed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		feed, err := svc.AddFeed(cmd.Context(), args[0], addTitle)
		if err != nil {
			return err
		}
		if addJSON {
			return writeJSON(cmd.OutOrStdout(), feed)
		}
		if addYAML {
			return writeYAML(cmd.OutOrStdout(), feed)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added feed #%d: %s (%s)\n", feed.ID, feed.Title, feed.URL)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", "override feed title")
	addCmd.Flags().BoolVar(&addJSON, "json", false, "output as JSON")
	addCmd.Flags().BoolVar(&addYAML, "yaml", false, "output as YAML")
	addCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	rootCmd.AddCommand(addCmd)
}
