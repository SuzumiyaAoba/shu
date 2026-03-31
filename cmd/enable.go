package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable <feed-id>",
	Short: "Re-enable a disabled feed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed ID: %w", err)
		}
		if err := svc.EnableFeed(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Enabled feed #%d\n", id)
		return nil
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable <feed-id>",
	Short: "Disable a feed (skip during fetch)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed ID: %w", err)
		}
		if err := svc.DisableFeed(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Disabled feed #%d\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
}
