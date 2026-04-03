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
	var importJSON bool
	var importYAML bool

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
			defer f.Close()

			result, err := svc.ImportOPMLDetailed(cmd.Context(), f)
			if err != nil {
				return err
			}

			if importJSON {
				return writeJSON(cmd.OutOrStdout(), result)
			}
			if importYAML {
				return writeYAML(cmd.OutOrStdout(), result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d feeds", result.AddedCount)
			if result.ReusedCount > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), ", reused %d", result.ReusedCount)
			}
			if result.TaggedCount > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), ", applied %d tags", result.TaggedCount)
			}
			if len(result.Issues) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), ", %d issues", len(result.Issues))
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}

	importCmd.Flags().BoolVar(&importJSON, "json", false, "output as JSON")
	importCmd.Flags().BoolVar(&importYAML, "yaml", false, "output as YAML")
	importCmd.MarkFlagsMutuallyExclusive("json", "yaml")
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
}
