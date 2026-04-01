package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

func newEntriesCmd(getService serviceGetter) *cobra.Command {
	var entriesFeedID int64
	var entriesLimit int
	var entriesJSON bool
	var entriesYAML bool
	var entriesUnread bool
	var entriesStarred bool
	var entriesTag string
	var entriesFormat string

	entriesCmd := &cobra.Command{
		Use:   "entries",
		Short: "List stored entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}

			filter := core.EntryFilter{
				Limit:       entriesLimit,
				UnreadOnly:  entriesUnread,
				StarredOnly: entriesStarred,
				Tag:         entriesTag,
			}
			if entriesFeedID > 0 {
				filter.FeedID = &entriesFeedID
			}

			entries, err := svc.ListEntries(cmd.Context(), filter)
			if err != nil {
				return err
			}

			if entriesJSON {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if entriesYAML {
				return writeYAML(cmd.OutOrStdout(), entries)
			}

			if entriesFormat == "markdown" {
				return renderEntriesMarkdown(cmd, entries)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK\tPUBLISHED")
			for _, e := range entries {
				pub := "-"
				if e.PublishedAt != nil {
					pub = e.PublishedAt.Format("2006-01-02 15:04")
				}
				fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link, pub)
			}
			return w.Flush()
		},
	}

	entriesCmd.Flags().Int64Var(&entriesFeedID, "feed-id", 0, "filter by feed ID")
	entriesCmd.Flags().IntVar(&entriesLimit, "limit", 20, "max entries to show")
	entriesCmd.Flags().BoolVar(&entriesJSON, "json", false, "output as JSON")
	entriesCmd.Flags().BoolVar(&entriesYAML, "yaml", false, "output as YAML")
	entriesCmd.Flags().BoolVar(&entriesUnread, "unread", false, "show only unread entries")
	entriesCmd.Flags().BoolVar(&entriesStarred, "starred", false, "show only starred entries")
	entriesCmd.Flags().StringVar(&entriesTag, "tag", "", "filter by feed tag")
	entriesCmd.Flags().StringVar(&entriesFormat, "format", "", "output format: markdown")
	entriesCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return entriesCmd
}

func renderEntriesMarkdown(cmd *cobra.Command, entries []*core.Entry) error {
	out := cmd.OutOrStdout()
	for i, e := range entries {
		if i > 0 {
			fmt.Fprintln(out, "---")
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "## %s\n\n", e.Title)
		if e.Link != "" {
			fmt.Fprintf(out, "URL: %s\n", e.Link)
		}
		if e.PublishedAt != nil {
			fmt.Fprintf(out, "Published: %s\n", e.PublishedAt.Format("2006-01-02 15:04"))
		}
		if e.Author != "" {
			fmt.Fprintf(out, "Author: %s\n", e.Author)
		}
		fmt.Fprintln(out)
		if e.Content != "" {
			fmt.Fprintln(out, e.Content)
		} else if e.Summary != "" {
			fmt.Fprintln(out, e.Summary)
		}
		fmt.Fprintln(out)
	}
	return nil
}
