package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

type entryMutationSpec struct {
	runSingle   func(entryService, context.Context, int64) error
	runBatch    func(entryService, context.Context, []int64) error
	singleLabel string
	batchLabel  string
}

func entryCommands(getService entryServiceGetter) []*cobra.Command {
	return withGroup("entries",
		newEntriesCmd(getService),
		newReadCmd(getService),
		newUnreadCmd(getService),
		newStarCmd(getService),
		newUnstarCmd(getService),
		newOpenCmd(getService, openBrowser),
		newSearchCmd(getService),
		newDuplicatesCmd(getService),
	)
}

func newEntriesCmd(getService entryServiceGetter) *cobra.Command {
	var entriesFeedID int64
	var pageFlags paginationFlags
	var output structuredOutputOptions
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
			offset, err := pageFlags.resolveOffset()
			if err != nil {
				return err
			}

			filter := core.EntryFilter{
				Limit:       pageFlags.Limit,
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

			render := renderEntriesTable
			if entriesFormat == "markdown" {
				render = func(w io.Writer, entries []*core.Entry) error {
					return renderEntriesMarkdown(cmd, entries)
				}
			}
			return renderEntryPageOutput(cmd, page, entryPageOutputOptions{
				PageInfo: pageFlags.PageInfo,
				Output:   output,
				Noun:     "entries",
				Render:   render,
			})
		},
	}

	entriesCmd.Flags().Int64Var(&entriesFeedID, "feed-id", 0, "filter by feed ID")
	addPaginationFlags(entriesCmd, &pageFlags, "max entries to show", "skip the first N entries")
	addStructuredOutputFlags(entriesCmd, &output.JSON, &output.YAML)
	entriesCmd.Flags().BoolVar(&entriesUnread, "unread", false, "show only unread entries")
	entriesCmd.Flags().BoolVar(&entriesStarred, "starred", false, "show only starred entries")
	entriesCmd.Flags().StringVar(&entriesTag, "tag", "", "filter by feed tag")
	entriesCmd.Flags().StringVar(&entriesFormat, "format", "", "output format: markdown")
	return entriesCmd
}

func renderEntriesMarkdown(cmd *cobra.Command, entries []*core.Entry) error {
	out := cmd.OutOrStdout()
	for i, e := range entries {
		if i > 0 {
			if _, err := fmt.Fprintln(out, "---"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(out, "## %s\n\n", e.Title); err != nil {
			return err
		}
		if e.Link != "" {
			if _, err := fmt.Fprintf(out, "URL: %s\n", e.Link); err != nil {
				return err
			}
		}
		if e.PublishedAt != nil {
			if _, err := fmt.Fprintf(out, "Published: %s\n", e.PublishedAt.Format("2006-01-02 15:04")); err != nil {
				return err
			}
		}
		if e.Author != "" {
			if _, err := fmt.Fprintf(out, "Author: %s\n", e.Author); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		if e.Content != "" {
			if _, err := fmt.Fprintln(out, e.Content); err != nil {
				return err
			}
		} else if e.Summary != "" {
			if _, err := fmt.Fprintln(out, e.Summary); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}
	return nil
}

func newReadCmd(getService entryServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "read <entry-id> [entry-id...]",
		Short: "Mark entries as read",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntryMutation(cmd, args, getService, entryMutationSpec{
				runSingle:   func(svc entryService, ctx context.Context, id int64) error { return svc.MarkEntryRead(ctx, id) },
				runBatch:    func(svc entryService, ctx context.Context, ids []int64) error { return svc.MarkEntriesRead(ctx, ids) },
				singleLabel: "Marked entry #%d as read\n",
				batchLabel:  "Marked %d entries as read\n",
			})
		},
	}
}

func newUnreadCmd(getService entryServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "unread <entry-id> [entry-id...]",
		Short: "Mark entries as unread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntryMutation(cmd, args, getService, entryMutationSpec{
				runSingle:   func(svc entryService, ctx context.Context, id int64) error { return svc.MarkEntryUnread(ctx, id) },
				runBatch:    func(svc entryService, ctx context.Context, ids []int64) error { return svc.MarkEntriesUnread(ctx, ids) },
				singleLabel: "Marked entry #%d as unread\n",
				batchLabel:  "Marked %d entries as unread\n",
			})
		},
	}
}

