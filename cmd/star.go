package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var starCmd = &cobra.Command{
	Use:   "star <entry-id>",
	Short: "Bookmark an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0])
		if err != nil {
			return err
		}
		if err := svc.StarEntry(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Starred entry #%d\n", id)
		return nil
	},
}

var unstarCmd = &cobra.Command{
	Use:   "unstar <entry-id>",
	Short: "Remove bookmark from an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0])
		if err != nil {
			return err
		}
		if err := svc.UnstarEntry(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Unstarred entry #%d\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(starCmd)
	rootCmd.AddCommand(unstarCmd)
}
