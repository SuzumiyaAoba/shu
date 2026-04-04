package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// noColor disables lipgloss styling even on a TTY.
// Set via the --no-color global flag.
var noColor bool

// useStyled returns true when the writer is a TTY and --no-color is not set.
func useStyled(w io.Writer) bool {
	return !noColor && isTTY(w)
}

// --- Feed table ---

func renderFeedsTable(w io.Writer, feeds []*core.Feed) error {
	if useStyled(w) {
		return renderFeedsTableStyled(w, feeds)
	}
	return renderFeedsTablePlain(w, feeds)
}

func renderFeedsTablePlain(w io.Writer, feeds []*core.Feed) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tTITLE\tURL\tFETCHED\tSTATUS"); err != nil {
		return err
	}
	for _, f := range feeds {
		fetched := "-"
		if f.FetchedAt != nil {
			fetched = f.FetchedAt.Format("2006-01-02 15:04")
		}
		status := feedStatus(f.Disabled, f.ErrorCount)
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n", f.ID, f.Title, f.URL, fetched, status); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderFeedsTableStyled(w io.Writer, feeds []*core.Feed) error {
	rows := make([][]string, 0, len(feeds))
	for _, f := range feeds {
		fetched := "-"
		if f.FetchedAt != nil {
			fetched = f.FetchedAt.Format("2006-01-02 15:04")
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", f.ID), f.Title, f.URL, fetched, feedStatus(f.Disabled, f.ErrorCount),
		})
	}
	_, err := fmt.Fprintln(w, newStyledTable("ID", "TITLE", "URL", "FETCHED", "STATUS").Rows(rows...).Render())
	return err
}

// --- Entries table ---

func renderEntriesTable(w io.Writer, entries []*core.Entry) error {
	if useStyled(w) {
		return renderEntriesTableStyled(w, entries)
	}
	return renderEntriesTablePlain(w, entries)
}

func renderEntriesTablePlain(w io.Writer, entries []*core.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tFEED\tTITLE\tLINK\tPUBLISHED"); err != nil {
		return err
	}
	for _, e := range entries {
		pub := "-"
		if e.PublishedAt != nil {
			pub = e.PublishedAt.Format("2006-01-02 15:04")
		}
		if _, err := fmt.Fprintf(tw, "%d\t%d\t%s\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link, pub); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderEntriesTableStyled(w io.Writer, entries []*core.Entry) error {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		pub := "-"
		if e.PublishedAt != nil {
			pub = e.PublishedAt.Format("2006-01-02 15:04")
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", e.ID), fmt.Sprintf("%d", e.FeedID), e.Title, e.Link, pub,
		})
	}
	_, err := fmt.Fprintln(w, newStyledTable("ID", "FEED", "TITLE", "LINK", "PUBLISHED").Rows(rows...).Render())
	return err
}

// --- Entry links table ---

func renderEntryLinksTable(w io.Writer, entries []*core.Entry) error {
	if useStyled(w) {
		return renderEntryLinksTableStyled(w, entries)
	}
	return renderEntryLinksTablePlain(w, entries)
}

func renderEntryLinksTablePlain(w io.Writer, entries []*core.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tFEED\tTITLE\tLINK"); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(tw, "%d\t%d\t%s\t%s\n", e.ID, e.FeedID, e.Title, e.Link); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderEntryLinksTableStyled(w io.Writer, entries []*core.Entry) error {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{fmt.Sprintf("%d", e.ID), fmt.Sprintf("%d", e.FeedID), e.Title, e.Link})
	}
	_, err := fmt.Fprintln(w, newStyledTable("ID", "FEED", "TITLE", "LINK").Rows(rows...).Render())
	return err
}

// --- Tags table ---

func renderTagsTable(w io.Writer, tags []core.Tag) error {
	if useStyled(w) {
		return renderTagsTableStyled(w, tags)
	}
	return renderTagsTablePlain(w, tags)
}

func renderTagsTablePlain(w io.Writer, tags []core.Tag) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tNAME"); err != nil {
		return err
	}
	for _, t := range tags {
		if _, err := fmt.Fprintf(tw, "%d\t%s\n", t.ID, t.Name); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderTagsTableStyled(w io.Writer, tags []core.Tag) error {
	rows := make([][]string, 0, len(tags))
	for _, t := range tags {
		rows = append(rows, []string{fmt.Sprintf("%d", t.ID), t.Name})
	}
	_, err := fmt.Fprintln(w, newStyledTable("ID", "NAME").Rows(rows...).Render())
	return err
}

// --- Feed stats table ---

func renderFeedStatsTable(w io.Writer, stats []core.FeedStats) error {
	if useStyled(w) {
		return renderFeedStatsTableStyled(w, stats)
	}
	return renderFeedStatsTablePlain(w, stats)
}

func renderFeedStatsTablePlain(w io.Writer, stats []core.FeedStats) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tTITLE\tTOTAL\tUNREAD\tSTARRED\tFETCHED\tSTATUS"); err != nil {
		return err
	}
	for _, s := range stats {
		fetched := "-"
		if s.FetchedAt != nil {
			fetched = s.FetchedAt.Format("2006-01-02 15:04")
		}
		status := feedStatus(s.Disabled, s.ErrorCount)
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%d\t%s\t%s\n",
			s.FeedID, s.Title, s.TotalCount, s.UnreadCount, s.StarredCount, fetched, status); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderFeedStatsTableStyled(w io.Writer, stats []core.FeedStats) error {
	rows := make([][]string, 0, len(stats))
	for _, s := range stats {
		fetched := "-"
		if s.FetchedAt != nil {
			fetched = s.FetchedAt.Format("2006-01-02 15:04")
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", s.FeedID), s.Title,
			fmt.Sprintf("%d", s.TotalCount), fmt.Sprintf("%d", s.UnreadCount), fmt.Sprintf("%d", s.StarredCount),
			fetched, feedStatus(s.Disabled, s.ErrorCount),
		})
	}
	_, err := fmt.Fprintln(w, newStyledTable("ID", "TITLE", "TOTAL", "UNREAD", "STARRED", "FETCHED", "STATUS").Rows(rows...).Render())
	return err
}

// --- Shared table factory ---

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func newStyledTable(headers ...string) *table.Table {
	return table.New().
		Headers(headers...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})
}
