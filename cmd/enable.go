package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable <feed-id>",
	Short: "Re-enable a disabled feed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0])
		if err != nil {
			return err
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
		id, err := parseIDArg(args[0])
		if err != nil {
			return err
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
