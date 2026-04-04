package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type structuredOutputOptions struct {
	JSON bool
	YAML bool
}

// encode writes v as JSON or YAML if enabled, returning true if output was handled.
func (o structuredOutputOptions) encode(w io.Writer, v any) (bool, error) {
	if o.JSON {
		return true, writeJSON(w, v)
	}
	if o.YAML {
		return true, writeYAML(w, v)
	}
	return false, nil
}

// renderOrWrite encodes v as structured output if enabled, otherwise calls render.
func (o structuredOutputOptions) renderOrWrite(w io.Writer, v any, render func() error) error {
	if handled, err := o.encode(w, v); handled || err != nil {
		return err
	}
	return render()
}

type entryPageOutputOptions struct {
	PageInfo bool
	Output   structuredOutputOptions
	Noun     string
	Render   func(io.Writer, []*core.Entry) error
}

func addStructuredOutputFlags(cmd *cobra.Command, outputJSON, outputYAML *bool) {
	cmd.Flags().BoolVar(outputJSON, "json", false, "output as JSON")
	cmd.Flags().BoolVar(outputYAML, "yaml", false, "output as YAML")
	cmd.MarkFlagsMutuallyExclusive("json", "yaml")
}

func renderEntryPageOutput(cmd *cobra.Command, page *core.EntryPage, options entryPageOutputOptions) error {
	out := cmd.OutOrStdout()

	var value any = page.Entries
	if options.PageInfo {
		value = page
	}
	if handled, err := options.Output.encode(out, value); handled || err != nil {
		return err
	}

	if err := options.Render(out, page.Entries); err != nil {
		return err
	}
	if options.PageInfo {
		return writeEntryPageSummary(out, page, options.Noun)
	}
	return nil
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

func runSingleIDCommand[T any](cmd *cobra.Command, args []string, getService func() (T, error), action func(T, context.Context, int64) error, successFormat string) error {
	svc, id, err := serviceAndID(args, getService)
	if err != nil {
		return err
	}
	if err := action(svc, cmd.Context(), id); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), successFormat, id)
	return err
}

func serviceAndID[T any](args []string, getService func() (T, error)) (T, int64, error) {
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

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(v)
}

func writeStderrf(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format+"\n", args...)
	return err
}

func writeWarning(w io.Writer, format string, args ...any) error {
	return writeStderrf(w, "warning: "+format, args...)
}

func feedStatus(disabled bool, errorCount int) string {
	if disabled {
		return "disabled"
	}
	if errorCount > 0 {
		return fmt.Sprintf("err(%d)", errorCount)
	}
	return "ok"
}
