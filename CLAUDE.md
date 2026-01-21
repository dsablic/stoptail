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
│   ├── config/            # Configuration parsing (~/.stoptail/config.yaml)
│   ├── es/                # Elasticsearch client and data fetching
│   ├── storage/           # Query history persistence (~/.stoptail/history.json)
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

# Run with named cluster from ~/.stoptail/config.yaml
./stoptail production

# Debug UI rendering without full TUI (flags must come before cluster name)
./stoptail --render overview --width 120 --height 40 [cluster]
./stoptail --render workbench --width 120 --height 30 [cluster]
./stoptail --render nodes --width 120 --height 40 [cluster]
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
- **No emojis** - Do not use emojis in code, commit messages, or documentation
- **No comments** - Write self-documenting code; do not add comments to the code

## Important Guidelines

### Always Update Tests

When adding or modifying functionality, tests are required:

1. **New functions** - Add unit tests in `*_test.go` files alongside the implementation
2. **New methods on structs** - Test the method behavior, including edge cases
3. **Bug fixes** - Add a test that would have caught the bug before fixing it
4. **ES API changes** - Update JSON parsing tests in `internal/es/cluster_test.go`
5. **Config changes** - Update tests in `internal/config/config_test.go`
6. **Sorting/filtering logic** - Test that results are ordered correctly and edge cases are handled

Run `go test ./...` before committing to ensure all tests pass.

### Always Update Documentation

When making changes:

1. **New CLI flags** - Update help text in `main.go` and README.md
2. **New keyboard shortcuts** - Update `internal/ui/help.go` and README.md keybindings table
3. **New config options** - Document in README.md and CLAUDE.md
4. **Changed file paths** - Update both README.md and CLAUDE.md (e.g., config paths)
5. **New UI patterns** - Add to "Lipgloss Layout Patterns" or "Mouse Click Detection" sections in CLAUDE.md
6. **New features** - Add to README.md Features section

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
r := []rune(s)
if len(r) > max {
    return string(r[:max])
}
```

### Bubble Tea Patterns

- Models are immutable - return new model from Update()
- Use tea.Cmd for async operations (ES fetches)
- Delegate to sub-models for tab-specific logic
- Handle tea.WindowSizeMsg to propagate dimensions

### Lipgloss Layout Patterns

**Width() excludes borders** - When using `Width(n)` on a bordered style, the border adds 2 chars to the total. For two side-by-side bordered panes:

```go
paneInnerWidth := (terminalWidth - 5) / 2
```

**Always show borders to prevent layout shifts** - If a component's border can appear/disappear based on focus, the layout shifts when focus changes. Always render borders, changing only the color:

```go
borderColor := ColorGray
if focused {
    borderColor = ColorBlue
}
style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
```

**Manual line joining for side-by-side panes** - `lipgloss.JoinHorizontal` with `Height()` causes excessive padding. Join lines manually with ANSI-aware trimming:

```go
trimANSI := func(s string) string {
    for strings.HasSuffix(s, " ") || strings.HasSuffix(s, "\x1b[0m") {
        s = strings.TrimSuffix(s, " ")
        s = strings.TrimSuffix(s, "\x1b[0m")
    }
    return s + "\x1b[0m"
}
for i := 0; i < maxLines; i++ {
    paneLines = append(paneLines, trimANSI(leftLines[i])+" "+trimANSI(rightLines[i]))
}
```

### Mouse Click Detection

**Match rendered content exactly** - Click handlers must use identical strings and styles as View() for accurate position calculations. Styles add padding that affects width:

```go
// WRONG: raw string width ignores style padding
tabWidth := lipgloss.Width("[1:Memory]")

// CORRECT: apply same style used in View()
tabWidth := lipgloss.Width(InactiveTabStyle.Render("[1:Memory]"))
```

**Account for multi-line components** - If a component spans multiple lines (bordered input = 3 lines), adjust Y boundaries:

```go
topRowHeight := 3
if msg.Y < topRowHeight+1 {
    // handle top row clicks
}
```

**Add mouse scroll support** - Handle wheel events for scrollable content:

```go
if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
    if msg.Button == tea.MouseButtonWheelUp {
        m.scrollY = max(0, m.scrollY-3)
    } else {
        m.scrollY = min(maxScroll, m.scrollY+3)
    }
}
```

**Adjust coordinates for nested views** - When delegating mouse events to sub-models, adjust Y for any header/chrome offset:

```go
delegateMsg := msg
if mouseMsg, ok := msg.(tea.MouseMsg); ok {
    mouseMsg.Y -= headerHeight
    delegateMsg = mouseMsg
}
subModel.Update(delegateMsg)
```

### Verify UI Changes

Always verify UI changes using the render flag before committing:

```bash
./stoptail --render overview --width 120 --height 40 [cluster]
./stoptail --render workbench --width 120 --height 30 [cluster]
./stoptail --render nodes --width 120 --height 40 [cluster]
```

Note: Flags must come before the cluster name argument.

This renders the UI to stdout without starting the full TUI, allowing visual verification of layout, borders, and styling.

## Releasing

Releases are automated via GitHub Actions + goreleaser:

```bash
# Tag a new version
git tag v0.x.y
git push origin v0.x.y
```

The workflow builds binaries for Linux, macOS, and Windows (amd64/arm64).

## Multi-cluster Config

Users can configure clusters in `~/.stoptail/config.yaml`:

```yaml
clusters:
  production:
    url: https://user:pass@es-prod:9200
  staging:
    url_command: "vault read -field=url secret/es-staging"
```

Note: `url_command` executes shell commands - this is intentional for secrets management integration.
