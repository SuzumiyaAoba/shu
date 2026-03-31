package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <entry-id>",
	Short: "Open an entry in the default browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0])
		if err != nil {
			return err
		}

		entry, err := svc.GetEntry(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("entry #%d: %w", id, err)
		}

		if entry.Link == "" {
			return fmt.Errorf("entry #%d has no link", id)
		}

		// Best-effort mark as read when opening.
		if err := svc.MarkEntryRead(cmd.Context(), id); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not mark entry as read: %v\n", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Opening %s\n", entry.Link)
		return openBrowser(entry.Link)
	},
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func init() {
	rootCmd.AddCommand(openCmd)
}
