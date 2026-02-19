# stoptail

[![Go Version](https://img.shields.io/github/go-mod/go-version/dsablic/stoptail)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/dsablic/stoptail)](https://github.com/dsablic/stoptail/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/labtiva/stoptail)](https://goreportcard.com/report/github.com/labtiva/stoptail)

A terminal UI for Elasticsearch, inspired by elasticsearch-head. Built with Go and the [Charm](https://charm.sh) stack.

![stoptail demo](demo.gif)

## Features

- **Overview Tab**: Visual shard grid showing nodes vs indices with colored shard boxes
  - Green: Primary shards
  - Blue: Replica shards
  - Purple: Relocating shards
  - Yellow: Initializing shards
  - Red: Unassigned shards
  - Master node marked with bold name and `*` indicator
  - Select any cell to view shard info (with allocation explain for problem shards)
  - Index management: create, delete, open, close indices
- **Workbench Tab**: Full request editor like Kibana Dev Tools
  - Support for GET, POST, PUT, DELETE, HEAD methods
  - JSON syntax highlighting in responses
  - Real-time JSON validation with error line marker
  - Query autocomplete for ES DSL keywords and index field names
  - Bracket auto-pairing for `{}`, `[]`, and `""`
  - ES|QL mode for SQL-like queries (Ctrl+E to toggle)
  - Save and load query bookmarks (Ctrl+S/Ctrl+B)
- **Browser Tab**: Document browser
  - Browse documents in any index
  - Three-pane layout: indices, document list, document detail
  - Auto-loads more documents on scroll
  - JSON syntax highlighting for document source
- **Mappings Tab**: Index mapping explorer
  - Browse field mappings for any index
  - Toggle between tree and flat view
  - View field types and properties (analyzer, index, doc_values, etc.)
  - See custom analyzers, tokenizers, filters, and normalizers
  - Toggle to view index settings (shards, replicas, refresh interval, etc.)
- **Cluster Tab**: Node statistics and cluster-wide information with 9 switchable views
  - Memory: heap%, GC stats, fielddata, query cache, segments (master node marked with `*`)
  - Disk: disk usage, shard counts, versions (master node marked with `*`)
  - Threads: thread pool stats (active, queue, rejected, completed)
  - Hot: hot threads diagnostic for CPU-intensive operations
  - Fielddata: per-node/index/field fielddata breakdown
  - Shards: shard health analysis (undersized, oversized, sparse detection)
  - Templates: index templates with patterns, priority, and component templates
  - Settings: cluster settings (persistent, transient, defaults)
  - Deprecations: migration deprecation warnings from `_migration/deprecations` API
- **Tasks Tab**: Monitor long-running and pending operations
  - Reindex, update-by-query, delete-by-query tracking
  - Force merge and snapshot operations
  - Pending cluster tasks (create index, update mapping, etc.)
  - Cancel with confirmation
- **Shard Calculator**: Plan optimal shard topology (press `S`)
  - Input total size and docs to get recommended shard count
  - Optional: node count for distribution planning
- **Index Filtering**: Filter by name patterns (wildcards supported) or aliases
- **Multi-cluster Config**: Configure multiple clusters in `~/.stoptail/config.yaml`
- **Help Overlay**: Press `?` for keybindings

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap dsablic/tap
brew install stoptail
```

### From Releases

Download the latest binary from [GitHub Releases](https://github.com/dsablic/stoptail/releases).

### Go Install

```bash
go install github.com/dsablic/stoptail@latest
```

## Usage

```bash
# Connect to localhost:9200 (default)
stoptail

# Connect with URL
stoptail https://user:pass@localhost:9200

# Connect to a named cluster from ~/.stoptail/config.yaml
stoptail production

# Or use environment variable
export ES_URL=https://user:pass@localhost:9200
stoptail

# Show version
stoptail --version
```

### Configuration Priority

1. URL argument (highest priority)
2. Named cluster from `~/.stoptail/config.yaml`
3. `ES_URL` environment variable
4. Default: `http://localhost:9200`

### Multi-cluster Configuration

Create `~/.stoptail/config.yaml` to configure multiple clusters:

```yaml
clusters:
  production:
    url: https://user:pass@es-prod.example.com:9200
  staging:
    url: https://user:pass@es-staging.example.com:9200
  local:
    url: http://localhost:9200
  # Dynamic URL from command (useful for secrets managers)
  vault-cluster:
    url_command: "vault read -field=url secret/elasticsearch"
```

Then connect by name:
```bash
stoptail production
```

If no argument is provided and multiple clusters are configured, you'll be prompted to select one.

**Note:** The legacy path `~/.stoptail.yaml` is still supported for backwards compatibility.

### AWS OpenSearch

stoptail automatically detects AWS OpenSearch endpoints and uses AWS SigV4 authentication:

```yaml
clusters:
  # AWS OpenSearch (managed) - auto-detected from URL
  aws-prod:
    url: https://search-mycluster.us-east-1.es.amazonaws.com

  # AWS OpenSearch Serverless - also auto-detected
  aws-serverless:
    url: https://abc123xyz.us-west-2.aoss.amazonaws.com

  # With specific AWS profile
  aws-staging:
    url: https://search-staging.eu-west-1.es.amazonaws.com
    aws_profile: staging-account
```

**Credentials** are resolved using the standard AWS SDK chain:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (for EC2/ECS/Lambda)
4. SSO credentials (`aws sso login`)

Use `aws_profile` to select a specific profile from `~/.aws/credentials` or `~/.aws/config`.

### Data Storage

stoptail stores data in `~/.stoptail/`:

| File | Description |
|------|-------------|
| `config.yaml` | Cluster configuration |
| `history.json` | Workbench query history |
| `bookmarks.json` | Saved query bookmarks |

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Switch between tabs (Overview, Workbench, Browser, Mappings, Cluster, Tasks) |
| `r` | Refresh data |
| `?` | Toggle help overlay |
| `S` | Shard calculator |
| `q` / `Ctrl+C` | Quit |

### Overview Tab

| Key | Action |
|-----|--------|
| `/` | Focus filter input |
| `Esc` | Clear all filters |
| `Left/Right` | Select index (column) |
| `Up/Down` | Select node (row) |
| `1-9` | Toggle alias filter buttons |
| `U` | Show only UNASSIGNED shards |
| `R` | Show only RELOCATING shards |
| `I` | Show only INITIALIZING shards |
| `.` | Toggle system indices (starting with .) |
| `Enter` | Show shard info (picker if multiple shards) |
| `c` | Create new index |
| `d` | Delete selected index |
| `o` | Open closed index |
| `x` | Close selected index |
| `a` | Add alias to selected index |
| `A` | Remove alias from selected index |

### Workbench Tab

| Key | Action |
|-----|--------|
| `Enter` | Activate editor |
| `Tab` | Trigger autocomplete (in body) / cycle focus |
| `Ctrl+R` | Execute request |
| `Ctrl+E` | Toggle REST/ES|QL mode |
| `Alt+F` | Format JSON body |
| `Ctrl+S` | Save query as bookmark |
| `Ctrl+B` | Load bookmark |
| `Ctrl+F` | Search in response |
| `Enter` / `n` | Next search match |
| `Shift+Enter` / `N` | Previous search match |
| `Ctrl+Y` | Copy body or response to clipboard |
| `Ctrl+A` | Select all text in body |
| `Ctrl+C` | Copy selected text |
| `Ctrl+V` | Paste from clipboard |
| `Ctrl+Z` | Undo |
| `Ctrl+Shift+Z` | Redo |
| `Shift+Arrow` | Select text |
| `Shift+Home/End` | Select to line start/end |
| `Up/Down` | Navigate completions (when open) |
| `Esc` | Dismiss completions / deactivate editor / close search |
| Mouse drag | Select text in editor |

### Browser Tab

| Key | Action |
|-----|--------|
| `/` | Filter indices |
| `Left/Right` | Switch between panes |
| `Up/Down` | Navigate / scroll |
| `Enter` | Load documents for selected index |
| `Ctrl+Y` | Copy document JSON |

### Mappings Tab

| Key | Action |
|-----|--------|
| `/` | Filter indices |
| `Ctrl+F` | Search fields |
| `Ctrl+Y` | Copy mappings to clipboard |
| `Enter` / `n` | Next search match |
| `Shift+Enter` / `N` | Previous search match |
| `Left/Right` | Switch between index list and mapping panes |
| `Enter` | Load mappings for selected index |
| `t` | Toggle tree/flat view |
| `s` | Toggle settings view |
| `Up/Down` | Scroll |

### Cluster Tab

| Key | Action |
|-----|--------|
| `1` | Memory view |
| `2` | Disk view |
| `3` | Thread pools view |
| `4` | Hot threads view |
| `5` | Fielddata view |
| `6` | Shard health view |
| `7` | Templates view |
| `8` | Settings view |
| `9` | Deprecations view |
| `/` | Filter |
| `Esc` | Clear filter |
| `Enter` | View setting details (in Settings view) |
| `Up/Down` | Scroll / select |

### Tasks Tab

| Key | Action |
|-----|--------|
| `Enter` | View task details |
| `Ctrl+F` | Search tasks |
| `c` | Cancel selected task |
| `y` | Confirm cancel |
| `n` / `Esc` | Abort cancel |
| `Up/Down` | Select task |

## Requirements

- Elasticsearch 7.x, 8.x, or 9.x

## Development

```bash
# Clone
git clone https://github.com/dsablic/stoptail.git
cd stoptail

# Start local Elasticsearch with sample data
docker compose up -d

# Run
go run .

# Test
go test ./...

# Build with version info
go build -ldflags "-X main.version=dev -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%d)" .
```

### Sample Data

The docker-compose setup automatically seeds Elasticsearch with sample indices:

| Index | Shards | Description |
|-------|--------|-------------|
| products | 2 | Electronics inventory |
| orders | 2 | Customer orders |
| users | 2 | Customer accounts |
| logs-2026.* | 2 | Application logs (3 indices) |
| metrics-cpu | 2 | CPU metrics |
| metrics-memory | 2 | Memory metrics |
| high-shard-index | 64 | Test index with many shards |
| medium-shard-index | 16 | Test index with moderate shards |
| analytics-events | 12 | Analytics events |
| search-content | 8 | Search content |

**Aliases:** `ecommerce`, `logs`, `logs-current`, `metrics`

To reset data:
```bash
docker compose down -v && docker compose up -d
```

## License

[MIT](LICENSE)
