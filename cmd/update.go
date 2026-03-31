package cmd

import (
	"fmt"
	"strconv"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

var (
	updateTitle string
	updateURL   string
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a feed's title or URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed ID: %w", err)
		}

		update := core.FeedUpdate{}
		if cmd.Flags().Changed("title") {
			update.Title = &updateTitle
		}
		if cmd.Flags().Changed("url") {
			update.URL = &updateURL
		}

		if err := svc.UpdateFeed(cmd.Context(), id, update); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Updated feed #%d\n", id)
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "new title")
	updateCmd.Flags().StringVar(&updateURL, "url", "", "new URL")
	rootCmd.AddCommand(updateCmd)
}
