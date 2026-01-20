# stoptail

A terminal UI for Elasticsearch, inspired by elasticsearch-head. Built with Go and the [Charm](https://charm.sh) stack.

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
- **Index Filtering**: Filter by name patterns (wildcards supported) or aliases
- **Help Overlay**: Press `?` for keybindings

## Installation

### Homebrew (macOS/Linux)

```bash
brew install labtiva/tap/stoptail
```

### Go Install

```bash
go install github.com/labtiva/stoptail@latest
```

### From Releases

Download the latest binary from [GitHub Releases](https://github.com/labtiva/stoptail/releases).

## Usage

```bash
# Connect to localhost:9200 (default)
stoptail

# Connect with URL
stoptail --url https://user:pass@localhost:9200

# Or use environment variable
export ES_URL=https://user:pass@localhost:9200
stoptail
```

### Configuration Priority

1. `--url` flag (highest priority)
2. `ES_URL` environment variable
3. Default: `http://localhost:9200`

### URL Format

Include credentials in the URL:
```
https://username:password@hostname:9200
```

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Switch between Overview and Workbench |
| `r` | Refresh data |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

### Overview Tab

| Key | Action |
|-----|--------|
| `/` | Focus filter input |
| `1-9` | Toggle alias filter buttons |
| `Esc` | Clear filter / unfocus |

### Workbench Tab

| Key | Action |
|-----|--------|
| `m` | Cycle request method |
| `p` | Focus path input |
| `Enter` | Execute request |
| `Esc` | Unfocus inputs |

## Requirements

- Elasticsearch 8.x or 9.x

## Development

```bash
# Clone
git clone https://github.com/labtiva/stoptail.git
cd stoptail

# Start local Elasticsearch
docker compose up -d

# Run
go run .

# Test
go test ./...
```

## License

[MIT](LICENSE)
