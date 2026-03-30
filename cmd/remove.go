package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// removeCmd implements "shu remove <id>".
//
// It deletes the feed with the given numeric ID and all of its associated
// entries (via cascade delete in the database). The ID can be found using
// "shu list".
var removeCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a feed and its entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed ID: %w", err)
		}
		if err := svc.RemoveFeed(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed feed #%d\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
