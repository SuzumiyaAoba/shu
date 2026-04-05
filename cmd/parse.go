package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
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

type paginationFlags struct {
	Limit    int
	Offset   int
	Page     int
	PageInfo bool
}

func (f paginationFlags) resolveOffset() (int, error) {
	if f.Page < 0 {
		return 0, fmt.Errorf("--page must be >= 0")
	}
	if f.Offset < 0 {
		return 0, fmt.Errorf("--offset must be >= 0")
	}
	if f.Page > 0 {
		return (f.Page - 1) * f.Limit, nil
	}
	return f.Offset, nil
}

// parseDate parses a date string in YYYY-MM-DD format and returns the
// corresponding time.Time at midnight UTC.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func addPaginationFlags(cmd *cobra.Command, flags *paginationFlags, limitUsage, offsetUsage string) {
	cmd.Flags().IntVar(&flags.Limit, "limit", 20, limitUsage)
	cmd.Flags().IntVar(&flags.Offset, "offset", 0, offsetUsage)
	cmd.Flags().IntVar(&flags.Page, "page", 0, "1-based page number (uses --limit)")
	cmd.Flags().BoolVar(&flags.PageInfo, "page-info", false, "include pagination metadata")
	cmd.MarkFlagsMutuallyExclusive("offset", "page")
}
