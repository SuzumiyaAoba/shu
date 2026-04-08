package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func tagCommands(getService tagServiceGetter) []*cobra.Command {
	return withGroup("tags",
		newTagCmd(getService),
		newUntagCmd(getService),
		newTagsCmd(getService),
	)
}

func newTagCmd(getService tagServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "tag <feed-id> <tag-name>",
		Short: "Add a tag to a feed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, id, err := serviceAndID(args, getService)
			if err != nil {
				return err
			}
			if err := svc.AddTag(cmd.Context(), id, args[1]); err != nil {
				return fmt.Errorf("tag feed #%d with %q: %w", id, args[1], err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Tagged feed #%d with %q\n", id, args[1])
			return err
		},
	}
}

func newUntagCmd(getService tagServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "untag <feed-id> <tag-name>",
		Short: "Remove a tag from a feed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, id, err := serviceAndID(args, getService)
			if err != nil {
				return err
			}
			if err := svc.RemoveTag(cmd.Context(), id, args[1]); err != nil {
				return fmt.Errorf("untag feed #%d %q: %w", id, args[1], err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed tag %q from feed #%d\n", args[1], id)
			return err
		},
	}
}

func newTagsCmd(getService tagServiceGetter) *cobra.Command {
	var output structuredOutputOptions

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
				return output.renderOrWrite(cmd.OutOrStdout(), tags, func() error {
					for _, t := range tags {
						if _, err := fmt.Fprintln(cmd.OutOrStdout(), t.Name); err != nil {
							return err
						}
					}
					return nil
				})
			}

			tags, err := svc.ListAllTags(ctx)
			if err != nil {
				return err
			}
			return output.renderOrWrite(cmd.OutOrStdout(), tags, func() error {
				return renderTagsTable(cmd.OutOrStdout(), tags, noColorFlag(cmd))
			})
		},
	}

	addStructuredOutputFlags(tagsCmd, &output.JSON, &output.YAML)
	return tagsCmd
}
