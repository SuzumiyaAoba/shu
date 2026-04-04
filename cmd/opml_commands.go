package cmd

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func opmlCommands(getService opmlServiceGetter) []*cobra.Command {
	return withGroup("opml",
		newImportCmd(getService),
		newExportCmd(getService),
	)
}

func newImportCmd(getService opmlServiceGetter) *cobra.Command {
	var output structuredOutputOptions

	importCmd := &cobra.Command{
		Use:   "import <file.opml>",
		Short: "Import feeds from an OPML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer func() { _ = f.Close() }()

			result, err := svc.ImportOPMLDetailed(cmd.Context(), f)
			if err != nil {
				return err
			}
			if handled, err := output.encode(cmd.OutOrStdout(), result); handled || err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Imported %d feeds", result.AddedCount); err != nil {
				return err
			}
			if result.ReusedCount > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), ", reused %d", result.ReusedCount); err != nil {
					return err
				}
			}
			if result.TaggedCount > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), ", applied %d tags", result.TaggedCount); err != nil {
					return err
				}
			}
			if len(result.Issues) > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), ", %d issues", len(result.Issues)); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout())
			return err
		},
	}

	addStructuredOutputFlags(importCmd, &output.JSON, &output.YAML)
	return importCmd
}

func newExportCmd(getService opmlServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export feeds as OPML",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			opml, err := svc.ExportOPML(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprint(out, xml.Header); err != nil {
				return err
			}
			enc := xml.NewEncoder(out)
			enc.Indent("", "  ")
			if err := enc.Encode(opml); err != nil {
				return fmt.Errorf("encode OPML: %w", err)
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			return nil
		},
	}
}
