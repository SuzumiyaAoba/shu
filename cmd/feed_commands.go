package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/SuzumiyaAoba/shu/core"
	"github.com/spf13/cobra"
)

func feedCommands(getService serviceGetter) []*cobra.Command {
	return []*cobra.Command{
		newAddCmd(getService),
		newListCmd(getService),
		newFetchCmd(getService),
		newDiscoverCmd(getService),
		newUpdateCmd(getService),
		newRemoveCmd(getService),
		newEnableCmd(getService),
		newDisableCmd(getService),
	}
}

// newAddCmd implements "shu add <url>".
//
// It fetches the feed at the given URL to validate it, extracts metadata
// (title, site URL), and persists the feed record. If --title is provided,
// the user-supplied title is stored instead of the one from the feed document.
func newAddCmd(getService serviceGetter) *cobra.Command {
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
			if addJSON {
				return writeJSON(cmd.OutOrStdout(), feed)
			}
			if addYAML {
				return writeYAML(cmd.OutOrStdout(), feed)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added feed #%d: %s (%s)\n", feed.ID, feed.Title, feed.URL)
			return nil
		},
	}

	addCmd.Flags().StringVar(&addTitle, "title", "", "override feed title")
	addCmd.Flags().BoolVar(&addJSON, "json", false, "output as JSON")
	addCmd.Flags().BoolVar(&addYAML, "yaml", false, "output as YAML")
	addCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return addCmd
}

// newListCmd implements "shu list".
func newListCmd(getService serviceGetter) *cobra.Command {
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

			if listJSON {
				return writeJSON(cmd.OutOrStdout(), feeds)
			}
			if listYAML {
				return writeYAML(cmd.OutOrStdout(), feeds)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tURL\tFETCHED\tSTATUS")
			for _, f := range feeds {
				fetched := "-"
				if f.FetchedAt != nil {
					fetched = f.FetchedAt.Format("2006-01-02 15:04")
				}
				status := feedStatus(f.Disabled, f.ErrorCount)
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", f.ID, f.Title, f.URL, fetched, status)
			}
			return w.Flush()
		},
	}

	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	listCmd.Flags().BoolVar(&listYAML, "yaml", false, "output as YAML")
	listCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return listCmd
}

// newFetchCmd implements "shu fetch".
func newFetchCmd(getService serviceGetter) *cobra.Command {
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
				if fetchJSON {
					return writeJSON(cmd.OutOrStdout(), entries)
				}
				if fetchYAML {
					return writeYAML(cmd.OutOrStdout(), entries)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries from feed #%d\n", len(entries), fetchFeedID)
				return nil
			}

			count, err := svc.FetchAll(ctx)
			if err != nil {
				return err
			}
			if fetchJSON {
				return writeJSON(cmd.OutOrStdout(), map[string]int{"count": count})
			}
			if fetchYAML {
				return writeYAML(cmd.OutOrStdout(), map[string]int{"count": count})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count)
			return nil
		},
	}

	fetchCmd.Flags().Int64Var(&fetchFeedID, "feed-id", 0, "fetch a specific feed by ID")
	fetchCmd.Flags().BoolVar(&fetchJSON, "json", false, "output as JSON")
	fetchCmd.Flags().BoolVar(&fetchYAML, "yaml", false, "output as YAML")
	fetchCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return fetchCmd
}

func newDiscoverCmd(getService serviceGetter) *cobra.Command {
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

			if discoverJSON {
				return writeJSON(cmd.OutOrStdout(), feeds)
			}
			if discoverYAML {
				return writeYAML(cmd.OutOrStdout(), feeds)
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

	discoverCmd.Flags().BoolVar(&discoverJSON, "json", false, "output as JSON")
	discoverCmd.Flags().BoolVar(&discoverYAML, "yaml", false, "output as YAML")
	discoverCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	return discoverCmd
}

func newUpdateCmd(getService serviceGetter) *cobra.Command {
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
func newRemoveCmd(getService serviceGetter) *cobra.Command {
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

func newEnableCmd(getService serviceGetter) *cobra.Command {
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

func newDisableCmd(getService serviceGetter) *cobra.Command {
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
