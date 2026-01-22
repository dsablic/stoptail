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

# Debug UI rendering without full TUI
./stoptail --render overview --width 120 --height 40 [cluster]
./stoptail --render workbench --width 120 --height 30 [cluster]
./stoptail --render workbench --width 120 --height 20 --body '{"invalid": json}' [cluster]
./stoptail --render mappings --width 120 --height 40 [cluster]
./stoptail --render nodes --width 120 --height 40 [cluster]
./stoptail --render nodes --view memory --width 120 --height 40 [cluster]
./stoptail --render nodes --view disk --width 120 --height 40 [cluster]
./stoptail --render nodes --view fielddata --width 120 --height 40 [cluster]
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
- **DRY** - Extract shared logic into utility functions (see Shared Utilities section below)

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

**Before every commit**, check if documentation needs updating. Search for references to changed functionality in README.md, CLAUDE.md, help.go, and status bar text in model.go.

When making changes:

1. **New CLI flags** - Update help text in `main.go` and README.md
2. **New/changed keyboard shortcuts** - Update `internal/ui/help.go`, README.md keybindings table, and status bar in `internal/ui/model.go`
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

### Shared Utilities

Common utility functions are in `internal/ui/utils.go` to avoid duplication:

**UI utilities** (`internal/ui/utils.go`):
- `Truncate(s string, maxLen int) string` - Unicode-safe string truncation with ellipsis
- `TrimANSI(s string) string` - Trim trailing spaces and ANSI reset codes for side-by-side panes
- `HealthColor(health string) lipgloss.Color` - Map ES health status to colors

**ES utilities** (`internal/es/cluster.go`):
- `sortShardsByIndexShardPrimary(shards []ShardInfo)` - Sort shards by index, shard number, primary first
- `sortShardsByShardPrimary(shards []ShardInfo)` - Sort shards by shard number, primary first (for single-index queries)

**Storage utilities** (`internal/storage/history.go`):
- `StoptailDir() (string, error)` - Get the stoptail config directory (`~/.stoptail`)

When adding new functionality, check if a utility already exists before creating inline code. If the same logic appears in multiple places, extract it to the appropriate utilities file.

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

**Manual line joining for side-by-side panes** - `lipgloss.JoinHorizontal` with `Height()` causes excessive padding. Join lines manually using `TrimANSI()` from `internal/ui/utils.go`:

```go
for i := 0; i < maxLines; i++ {
    paneLines = append(paneLines, TrimANSI(leftLines[i])+" "+TrimANSI(rightLines[i]))
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
./stoptail --render mappings --width 120 --height 40 [cluster]
./stoptail --render nodes --width 120 --height 40 [cluster]
./stoptail --render nodes --view disk --width 120 --height 40 [cluster]
./stoptail --render nodes --view fielddata --width 120 --height 40 [cluster]
```

This renders the UI to stdout without starting the full TUI, allowing visual verification of layout, borders, and styling.

### Update Demo GIF When UI Changes

When making UI changes (new tabs, new keybindings, layout changes), update `demo.tape` and regenerate `demo.gif`:

1. **Build first**: `go build .`
2. **Update demo.tape** to showcase the new functionality
3. **Regenerate**: `vhs demo.tape`
4. **Verify and commit** both `demo.tape` and `demo.gif`

### Verify Demo GIF Before Committing

When modifying `demo.tape`, always verify the generated `demo.gif` before committing:

1. **Regenerate the demo**: `vhs demo.tape`
2. **Extract frames for verification**:
   ```bash
   mkdir -p /tmp/demo-frames
   ffmpeg -i demo.gif -vf "select=not(mod(n\\,50))" -vsync vfr /tmp/demo-frames/frame_%03d.png
   ```
3. **Check key frames** to verify:
   - Overview tab shows `es-node-1` (not Docker container ID)
   - Filter shows filtered indices
   - Workbench shows correct path (`/products/_search`) and 200 response
   - Mappings tab shows field mappings with Ctrl+F search
   - Nodes tab shows all 3 views (Memory, Disk, Fielddata)
   - Help overlay displays correctly

**Before recording demo**:
- Build first: `go build .`
- Clear history to start fresh: `rm ~/.stoptail/history.json`
- Demo uses `./stoptail` (local build) not `stoptail` (may be old homebrew version)

**Common demo.tape issues**:
- Tab from Overview goes to Workbench with focus on path (no extra Tab needed)
- Path input loads from history if exists - clear history first or use `Ctrl+a` then `Ctrl+k` to select all and delete before typing
- Body input loads from history - use `Ctrl+a` then `Ctrl+k` to clear before typing query
- After typing in body, Tab cycles to response, then another Tab switches to next tab
- Ctrl+F search only works when focus is in the right pane (response for Workbench)

**Keep demo generic**:
- Use `http://localhost:9200` URL directly (not cluster names from config)
- Use `Hide`/`Show` to hide shell prompt (avoids showing username/machine)
- ES node should be named `es-node-1` (set via `node.name` in docker-compose.yml)

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
