package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type entryPageOutputOptions struct {
	PageInfo bool
	Output   structuredOutputOptions
	Noun     string
	Render   func(io.Writer, []*core.Entry) error
}

type structuredOutputOptions struct {
	JSON bool
	YAML bool
}

type paginationFlags struct {
	Limit    int
	Offset   int
	Page     int
	PageInfo bool
}

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

func (o structuredOutputOptions) write(w io.Writer, v any) (bool, error) {
	return writeStructuredOutput(w, v, o.JSON, o.YAML)
}

func (o structuredOutputOptions) renderOrWrite(w io.Writer, v any, render func() error) error {
	if handled, err := o.write(w, v); handled || err != nil {
		return err
	}
	return render()
}

func (f paginationFlags) resolveOffset() (int, error) {
	return resolvePageOffset(f.Limit, f.Offset, f.Page)
}

func writeEntryPageStructuredOutput(w io.Writer, page *core.EntryPage, pageInfo, outputJSON, outputYAML bool) (bool, error) {
	if !outputJSON && !outputYAML {
		return false, nil
	}
	return true, writeStructuredValue(w, page, page.Entries, pageInfo, outputJSON, outputYAML)
}

func writeEntryPageSummary(w io.Writer, page *core.EntryPage, noun string) error {
	if _, err := fmt.Fprintf(w, "Showing %d/%d %s", len(page.Entries), page.TotalCount, noun); err != nil {
		return err
	}
	if page.HasMore {
		if _, err := fmt.Fprintf(w, " (next offset: %d)", page.Offset+len(page.Entries)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func renderEntryPageOutput(cmd *cobra.Command, page *core.EntryPage, options entryPageOutputOptions) error {
	if handled, err := writeEntryPageStructuredOutput(cmd.OutOrStdout(), page, options.PageInfo, options.Output.JSON, options.Output.YAML); handled || err != nil {
		return err
	}
	if err := options.Render(cmd.OutOrStdout(), page.Entries); err != nil {
		return err
	}
	if options.PageInfo {
		return writeEntryPageSummary(cmd.OutOrStdout(), page, options.Noun)
	}
	return nil
}

func addStructuredOutputFlags(cmd *cobra.Command, outputJSON, outputYAML *bool) {
	cmd.Flags().BoolVar(outputJSON, "json", false, "output as JSON")
	cmd.Flags().BoolVar(outputYAML, "yaml", false, "output as YAML")
	cmd.MarkFlagsMutuallyExclusive("json", "yaml")
}

func addPaginationFlags(cmd *cobra.Command, flags *paginationFlags, limitUsage, offsetUsage string) {
	cmd.Flags().IntVar(&flags.Limit, "limit", 20, limitUsage)
	cmd.Flags().IntVar(&flags.Offset, "offset", 0, offsetUsage)
	cmd.Flags().IntVar(&flags.Page, "page", 0, "1-based page number (uses --limit)")
	cmd.Flags().BoolVar(&flags.PageInfo, "page-info", false, "include pagination metadata")
	cmd.MarkFlagsMutuallyExclusive("offset", "page")
}

func runSingleIDCommand[T any](cmd *cobra.Command, args []string, getService func() (T, error), action func(T, context.Context, int64) error, successFormat string) error {
	svc, id, err := serviceAndID(cmd, args, getService)
	if err != nil {
		return err
	}
	if err := action(svc, cmd.Context(), id); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), successFormat, id)
	return err
}

func serviceAndID[T any](cmd *cobra.Command, args []string, getService func() (T, error)) (T, int64, error) {
	var zero T

	svc, err := getService()
	if err != nil {
		return zero, 0, err
	}
	id, err := parseIDArg(args[0])
	if err != nil {
		return zero, 0, err
	}
	return svc, id, nil
}

func writeStructuredOutput(w io.Writer, v any, outputJSON, outputYAML bool) (bool, error) {
	if outputJSON || outputYAML {
		return true, writeStructuredValue(w, v, v, true, outputJSON, outputYAML)
	}
	return false, nil
}

func writeStderrf(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format+"\n", args...)
	return err
}

func writeWarning(w io.Writer, format string, args ...any) error {
	return writeStderrf(w, "warning: "+format, args...)
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