func newStarCmd(getService entryServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "star <entry-id> [entry-id...]",
		Short: "Bookmark entries",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntryMutation(cmd, args, getService, entryMutationSpec{
				runSingle:   func(svc entryService, ctx context.Context, id int64) error { return svc.StarEntry(ctx, id) },
				runBatch:    func(svc entryService, ctx context.Context, ids []int64) error { return svc.StarEntries(ctx, ids) },
				singleLabel: "Starred entry #%d\n",
				batchLabel:  "Starred %d entries\n",
			})
		},
	}
}

func newUnstarCmd(getService entryServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "unstar <entry-id> [entry-id...]",
		Short: "Remove bookmarks from entries",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntryMutation(cmd, args, getService, entryMutationSpec{
				runSingle:   func(svc entryService, ctx context.Context, id int64) error { return svc.UnstarEntry(ctx, id) },
				runBatch:    func(svc entryService, ctx context.Context, ids []int64) error { return svc.UnstarEntries(ctx, ids) },
				singleLabel: "Unstarred entry #%d\n",
				batchLabel:  "Unstarred %d entries\n",
			})
		},
	}
}

func newOpenCmd(getService entryServiceGetter, openURL func(string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "open <entry-id>",
		Short: "Open an entry in the default browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, id, err := serviceAndID(args, getService)
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
				if writeErr := writeWarning(cmd.ErrOrStderr(), "could not mark entry as read: %v", err); writeErr != nil {
					return writeErr
				}
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Opening %s\n", entry.Link); err != nil {
				return err
			}
			return openURL(entry.Link)
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
	var pageFlags paginationFlags
	var output structuredOutputOptions

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
			offset, err := pageFlags.resolveOffset()
			if err != nil {
				return err
			}

			page, err := svc.SearchEntriesPage(cmd.Context(), query, pageFlags.Limit, offset)
			if err != nil {
				return err
			}

			return renderEntryPageOutput(cmd, page, entryPageOutputOptions{
				PageInfo: pageFlags.PageInfo,
				Output:   output,
				Noun:     "results",
				Render:   renderEntryLinksTable,
			})
		},
	}

	addPaginationFlags(searchCmd, &pageFlags, "max results", "skip the first N results")
	addStructuredOutputFlags(searchCmd, &output.JSON, &output.YAML)
	return searchCmd
}

func newDuplicatesCmd(getService entryServiceGetter) *cobra.Command {
	var output structuredOutputOptions

	duplicatesCmd := &cobra.Command{
		Use:   "duplicates <entry-id>",
		Short: "Find entries from other feeds with the same link",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, id, err := serviceAndID(args, getService)
			if err != nil {
				return err
			}

			dupes, err := svc.FindDuplicateEntries(cmd.Context(), id)
			if err != nil {
				return err
			}
			return output.renderOrWrite(cmd.OutOrStdout(), dupes, func() error {
				if len(dupes) == 0 {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "No duplicates found.")
					return err
				}
				return renderEntryLinksTable(cmd.OutOrStdout(), dupes)
			})
		},
	}

	addStructuredOutputFlags(duplicatesCmd, &output.JSON, &output.YAML)
	return duplicatesCmd
}

func runEntryMutation(cmd *cobra.Command, args []string, getService entryServiceGetter, spec entryMutationSpec) error {
	svc, err := getService()
	if err != nil {
		return err
	}
	ids, err := parseIDArgs(args)
	if err != nil {
		return err
	}
	if len(ids) == 1 {
		if err := spec.runSingle(svc, cmd.Context(), ids[0]); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), spec.singleLabel, ids[0])
		return err
	}
	if err := spec.runBatch(svc, cmd.Context(), ids); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), spec.batchLabel, len(ids))
	return err
}
