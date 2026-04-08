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
	var fetchAfterImport bool

	importCmd := &cobra.Command{
		Use:   "import <file.opml>",
		Short: "Import feeds from an OPML file",
		Long: `Import feeds from an OPML file without fetching feed content.

Feeds are registered immediately without any HTTP requests.
Run 'shu fetch' afterwards to download entries for the imported feeds.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("open OPML file %q: %w", args[0], err)
			}
			defer func() { _ = f.Close() }()

			result, err := svc.ImportOPMLDetailed(cmd.Context(), f)
			if err != nil {
				return fmt.Errorf("import OPML %q: %w", args[0], err)
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
			if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
				return err
			}

			if fetchAfterImport && result.AddedCount > 0 {
				count, err := svc.FetchAll(cmd.Context())
				if err != nil {
					return fmt.Errorf("fetch: %w", err)
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
				return err
			}
			return nil
		},
	}

	addStructuredOutputFlags(importCmd, &output.JSON, &output.YAML)
	importCmd.Flags().BoolVar(&fetchAfterImport, "fetch", false, "fetch entries for newly imported feeds immediately")
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
