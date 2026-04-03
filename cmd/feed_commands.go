package cmd

import (
	"fmt"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

func feedCommands(getService feedServiceGetter) []*cobra.Command {
	return withGroup("feeds",
		newAddCmd(getService),
		newListCmd(getService),
		newFetchCmd(getService),
		newDiscoverCmd(getService),
		newUpdateCmd(getService),
		newRemoveCmd(getService),
		newRemoveDeadFeedsCmd(getService),
		newEnableCmd(getService),
		newDisableCmd(getService),
	)
}

// newAddCmd implements "shu add <url>".
//
// It fetches the feed at the given URL to validate it, extracts metadata
// (title, site URL), and persists the feed record. If --title is provided,
// the user-supplied title is stored instead of the one from the feed document.
func newAddCmd(getService feedServiceGetter) *cobra.Command {
	var addTitle string
	var addJSON bool
	var addYAML bool

	addCmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Add a feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}

			feed, err := svc.AddFeed(cmd.Context(), args[0], addTitle)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd.OutOrStdout(), feed, addJSON, addYAML); handled || err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added feed #%d: %s (%s)\n", feed.ID, feed.Title, feed.URL)
			return nil
		},
	}

	addCmd.Flags().StringVar(&addTitle, "title", "", "override feed title")
	addStructuredOutputFlags(addCmd, &addJSON, &addYAML)
	return addCmd
}

// newListCmd implements "shu list".
func newListCmd(getService feedServiceGetter) *cobra.Command {
	var listJSON bool
	var listYAML bool

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}

			feeds, err := svc.ListFeeds(cmd.Context())
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd.OutOrStdout(), feeds, listJSON, listYAML); handled || err != nil {
				return err
			}
			return renderFeedsTable(cmd.OutOrStdout(), feeds)
		},
	}

	addStructuredOutputFlags(listCmd, &listJSON, &listYAML)
	return listCmd
}

// newFetchCmd implements "shu fetch".
func newFetchCmd(getService feedServiceGetter) *cobra.Command {
	var fetchFeedID int64
	var fetchJSON bool
	var fetchYAML bool

	fetchCmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch all feeds (or a specific feed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			if fetchFeedID > 0 {
				entries, err := svc.FetchFeed(ctx, fetchFeedID)
				if err != nil {
					return err
				}
				if handled, err := writeStructuredOutput(cmd.OutOrStdout(), entries, fetchJSON, fetchYAML); handled || err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries from feed #%d\n", len(entries), fetchFeedID)
				return nil
			}

			count, err := svc.FetchAll(ctx)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd.OutOrStdout(), map[string]int{"count": count}, fetchJSON, fetchYAML); handled || err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
			return nil
		},
	}

	fetchCmd.Flags().Int64Var(&fetchFeedID, "feed-id", 0, "fetch a specific feed by ID")
	addStructuredOutputFlags(fetchCmd, &fetchJSON, &fetchYAML)
	return fetchCmd
}

func newDiscoverCmd(getService feedServiceGetter) *cobra.Command {
	var discoverJSON bool
	var discoverYAML bool

	discoverCmd := &cobra.Command{
		Use:   "discover <url>",
		Short: "Discover feed URLs from a web page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			feeds, err := svc.DiscoverFeeds(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd.OutOrStdout(), feeds, discoverJSON, discoverYAML); handled || err != nil {
				return err
			}

			if len(feeds) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No feeds found.")
				return nil
			}
			for _, u := range feeds {
				fmt.Fprintln(cmd.OutOrStdout(), u)
			}
			return nil
		},
	}

	addStructuredOutputFlags(discoverCmd, &discoverJSON, &discoverYAML)
	return discoverCmd
}

func newUpdateCmd(getService feedServiceGetter) *cobra.Command {
	var updateTitle string
	var updateURL string

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a feed's title or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
			}

			update := core.FeedUpdate{}
			if cmd.Flags().Changed("title") {
				update.Title = &updateTitle
			}
			if cmd.Flags().Changed("url") {
				update.URL = &updateURL
			}

			if err := svc.UpdateFeed(cmd.Context(), id, update); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated feed #%d\n", id)
			return nil
		},
	}

	updateCmd.Flags().StringVar(&updateTitle, "title", "", "new title")
	updateCmd.Flags().StringVar(&updateURL, "url", "", "new URL")
	return updateCmd
}

// newRemoveCmd implements "shu remove <id>".
func newRemoveCmd(getService feedServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a feed and its entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
			}
			if err := svc.RemoveFeed(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed feed #%d\n", id)
			return nil
		},
	}
}

func newEnableCmd(getService feedServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <feed-id>",
		Short: "Re-enable a disabled feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
			}
			if err := svc.EnableFeed(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Enabled feed #%d\n", id)
			return nil
		},
	}
}

func newDisableCmd(getService feedServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <feed-id>",
		Short: "Disable a feed (skip during fetch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}
			id, err := parseIDArg(args[0])
			if err != nil {
				return err
			}
			if err := svc.DisableFeed(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Disabled feed #%d\n", id)
			return nil
		},
	}
}

func newRemoveDeadFeedsCmd(getService feedServiceGetter) *cobra.Command {
	var dryRun bool
	var outputJSON bool
	var outputYAML bool

	removeDeadCmd := &cobra.Command{
		Use:   "remove-dead-feeds",
		Short: "Remove feeds with recorded fetch failures",
		Long:  "Remove feeds considered dead: feeds with recorded fetch errors. Manually disabled feeds without errors are not removed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getService()
			if err != nil {
				return err
			}

			var feeds []*core.Feed
			if dryRun {
				feeds, err = svc.ListDeadFeeds(cmd.Context())
			} else {
				feeds, err = svc.RemoveDeadFeeds(cmd.Context())
			}
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd.OutOrStdout(), feeds, outputJSON, outputYAML); handled || err != nil {
				return err
			}

			if len(feeds) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No dead feeds found.")
				return nil
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Would remove %d dead feeds\n", len(feeds))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %d dead feeds\n", len(feeds))
			return nil
		},
	}

	removeDeadCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show dead feeds without removing them")
	addStructuredOutputFlags(removeDeadCmd, &outputJSON, &outputYAML)
	return removeDeadCmd
}
