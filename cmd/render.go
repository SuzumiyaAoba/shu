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

// --- Generic table renderer ---

// tableDefinition describes the columns and row-conversion for a table.
type tableDefinition[T any] struct {
	headers []string
	toRow   func(T) []string
}

// renderTable dispatches to the styled or plain renderer depending on the writer.
func renderTable[T any](w io.Writer, items []T, def tableDefinition[T]) error {
	if useStyled(w) {
		return renderTableStyled(w, items, def)
	}
	return renderTablePlain(w, items, def)
}

func renderTablePlain[T any](w io.Writer, items []T, def tableDefinition[T]) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	header := ""
	for i, h := range def.headers {
		if i > 0 {
			header += "\t"
		}
		header += h
	}
	if _, err := fmt.Fprintln(tw, header); err != nil {
		return err
	}
	for _, item := range items {
		row := def.toRow(item)
		line := ""
		for i, cell := range row {
			if i > 0 {
				line += "\t"
			}
			line += cell
		}
		if _, err := fmt.Fprintln(tw, line); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderTableStyled[T any](w io.Writer, items []T, def tableDefinition[T]) error {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, def.toRow(item))
	}
	_, err := fmt.Fprintln(w, newStyledTable(def.headers...).Rows(rows...).Render())
	return err
}

// --- Table definitions ---

var feedTableDef = tableDefinition[*core.Feed]{
	headers: []string{"ID", "TITLE", "URL", "FETCHED", "STATUS"},
	toRow: func(f *core.Feed) []string {
		fetched := "-"
		if f.FetchedAt != nil {
			fetched = f.FetchedAt.Format("2006-01-02 15:04")
		}
		return []string{
			fmt.Sprintf("%d", f.ID), f.Title, f.URL, fetched, feedStatus(f.Disabled, f.ErrorCount),
		}
	},
}

var entryTableDef = tableDefinition[*core.Entry]{
	headers: []string{"ID", "FEED", "TITLE", "LINK", "PUBLISHED"},
	toRow: func(e *core.Entry) []string {
		pub := "-"
		if e.PublishedAt != nil {
			pub = e.PublishedAt.Format("2006-01-02 15:04")
		}
		return []string{
			fmt.Sprintf("%d", e.ID), fmt.Sprintf("%d", e.FeedID), e.Title, e.Link, pub,
		}
	},
}

var entryLinkTableDef = tableDefinition[*core.Entry]{
	headers: []string{"ID", "FEED", "TITLE", "LINK"},
	toRow: func(e *core.Entry) []string {
		return []string{fmt.Sprintf("%d", e.ID), fmt.Sprintf("%d", e.FeedID), e.Title, e.Link}
	},
}

var tagTableDef = tableDefinition[core.Tag]{
	headers: []string{"ID", "NAME"},
	toRow: func(t core.Tag) []string {
		return []string{fmt.Sprintf("%d", t.ID), t.Name}
	},
}

var feedStatsTableDef = tableDefinition[core.FeedStats]{
	headers: []string{"ID", "TITLE", "TOTAL", "UNREAD", "STARRED", "FETCHED", "STATUS"},
	toRow: func(s core.FeedStats) []string {
		fetched := "-"
		if s.FetchedAt != nil {
			fetched = s.FetchedAt.Format("2006-01-02 15:04")
		}
		return []string{
			fmt.Sprintf("%d", s.FeedID), s.Title,
			fmt.Sprintf("%d", s.TotalCount), fmt.Sprintf("%d", s.UnreadCount), fmt.Sprintf("%d", s.StarredCount),
			fetched, feedStatus(s.Disabled, s.ErrorCount),
		}
	},
}

// --- Public render functions ---

func renderFeedsTable(w io.Writer, feeds []*core.Feed) error {
	return renderTable(w, feeds, feedTableDef)
}

func renderEntriesTable(w io.Writer, entries []*core.Entry) error {
	return renderTable(w, entries, entryTableDef)
}

func renderEntryLinksTable(w io.Writer, entries []*core.Entry) error {
	return renderTable(w, entries, entryLinkTableDef)
}

func renderTagsTable(w io.Writer, tags []core.Tag) error {
	return renderTable(w, tags, tagTableDef)
}

func renderFeedStatsTable(w io.Writer, stats []core.FeedStats) error {
	return renderTable(w, stats, feedStatsTableDef)
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
