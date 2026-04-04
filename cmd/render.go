package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
)

func renderFeedsTable(w io.Writer, feeds []*core.Feed) error {
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

func renderEntriesTable(w io.Writer, entries []*core.Entry) error {
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

func renderEntryLinksTable(w io.Writer, entries []*core.Entry) error {
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

func renderTagsTable(w io.Writer, tags []core.Tag) error {
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

func renderFeedStatsTable(w io.Writer, stats []core.FeedStats) error {
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
