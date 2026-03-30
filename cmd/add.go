package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addTitle holds the optional --title flag value. When non-empty, it overrides
// the title extracted from the feed document.
var addTitle string

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
		fmt.Fprintf(cmd.OutOrStdout(), "Added feed #%d: %s (%s)\n", feed.ID, feed.Title, feed.URL)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", "override feed title")
	rootCmd.AddCommand(addCmd)
}
