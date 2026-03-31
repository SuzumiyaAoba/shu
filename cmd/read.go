package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <entry-id>",
	Short: "Mark an entry as read",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid entry ID: %w", err)
		}
		if err := svc.MarkEntryRead(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Marked entry #%d as read\n", id)
		return nil
	},
}

var unreadCmd = &cobra.Command{
	Use:   "unread <entry-id>",
	Short: "Mark an entry as unread",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid entry ID: %w", err)
		}
		if err := svc.MarkEntryUnread(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Marked entry #%d as unread\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(unreadCmd)
}
