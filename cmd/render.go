package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/model"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

// noColorFlag reads the --no-color persistent flag from the root command.
// It returns false when the flag is absent (e.g. in tests without a full CLI).
func noColorFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("no-color")
	return v
}

// useStyled returns true when the writer is a TTY and --no-color is not set.
func useStyled(w io.Writer, noColor bool) bool {
	return !noColor && isTTY(w)
}

// --- Generic table renderer ---

// tableDefinition describes the columns and row-conversion for a table.
type tableDefinition[T any] struct {
	headers []string
	toRow   func(T) []string
}

// renderTable dispatches to the styled or plain renderer depending on the writer.
func renderTable[T any](w io.Writer, items []T, def tableDefinition[T], noColor bool) error {
	if useStyled(w, noColor) {
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

var feedTableDef = tableDefinition[*model.Feed]{
	headers: []string{"ID", "TITLE", "URL", "FETCHED", "STATUS"},
	toRow: func(f *model.Feed) []string {
		fetched := "-"
		if f.FetchedAt != nil {
			fetched = f.FetchedAt.Format("2006-01-02 15:04")
		}
		return []string{
			fmt.Sprintf("%d", f.ID), f.Title, f.URL, fetched, feedStatus(f.Disabled, f.ErrorCount),
		}
	},
}

var entryTableDef = tableDefinition[*model.Entry]{
	headers: []string{"ID", "FEED", "TITLE", "LINK", "PUBLISHED"},
	toRow: func(e *model.Entry) []string {
		pub := "-"
		if e.PublishedAt != nil {
			pub = e.PublishedAt.Format("2006-01-02 15:04")
		}
		return []string{
			fmt.Sprintf("%d", e.ID), fmt.Sprintf("%d", e.FeedID), e.Title, e.Link, pub,
		}
	},
}

var entryLinkTableDef = tableDefinition[*model.Entry]{
	headers: []string{"ID", "FEED", "TITLE", "LINK"},
	toRow: func(e *model.Entry) []string {
		return []string{fmt.Sprintf("%d", e.ID), fmt.Sprintf("%d", e.FeedID), e.Title, e.Link}
	},
}

var tagTableDef = tableDefinition[model.Tag]{
	headers: []string{"ID", "NAME"},
	toRow: func(t model.Tag) []string {
		return []string{fmt.Sprintf("%d", t.ID), t.Name}
	},
}

var feedStatsTableDef = tableDefinition[model.FeedStats]{
	headers: []string{"ID", "TITLE", "TOTAL", "UNREAD", "STARRED", "FETCHED", "STATUS"},
	toRow: func(s model.FeedStats) []string {
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

func renderFeedsTable(w io.Writer, feeds []*model.Feed, noColor bool) error {
	return renderTable(w, feeds, feedTableDef, noColor)
}

func renderEntriesTable(w io.Writer, entries []*model.Entry, noColor bool) error {
	return renderTable(w, entries, entryTableDef, noColor)
}

func renderEntryLinksTable(w io.Writer, entries []*model.Entry, noColor bool) error {
	return renderTable(w, entries, entryLinkTableDef, noColor)
}

func renderTagsTable(w io.Writer, tags []model.Tag, noColor bool) error {
	return renderTable(w, tags, tagTableDef, noColor)
}

func renderFeedStatsTable(w io.Writer, stats []model.FeedStats, noColor bool) error {
	return renderTable(w, stats, feedStatsTableDef, noColor)
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
