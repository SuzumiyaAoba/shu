package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReadCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "read <entry-id> [entry-id...]",
		Short: "Mark entries as read",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			ids, err := parseIDArgs(args)
			if err != nil {
				return err
			}
			if len(ids) == 1 {
				if err := svc.MarkEntryRead(cmd.Context(), ids[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Marked entry #%d as read\n", ids[0])
				return nil
			}
			if err := svc.MarkEntriesRead(cmd.Context(), ids); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Marked %d entries as read\n", len(ids))
			return nil
		},
	}
}

func newUnreadCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "unread <entry-id> [entry-id...]",
		Short: "Mark entries as unread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			ids, err := parseIDArgs(args)
			if err != nil {
				return err
			}
			if len(ids) == 1 {
				if err := svc.MarkEntryUnread(cmd.Context(), ids[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Marked entry #%d as unread\n", ids[0])
				return nil
			}
			if err := svc.MarkEntriesUnread(cmd.Context(), ids); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Marked %d entries as unread\n", len(ids))
			return nil
		},
	}
}
