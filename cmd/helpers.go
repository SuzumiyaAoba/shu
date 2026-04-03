package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

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
		if pageInfo {
			return true, writeJSON(w, page)
		}
		return true, writeJSON(w, page.Entries)
	}
	if outputYAML {
		if pageInfo {
			return true, writeYAML(w, page)
		}
		return true, writeYAML(w, page.Entries)
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
