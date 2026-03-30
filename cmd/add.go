package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addTitle string

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
