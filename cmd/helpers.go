package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// parseIDArg parses a command-line argument as an int64 ID.
func parseIDArg(arg string) (int64, error) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID %q: %w", arg, err)
	}
	return id, nil
}

// writeJSON encodes v as pretty-printed JSON to w.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
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
