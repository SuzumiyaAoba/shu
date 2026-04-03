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

func entryCommands(getService entryServiceGetter) []*cobra.Command {
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

func newEntriesCmd(getService entryServiceGetter) *cobra.Command {
	var entriesFeedID int64
	var entriesLimit int
	var entriesOffset int
	var entriesPage int
	var entriesPageInfo bool
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
			offset, err := resolvePageOffset(entriesLimit, entriesOffset, entriesPage)
			if err != nil {
				return err
			}

			filter := core.EntryFilter{
				Limit:       entriesLimit,
				Offset:      offset,
				UnreadOnly:  entriesUnread,
				StarredOnly: entriesStarred,
				Tag:         entriesTag,
			}
			if entriesFeedID > 0 {
				filter.FeedID = &entriesFeedID
			}

			page, err := svc.ListEntriesPage(cmd.Context(), filter)
			if err != nil {
				return err
			}

			if handled, err := writeEntryPageStructuredOutput(cmd.OutOrStdout(), page, entriesPageInfo, entriesJSON, entriesYAML); handled || err != nil {
				return err
			}

			if entriesFormat == "markdown" {
				if err := renderEntriesMarkdown(cmd, page.Entries); err != nil {
					return err
				}
			} else {
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK\tPUBLISHED")
				for _, e := range page.Entries {
					pub := "-"
					if e.PublishedAt != nil {
						pub = e.PublishedAt.Format("2006-01-02 15:04")
					}
					fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link, pub)
				}
				if err := w.Flush(); err != nil {
					return err
				}
			}

			if entriesPageInfo {
				return writeEntryPageSummary(cmd.OutOrStdout(), page, "entries")
			}
			return nil
		},
	}

	entriesCmd.Flags().Int64Var(&entriesFeedID, "feed-id", 0, "filter by feed ID")
	entriesCmd.Flags().IntVar(&entriesLimit, "limit", 20, "max entries to show")
	entriesCmd.Flags().IntVar(&entriesOffset, "offset", 0, "skip the first N entries")
	entriesCmd.Flags().IntVar(&entriesPage, "page", 0, "1-based page number (uses --limit)")
	entriesCmd.Flags().BoolVar(&entriesPageInfo, "page-info", false, "include pagination metadata")
	entriesCmd.Flags().BoolVar(&entriesJSON, "json", false, "output as JSON")
	entriesCmd.Flags().BoolVar(&entriesYAML, "yaml", false, "output as YAML")
	entriesCmd.Flags().BoolVar(&entriesUnread, "unread", false, "show only unread entries")
	entriesCmd.Flags().BoolVar(&entriesStarred, "starred", false, "show only starred entries")
	entriesCmd.Flags().StringVar(&entriesTag, "tag", "", "filter by feed tag")
	entriesCmd.Flags().StringVar(&entriesFormat, "format", "", "output format: markdown")
	entriesCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	entriesCmd.MarkFlagsMutuallyExclusive("offset", "page")
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

func newReadCmd(getService entryServiceGetter) *cobra.Command {
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

func newUnreadCmd(getService entryServiceGetter) *cobra.Command {
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

func newStarCmd(getService entryServiceGetter) *cobra.Command {
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

func newUnstarCmd(getService entryServiceGetter) *cobra.Command {
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

func newOpenCmd(getService entryServiceGetter) *cobra.Command {
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

func newSearchCmd(getService entryServiceGetter) *cobra.Command {
	var searchLimit int
	var searchOffset int
	var searchPage int
	var searchPageInfo bool
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
			offset, err := resolvePageOffset(searchLimit, searchOffset, searchPage)
			if err != nil {
				return err
			}

			page, err := svc.SearchEntriesPage(cmd.Context(), query, searchLimit, offset)
			if err != nil {
				return err
			}

			if handled, err := writeEntryPageStructuredOutput(cmd.OutOrStdout(), page, searchPageInfo, searchJSON, searchYAML); handled || err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tFEED\tTITLE\tLINK")
			for _, e := range page.Entries {
				fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if searchPageInfo {
				return writeEntryPageSummary(cmd.OutOrStdout(), page, "results")
			}
			return nil
		},
	}

	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "max results")
	searchCmd.Flags().IntVar(&searchOffset, "offset", 0, "skip the first N results")
	searchCmd.Flags().IntVar(&searchPage, "page", 0, "1-based page number (uses --limit)")
	searchCmd.Flags().BoolVar(&searchPageInfo, "page-info", false, "include pagination metadata")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	searchCmd.Flags().BoolVar(&searchYAML, "yaml", false, "output as YAML")
	searchCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	searchCmd.MarkFlagsMutuallyExclusive("offset", "page")
	return searchCmd
}

func newDuplicatesCmd(getService entryServiceGetter) *cobra.Command {
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
