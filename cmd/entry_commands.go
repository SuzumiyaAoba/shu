package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

func entryCommands(getService serviceGetter) []*cobra.Command {
	return withGroup("entries",
		newEntriesCmd(getService),
		newReadCmd(getService),
		newUnreadCmd(getService),
		newStarCmd(getService),
		newUnstarCmd(getService),
		newOpenCmd(getService),
		newSearchCmd(getService),
		newDuplicatesCmd(getService),
	)
}

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

func newStarCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "star <entry-id> [entry-id...]",
		Short: "Bookmark entries",
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
}

func newUnstarCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "unstar <entry-id> [entry-id...]",
		Short: "Remove bookmarks from entries",
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
}

func newOpenCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "open <entry-id>",
		Short: "Open an entry in the default browser",
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

			entry, err := svc.GetEntry(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("entry #%d: %w", id, err)
			}
			if entry.Link == "" {
				return fmt.Errorf("entry #%d has no link", id)
			}

			if err := svc.MarkEntryRead(cmd.Context(), id); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not mark entry as read: %v\n", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Opening %s\n", entry.Link)
			return openBrowser(entry.Link)
		},
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func newSearchCmd(getService serviceGetter) *cobra.Command {
	var searchLimit int
	var searchJSON bool
	var searchYAML bool

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search across entries",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")

			entries, err := svc.SearchEntries(cmd.Context(), query, searchLimit)
			if err != nil {
				return err
			}

			if searchJSON {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if searchYAML {
				return writeYAML(cmd.OutOrStdout(), entries)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK")
			for _, e := range entries {
				fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
			}
			return w.Flush()
		},
	}

	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "max results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	searchCmd.Flags().BoolVar(&searchYAML, "yaml", false, "output as YAML")
	searchCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return searchCmd
}

func newDuplicatesCmd(getService serviceGetter) *cobra.Command {
	var duplicatesJSON bool
	var duplicatesYAML bool

	duplicatesCmd := &cobra.Command{
		Use:   "duplicates <entry-id>",
		Short: "Find entries from other feeds with the same link",
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

			dupes, err := svc.FindDuplicateEntries(cmd.Context(), id)
			if err != nil {
				return err
			}

			if duplicatesJSON {
				return writeJSON(cmd.OutOrStdout(), dupes)
			}
			if duplicatesYAML {
				return writeYAML(cmd.OutOrStdout(), dupes)
			}

			if len(dupes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No duplicates found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK")
			for _, e := range dupes {
				fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
			}
			return w.Flush()
		},
	}

	duplicatesCmd.Flags().BoolVar(&duplicatesJSON, "json", false, "output as JSON")
	duplicatesCmd.Flags().BoolVar(&duplicatesYAML, "yaml", false, "output as YAML")
	duplicatesCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return duplicatesCmd
}
