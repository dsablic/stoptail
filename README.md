# stoptail

A terminal UI for Elasticsearch, inspired by elasticsearch-head. Built with Go and the [Charm](https://charm.sh) stack.

![stoptail demo](demo.gif)

## Features

- **Overview Tab**: Visual shard grid showing nodes vs indices with colored shard boxes
  - Green: Primary shards
  - Blue: Replica shards
  - Yellow: Relocating shards
  - Red: Unassigned shards
- **Workbench Tab**: Full request editor like Kibana Dev Tools
  - Support for GET, POST, PUT, DELETE, HEAD methods
  - JSON syntax highlighting in responses
  - Real-time JSON validation
  - Query autocomplete for ES DSL keywords and index field names
- **Nodes Tab**: Node statistics with 4 switchable views
  - Memory: heap%, GC stats, fielddata, query cache, segments
  - Disk: disk usage, shard counts, versions
  - Fielddata by Index: top 20 indices by fielddata memory
  - Fielddata by Field: field-level fielddata breakdown
- **Tasks Tab**: Monitor long-running operations
  - Reindex, update-by-query, delete-by-query tracking
  - Force merge and snapshot operations
  - Cancel with confirmation
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

### Data Storage

stoptail stores data in `~/.stoptail/`:

| File | Description |
|------|-------------|
| `config.yaml` | Cluster configuration |
| `history.json` | Workbench query history |

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Switch between tabs (Overview, Workbench, Nodes) |
| `r` | Refresh data |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

### Overview Tab

| Key | Action |
|-----|--------|
| `/` | Focus filter input |
| `Esc` | Clear all filters |
| `Left/Right` | Select index (column) |
| `Up/Down` | Scroll nodes (rows) |
| `1-9` | Toggle alias filter buttons |
| `U` | Show only UNASSIGNED shards |
| `R` | Show only RELOCATING shards |
| `I` | Show only INITIALIZING shards |
| `Enter` | Open selected index in Workbench |

### Workbench Tab

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus (method, path, body, response) |
| `Ctrl+Enter` | Execute request |
| `"` or `:` | Trigger autocomplete |
| `Up/Down` | Navigate completions (when open) |
| `Enter/Tab` | Accept completion |
| `Esc` | Dismiss completions |

### Nodes Tab

| Key | Action |
|-----|--------|
| `1` | Memory view |
| `2` | Disk view |
| `3` | Fielddata by Index view |
| `4` | Fielddata by Field view |
| `Up/Down` | Scroll |

### Tasks Tab

| Key | Action |
|-----|--------|
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
