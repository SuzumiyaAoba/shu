package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"gopkg.in/yaml.v3"
)

// parseIDArg parses a command-line argument as an int64 ID.
func parseIDArg(arg string) (int64, error) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID %q: %w", arg, err)
	}
	return id, nil
}

// parseIDArgs parses multiple command-line arguments as int64 IDs.
func parseIDArgs(args []string) ([]int64, error) {
	ids := make([]int64, len(args))
	for i, arg := range args {
		id, err := parseIDArg(arg)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func resolvePageOffset(limit, offset, page int) (int, error) {
	if page < 0 {
		return 0, fmt.Errorf("--page must be >= 0")
	}
	if offset < 0 {
		return 0, fmt.Errorf("--offset must be >= 0")
	}
	if page > 0 {
		return (page - 1) * limit, nil
	}
	return offset, nil
}

func writeEntryPageStructuredOutput(w io.Writer, page *core.EntryPage, pageInfo, outputJSON, outputYAML bool) (bool, error) {
	if outputJSON {
		return true, writeStructuredValue(w, page, page.Entries, pageInfo, outputJSON, outputYAML)
	}
	if outputYAML {
		return true, writeStructuredValue(w, page, page.Entries, pageInfo, outputJSON, outputYAML)
	}
	return false, nil
}

func writeEntryPageSummary(w io.Writer, page *core.EntryPage, noun string) error {
	fmt.Fprintf(w, "Showing %d/%d %s", len(page.Entries), page.TotalCount, noun)
	if page.HasMore {
		fmt.Fprintf(w, " (next offset: %d)", page.Offset+len(page.Entries))
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeStructuredOutput(w io.Writer, v any, outputJSON, outputYAML bool) (bool, error) {
	if outputJSON || outputYAML {
		return true, writeStructuredValue(w, v, v, true, outputJSON, outputYAML)
	}
	return false, nil
}

func writeStructuredValue(w io.Writer, fullValue, simpleValue any, includeFull, outputJSON, outputYAML bool) error {
	value := simpleValue
	if includeFull {
		value = fullValue
	}
	if outputJSON {
		return writeJSON(w, value)
	}
	if outputYAML {
		return writeYAML(w, value)
	}
	return nil
}

func renderFeedsTable(w io.Writer, feeds []*core.Feed) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tURL\tFETCHED\tSTATUS")
	for _, f := range feeds {
		fetched := "-"
		if f.FetchedAt != nil {
			fetched = f.FetchedAt.Format("2006-01-02 15:04")
		}
		status := feedStatus(f.Disabled, f.ErrorCount)
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n", f.ID, f.Title, f.URL, fetched, status)
	}
	return tw.Flush()
}

func renderEntriesTable(w io.Writer, entries []*core.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tFEED\tTITLE\tLINK\tPUBLISHED")
	for _, e := range entries {
		pub := "-"
		if e.PublishedAt != nil {
			pub = e.PublishedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(tw, "%d\t%d\t%s\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link, pub)
	}
	return tw.Flush()
}

func renderEntryLinksTable(w io.Writer, entries []*core.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tFEED\tTITLE\tLINK")
	for _, e := range entries {
		fmt.Fprintf(tw, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link)
	}
	return tw.Flush()
}

func renderTagsTable(w io.Writer, tags []core.Tag) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME")
	for _, t := range tags {
		fmt.Fprintf(tw, "%d\t%s\n", t.ID, t.Name)
	}
	return tw.Flush()
}

func renderFeedStatsTable(w io.Writer, stats []core.FeedStats) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tTOTAL\tUNREAD\tSTARRED\tFETCHED\tSTATUS")
	for _, s := range stats {
		fetched := "-"
		if s.FetchedAt != nil {
			fetched = s.FetchedAt.Format("2006-01-02 15:04")
		}
		status := feedStatus(s.Disabled, s.ErrorCount)
		fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%d\t%s\t%s\n",
			s.FeedID, s.Title, s.TotalCount, s.UnreadCount, s.StarredCount, fetched, status)
	}
	return tw.Flush()
}

// writeJSON encodes v as pretty-printed JSON to w.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// writeYAML encodes v as YAML to w.
func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(v)
}

// feedStatus returns a human-readable status string for a feed.
func feedStatus(disabled bool, errorCount int) string {
	if disabled {
		return "disabled"
	}
	if errorCount > 0 {
		return fmt.Sprintf("err(%d)", errorCount)
	}
	return "ok"
}
