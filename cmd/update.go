package cmd

import (
	"fmt"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

func newUpdateCmd(getService serviceGetter) *cobra.Command {
	var updateTitle string
	var updateURL string

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a feed's title or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
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

	updateCmd.Flags().StringVar(&updateTitle, "title", "", "new title")
	updateCmd.Flags().StringVar(&updateURL, "url", "", "new URL")
	return updateCmd
}
