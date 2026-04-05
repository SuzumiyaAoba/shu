package cmd

import (
	"context"
	"fmt"
	"os"

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
	var output structuredOutputOptions

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
			if handled, err := output.encode(cmd.OutOrStdout(), feed); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added feed #%d: %s (%s)\n", feed.ID, feed.Title, feed.URL)
			return err
		},
	}

	addCmd.Flags().StringVar(&addTitle, "title", "", "override feed title")
	addStructuredOutputFlags(addCmd, &output.JSON, &output.YAML)
	return addCmd
}

// newListCmd implements "shu list".
func newListCmd(getService feedServiceGetter) *cobra.Command {
	var output structuredOutputOptions

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
			return output.renderOrWrite(cmd.OutOrStdout(), feeds, func() error {
				return renderFeedsTable(cmd.OutOrStdout(), feeds, noColorFlag(cmd))
			})
		},
	}

	addStructuredOutputFlags(listCmd, &output.JSON, &output.YAML)
	return listCmd
}

// newFetchCmd implements "shu fetch".
func newFetchCmd(getService feedServiceGetter) *cobra.Command {
	var fetchFeedID int64
	var output structuredOutputOptions

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
				if handled, err := output.encode(cmd.OutOrStdout(), entries); handled || err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries from feed #%d\n", len(entries), fetchFeedID)
				return err
			}

			// Show a progress bar when stderr is a TTY and --quiet is not set.
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")
			var observer core.FetchObserver
			var progress *fetchProgress
			if !quiet && isTTY(os.Stderr) {
				feeds, err := svc.ListFeeds(ctx)
				if err != nil {
					return err
				}
				if len(feeds) > 0 {
					progress = newFetchProgress(os.Stderr, len(feeds))
					observer = progress
				}
			}

			count, fetchErr := svc.FetchAllWithObserver(ctx, observer)
			if progress != nil {
				progress.Wait()
			}
			// A context cancellation is a hard stop — return immediately.
			if fetchErr != nil && ctx.Err() != nil {
				return fetchErr
			}
			if handled, err := output.encode(cmd.OutOrStdout(), map[string]int{"count": count}); handled || err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d new entries\n", count); err != nil {
				return err
			}
			// Print per-feed warnings for partial failures so the user can see
			// which feeds failed without aborting the command.
			if fetchErr != nil {
				type multiErr interface{ Unwrap() []error }
				if me, ok := fetchErr.(multiErr); ok {
					for _, e := range me.Unwrap() {
						if err := writeWarning(cmd.ErrOrStderr(), "%v", e); err != nil {
							return err
						}
					}
				} else {
					if err := writeWarning(cmd.ErrOrStderr(), "%v", fetchErr); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}

	fetchCmd.Flags().Int64Var(&fetchFeedID, "feed-id", 0, "fetch a specific feed by ID")
	addStructuredOutputFlags(fetchCmd, &output.JSON, &output.YAML)
	return fetchCmd
}

func newDiscoverCmd(getService feedServiceGetter) *cobra.Command {
	var output structuredOutputOptions

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
			return output.renderOrWrite(cmd.OutOrStdout(), feeds, func() error {
				if len(feeds) == 0 {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "No feeds found.")
					return err
				}
				for _, u := range feeds {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), u); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	addStructuredOutputFlags(discoverCmd, &output.JSON, &output.YAML)
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
			update := core.FeedUpdate{}
			if cmd.Flags().Changed("title") {
				update.Title = &updateTitle
			}
			if cmd.Flags().Changed("url") {
				update.URL = &updateURL
			}

			return runSingleIDCommand(cmd, args, getService, func(svc feedService, ctx context.Context, id int64) error {
				return svc.UpdateFeed(ctx, id, update)
			}, "Updated feed #%d\n")
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
			return runSingleIDCommand(cmd, args, getService, func(svc feedService, ctx context.Context, id int64) error {
				return svc.RemoveFeed(ctx, id)
			}, "Removed feed #%d\n")
		},
	}
}

func newEnableCmd(getService feedServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <feed-id>",
		Short: "Re-enable a disabled feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSingleIDCommand(cmd, args, getService, func(svc feedService, ctx context.Context, id int64) error {
				return svc.EnableFeed(ctx, id)
			}, "Enabled feed #%d\n")
		},
	}
}

func newDisableCmd(getService feedServiceGetter) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <feed-id>",
		Short: "Disable a feed (skip during fetch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSingleIDCommand(cmd, args, getService, func(svc feedService, ctx context.Context, id int64) error {
				return svc.DisableFeed(ctx, id)
			}, "Disabled feed #%d\n")
		},
	}
}

func newRemoveDeadFeedsCmd(getService feedServiceGetter) *cobra.Command {
	var dryRun bool
	var output structuredOutputOptions

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
			return output.renderOrWrite(cmd.OutOrStdout(), feeds, func() error {
				if len(feeds) == 0 {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "No dead feeds found.")
					return err
				}
				if dryRun {
					_, err := fmt.Fprintf(cmd.OutOrStdout(), "Would remove %d dead feeds\n", len(feeds))
					return err
				}
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Removed %d dead feeds\n", len(feeds))
				return err
			})
		},
	}

	removeDeadCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show dead feeds without removing them")
	addStructuredOutputFlags(removeDeadCmd, &output.JSON, &output.YAML)
	return removeDeadCmd
}
