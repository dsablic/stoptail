# CLAUDE.md - Project Guidelines for Claude

## Project Overview

stoptail is an Elasticsearch TUI (Terminal User Interface) built with Go. It provides a terminal-based interface for monitoring Elasticsearch clusters, similar to elasticsearch-head but for the terminal.

## Tech Stack

- **Language:** Go 1.22+
- **TUI Framework:** Bubble Tea (bubbletea) with Lipgloss for styling
- **ES Client:** elastic/go-elasticsearch/v8
- **Config:** YAML via gopkg.in/yaml.v3

## Project Structure

```
.
├── main.go                 # Entry point, CLI flags, cluster selection
├── internal/
│   ├── config/            # Configuration parsing (~/.stoptail.yaml)
│   ├── es/                # Elasticsearch client and data fetching
│   └── ui/                # Bubble Tea models (overview, workbench, nodes)
├── scripts/               # Helper scripts (sample data)
└── .goreleaser.yaml       # Release configuration
```

## Building and Running

```bash
# Build
go build .

# Run (connects to localhost:9200 by default)
./stoptail

# Run with specific cluster
./stoptail https://user:pass@elasticsearch:9200

# Run with named cluster from ~/.stoptail.yaml
./stoptail production

# Debug UI rendering without full TUI
./stoptail --render overview --width 120 --height 40
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...
```

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Keep functions focused and small
- Use table-driven tests where appropriate

## Important Guidelines

### Always Update Tests

When adding or modifying functionality:

1. **New functions** - Add unit tests in `*_test.go` files
2. **Bug fixes** - Add a test that would have caught the bug
3. **ES API changes** - Update JSON parsing tests in `internal/es/cluster_test.go`
4. **Config changes** - Update tests in `internal/config/config_test.go`

### Always Update Documentation

When making changes:

1. **New CLI flags** - Update help text in `main.go`
2. **New keyboard shortcuts** - Update `internal/ui/help.go`
3. **New config options** - Document in README.md
4. **API changes** - Update relevant comments

### ES API Error Handling

Always check `res.IsError()` after ES API calls:

```go
res, err := c.es.Cat.Indices(...)
if err != nil {
    return nil, fmt.Errorf("fetching indices: %w", err)
}
defer res.Body.Close()

if res.IsError() {
    body, _ := io.ReadAll(res.Body)
    return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
}
```

### Unicode Safety

Use rune-based operations for string manipulation that may contain non-ASCII:

```go
// Good
r := []rune(s)
if len(r) > max {
    return string(r[:max])
}

// Bad - may corrupt multi-byte characters
if len(s) > max {
    return s[:max]
}
```

### Bubble Tea Patterns

- Models are immutable - return new model from Update()
- Use tea.Cmd for async operations (ES fetches)
- Delegate to sub-models for tab-specific logic
- Handle tea.WindowSizeMsg to propagate dimensions

## Releasing

Releases are automated via GitHub Actions + goreleaser:

```bash
# Tag a new version
git tag v0.x.y
git push origin v0.x.y
```

The workflow builds binaries for Linux, macOS, and Windows (amd64/arm64).

## Multi-cluster Config

Users can configure clusters in `~/.stoptail.yaml`:

```yaml
clusters:
  production:
    url: https://user:pass@es-prod:9200
  staging:
    url_command: "vault read -field=url secret/es-staging"
```

Note: `url_command` executes shell commands - this is intentional for secrets management integration.
