package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func tagCommands(getService serviceGetter) []*cobra.Command {
	return []*cobra.Command{
		newTagCmd(getService),
		newUntagCmd(getService),
		newTagsCmd(getService),
	}
}

func newTagCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "tag <feed-id> <tag-name>",
		Short: "Add a tag to a feed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
			}
			if err := svc.AddTag(cmd.Context(), id, args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tagged feed #%d with %q\n", id, args[1])
			return nil
		},
	}
}

func newUntagCmd(getService serviceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "untag <feed-id> <tag-name>",
		Short: "Remove a tag from a feed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
			}
			if err := svc.RemoveTag(cmd.Context(), id, args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed tag %q from feed #%d\n", args[1], id)
			return nil
		},
	}
}

func newTagsCmd(getService serviceGetter) *cobra.Command {
	var tagsJSON bool
	var tagsYAML bool

	tagsCmd := &cobra.Command{
		Use:   "tags [feed-id]",
		Short: "List tags (all or for a specific feed)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if len(args) == 1 {
				id, err := parseIDArg(args[0])
				if err != nil {
					return err
				}
				tags, err := svc.ListTags(ctx, id)
				if err != nil {
					return err
				}
				if tagsJSON {
					return writeJSON(cmd.OutOrStdout(), tags)
				}
				if tagsYAML {
					return writeYAML(cmd.OutOrStdout(), tags)
				}
				for _, t := range tags {
					fmt.Fprintln(cmd.OutOrStdout(), t.Name)
				}
				return nil
			}

			tags, err := svc.ListAllTags(ctx)
			if err != nil {
				return err
			}

			if tagsJSON {
				return writeJSON(cmd.OutOrStdout(), tags)
			}
			if tagsYAML {
				return writeYAML(cmd.OutOrStdout(), tags)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME")
			for _, t := range tags {
				fmt.Fprintf(w, "%d\t%s\n", t.ID, t.Name)
			}
			return w.Flush()
		},
	}

	tagsCmd.Flags().BoolVar(&tagsJSON, "json", false, "output as JSON")
	tagsCmd.Flags().BoolVar(&tagsYAML, "yaml", false, "output as YAML")
	tagsCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return tagsCmd
}
