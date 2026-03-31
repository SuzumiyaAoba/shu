package cmd

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file.opml>",
	Short: "Import feeds from an OPML file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		added, err := svc.ImportOPML(cmd.Context(), f)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Imported %d feeds\n", added)
		return nil
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export feeds as OPML",
	RunE: func(cmd *cobra.Command, args []string) error {
		opml, err := svc.ExportOPML(cmd.Context())
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprint(out, xml.Header)
		enc := xml.NewEncoder(out)
		enc.Indent("", "  ")
		if err := enc.Encode(opml); err != nil {
			return fmt.Errorf("encode OPML: %w", err)
		}
		fmt.Fprintln(out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
}
