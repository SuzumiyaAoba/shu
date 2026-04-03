# shu

`shu` is a clean, dependency-free, efficient command-line RSS/Atom feed aggregator and reader. It is designed around simplicity, power, and Unix philosophy, providing flexible JSON/YAML outputs for jq/yq pipelining and built entirely in Go with SQLite.

## Features

- **Full RSS/Atom Support**: Uses `mmcdole/gofeed` to cover pretty much any RSS/Atom feed out there.
- **Smart Fetching**: Automatic deduplication, health tracking for broken feeds, and full support for `ETag`/`Last-Modified` conditional GET requests.
- **Organization**: Bookmark stars (`star`/`unstar`), marking read/unread, and custom tags for feeds.
- **UI-ready Core**: Progress observers for fetch workflows and batch entry state APIs make it easier to build alternate frontends such as a TUI.
- **Discovery**: Built-in RSS feed auto-discovery from standard website URLs (`shu discover`).
- **Flexible Formats**: Native stdout table rendering or machine-readable `--json` and `--yaml` formats seamlessly integrated across all query commands.
- **Import/Export**: Full OPML import and export capabilities.

## Installation

You can install `shu` using standard Go toolchain:

```bash
go install github.com/SuzumiyaAoba/shu@latest
```

Ensure `$(go env GOPATH)/bin` is in your `$PATH`.

## Usage

### Managing Feeds

```bash
# Add a new feed
shu add https://news.ycombinator.com/rss

# Add a feed with a custom title
shu add https://example.com/feed.xml --title "My Custom Title"

# List all subscribed feeds
shu list

# Discover feeds on a website
shu discover https://example.com

# Update a feed's title or URL
shu update 1 --title "Updated Title"

# Remove a feed
shu remove 1
```

### Fetching & Reading

```bash
# Fetch new entries for all feeds
shu fetch

# Fetch new entries for a specific feed
shu fetch --feed-id 1

# View latest entries (globally)
shu entries

# View unread entries for a specific feed
shu entries --feed-id 1 --unread

# Mark an entry as read or unread
shu read 1
shu unread 1

# Star/bookmark an entry
shu star 1
shu unstar 1
```

### Organizing & Searching

```bash
# Tag a feed
shu tag 1 "tech"
shu tag 1 "news"

# Untag a feed
shu untag 1 "tech"

# View all tags
shu tags

# Search across all downloaded entries
shu search "golang" --limit 50

# View feed statistics
shu stats
```

### Import & Export

```bash
# Export all feeds to OPML
shu export > feeds.opml

# Import feeds from OPML
shu import feeds.opml

# Inspect import results as JSON
shu import feeds.opml --json
```

### Machine Readable Outputs (JSON / YAML)

All commands that output data (like `list`, `entries`, `stats`, `discover`, `search`, etc.) support `--json` and `--yaml` flags, perfect for scripting:

```bash
shu list --json | jq '.[].title'
shu entries --unread --yaml | yq '.[].link'
```

### Maintenance

```bash
# Clean up entries older than 90 days (keeps starred entries)
shu cleanup --older-than 2160h
```

## Architecture

`shu` follows a clean 4-layer architecture:
1.  **Frontend Layer (`cmd/`)**: Parses commands, flags, handles stdin/stdout formatting, and outputs tables/JSON/YAML using Cobra.
2.  **Runtime Bootstrap (`app/`)**: Owns reusable startup concerns such as logger/store/service wiring so multiple frontends can share the same initialization path.
3.  **Core Domain (`core/`)**: Contains business logic, the `Feed` and `Entry` data models, feed fetching, typed entry metadata helpers, URL discovery logic, and the `Service` interface.
4.  **Storage (`store/`)**: Provides the SQLite-backed persistence logic, implementing the domain interfaces with transactional, concurrent-safe operations.

The core fetch API also exposes structured observer callbacks (`FetchObserver`) so long-running frontends can react to per-feed progress, skips, and completion events without parsing logs or stdout.

## Database

By default, the SQLite database is located at `~/.shu/shu.db`. You can override this using the global `--db` flag:

```bash
shu --db /path/to/custom.db list

# Optional SQLite tuning
shu --sqlite-busy-timeout 5s --sqlite-max-open-conns 2 list
```

## License

MIT License
