package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var starCmd = &cobra.Command{
	Use:   "star <entry-id> [entry-id...]",
	Short: "Bookmark entries",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDArgs(args)
		if err != nil {
			return err
		}
		if len(ids) == 1 {
			if err := svc.StarEntry(cmd.Context(), ids[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Starred entry #%d\n", ids[0])
			return nil
		}
		if err := svc.StarEntries(cmd.Context(), ids); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Starred %d entries\n", len(ids))
		return nil
	},
}

var unstarCmd = &cobra.Command{
	Use:   "unstar <entry-id> [entry-id...]",
	Short: "Remove bookmarks from entries",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDArgs(args)
		if err != nil {
			return err
		}
		if len(ids) == 1 {
			if err := svc.UnstarEntry(cmd.Context(), ids[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unstarred entry #%d\n", ids[0])
			return nil
		}
		if err := svc.UnstarEntries(cmd.Context(), ids); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Unstarred %d entries\n", len(ids))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(starCmd)
	rootCmd.AddCommand(unstarCmd)
}
