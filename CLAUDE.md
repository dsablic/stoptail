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
./stoptail --render cluster --width 120 --height 40 [cluster]
./stoptail --render cluster --view memory --width 120 --height 40 [cluster]
./stoptail --render cluster --view disk --width 120 --height 40 [cluster]
./stoptail --render cluster --view fielddata --width 120 --height 40 [cluster]
./stoptail --render cluster --view settings --width 120 --height 40 [cluster]
./stoptail --render cluster --view threadpools --width 120 --height 40 [cluster]
./stoptail --render cluster --view hotthreads --width 120 --height 40 [cluster]
./stoptail --render cluster --view templates --width 120 --height 40 [cluster]
./stoptail --render cluster --view deprecations --width 120 --height 40 [cluster]
./stoptail --render cluster --view shardhealth --width 120 --height 40 [cluster]
./stoptail --render tasks --width 120 --height 40 [cluster]
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run linters (always run before committing)
go vet ./...
staticcheck ./...

# Test query editor (syntax highlighting, cursor, selection, mouse handling)
go run cmd/editor-test/main.go -test

# Interactive query editor test (for manual verification)
go run cmd/editor-test/main.go
```

### Query Editor Testing

When modifying `internal/ui/editor.go`, always run the editor test suite:

```bash
go run cmd/editor-test/main.go -test
```

This tests:
- Syntax highlighting in unfocused/focused states
- Cursor rendering with syntax highlighting preserved
- Selection highlighting (Shift+arrow)
- GetSelectedText functionality

Note: Mouse text selection uses terminal-native selection (Alt+drag on Linux/Windows, Option+drag on macOS).

For interactive testing of cursor positioning and visual appearance:

```bash
go run cmd/editor-test/main.go
```

**Update the tests** in `cmd/editor-test/main.go` when:
- Changing cursor rendering or positioning
- Modifying syntax highlighting
- Adding new Shift+arrow selection features

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Keep functions focused and small
- Use table-driven tests where appropriate
- **No emojis** - Do not use emojis in code, commit messages, or documentation
- **No comments** - Write self-documenting code; do not add comments to the code
- **DRY** - Don't Repeat Yourself:
  - Extract repeated conditions into helper methods (e.g., `overviewAcceptsGlobalKeys()`)
  - Extract shared logic into utility functions (see Shared Utilities section)
  - If the same check appears 2+ times, create a helper
  - If the same string/value appears 2+ times, use a constant
  - Before writing new code, check if similar logic already exists

## Important Guidelines

### Always Update Tests

When adding or modifying functionality, tests are required:

1. **New functions** - Add unit tests in `*_test.go` files alongside the implementation
2. **New methods on structs** - Test the method behavior, including edge cases
3. **Bug fixes** - Add a test that would have caught the bug before fixing it
4. **ES API changes** - Update JSON parsing tests in `internal/es/cluster_test.go`
5. **Config changes** - Update tests in `internal/config/config_test.go`
6. **Sorting/filtering logic** - Test that results are ordered correctly and edge cases are handled

Run `go test ./...` and `staticcheck ./...` before committing to ensure all tests pass and there are no linter warnings.

### Always Update Documentation

**Before every commit**, check if documentation needs updating. Search for references to changed functionality in README.md, CLAUDE.md, help.go, and status bar text in model.go.

When making changes:

1. **New CLI flags** - Update help text in `main.go` and README.md
2. **New/changed keyboard shortcuts** - Update `internal/ui/help.go`, README.md keybindings table, and status bar in `internal/ui/model.go`
3. **New config options** - Document in README.md and CLAUDE.md
4. **Changed file paths** - Update both README.md and CLAUDE.md (e.g., config paths)
5. **New UI patterns** - Add to "Lipgloss Layout Patterns" or "Mouse Click Detection" sections in CLAUDE.md
6. **New features** - Add to README.md Features section
7. **Code quality improvements** - When consolidating logic, adding helper functions, or improving code structure, update CLAUDE.md patterns (e.g., `hasActiveInput()` pattern for keyboard handling)

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

### Shared Action Methods

When keyboard shortcuts and mouse clicks perform the same action, extract the logic into a shared helper method:

```go
// Helper method encapsulates the action
func (m *WorkbenchModel) startExecution() tea.Cmd {
    if m.client == nil || m.executing {
        return nil
    }
    m.prettyPrintBody()
    m.executing = true
    return tea.Batch(m.spinner.Tick, m.execute())
}

// Keyboard handler
case "ctrl+r":
    if cmd := m.startExecution(); cmd != nil {
        return m, cmd
    }

// Mouse handler (Run button click)
} else if msg.X < execEnd {
    if cmd := m.startExecution(); cmd != nil {
        return m, cmd
    }
}
```

Examples in codebase:
- `workbench.go`: `startExecution()` for Ctrl+R and Run button
- `nodes.go`: `selectView()` for 1/2/3 keys and tab clicks, `getMaxScroll()` for scroll bounds

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
- `OverlayModal(background, modal string, width, height int) string` - Overlay a centered modal on dimmed background
- `AutoColumnWidths(headers []string, rows [][]string, maxTotal int) []int` - Calculate optimal column widths for tables (ANSI-aware)
- `FitColumns(rows [][]string, widths []int) [][]string` - Truncate table cells to fit widths (ANSI-safe)

**ES utilities** (`internal/es/cluster.go`):
- `sortShardsByIndexShardPrimary(shards []ShardInfo)` - Sort shards by index, shard number, primary first
- `sortShardsByShardPrimary(shards []ShardInfo)` - Sort shards by shard number, primary first (for single-index queries)
- `AnalyzeShardHealth(idx IndexInfo) ShardHealth` - Analyze index for shard sizing issues (undersized, oversized, sparse)

**Storage utilities** (`internal/storage/history.go`):
- `StoptailDir() (string, error)` - Get the stoptail config directory (`~/.stoptail`)

When adding new functionality, check if a utility already exists before creating inline code. If the same logic appears in multiple places, extract it to the appropriate utilities file.

### Bubble Tea Patterns

- Models are immutable - return new model from Update()
- Use tea.Cmd for async operations (ES fetches)
- Delegate to sub-models for tab-specific logic
- Handle tea.WindowSizeMsg to propagate dimensions

**Create reusable components** - Always build reusable UI components that can be composed to create custom views. When a feature appears in multiple places (e.g., search, filtering, navigation), create a shared component:

```go
type SearchBar struct {
    input      textinput.Model
    matches    []int
    currentIdx int
    active     bool
}

func (s *SearchBar) HandleKey(msg tea.KeyMsg) (tea.Cmd, SearchAction) {
    switch msg.String() {
    case "esc":
        s.Deactivate()
        return nil, SearchActionClose
    case "enter", "ctrl+n":
        return nil, SearchActionNext
    // ...
    }
}
```

Components should:
- Encapsulate their own state and logic
- Return action enums so parent models can respond appropriately
- Provide consistent keyboard/mouse handling
- Be testable in isolation

See `internal/ui/search.go` for the SearchBar component used by both workbench and mappings views.

See `internal/ui/clipboard.go` for the Clipboard component used for cross-platform copy functionality (Ctrl+Y).

**Global keyboard handling** - When any input is active (search, filter, editor, modal), global keybindings (q, r, tab, ?, m, etc.) must be disabled so users can type. Use the consolidated `hasActiveInput()` helper:

```go
func (m Model) hasActiveInput() bool {
    switch m.activeTab {
    case TabOverview:
        return m.overview.filterActive || m.overview.HasModal()
    case TabWorkbench:
        return m.workbench.HasActiveInput()
    case TabMappings:
        return m.mappings.filterActive || m.mappings.search.Active()
    case TabTasks:
        return m.tasks.confirming != ""
    }
    return false
}

// In Update(), wrap all global shortcuts:
if !m.hasActiveInput() {
    switch msg.String() {
    case "q":
        return m, tea.Quit
    case "r":
        // refresh
    case "tab", "shift+tab":
        // tab navigation
    }
}

// Keys that work even with active input go in separate switch:
switch msg.String() {
case "ctrl+c":
    // special handling
}
```

Sub-models should expose `HasActiveInput()` or `HasModal()` methods. When adding new global shortcuts, add them inside the `if !m.hasActiveInput()` block.

**Adding new views to existing tabs** - When adding a new view (like Hot Threads to Cluster tab), follow this pattern:

1. Add view constant to the `NodesView` enum
2. Add data field to the model struct
3. Add `Set<Data>()` method
4. Add message type in `model.go` (e.g., `hotThreadsMsg`)
5. Add `fetch<Data>()` function in `model.go`
6. Add message handler in `Update()`
7. Add keyboard shortcut (e.g., case "6")
8. Add tab to `renderTabs()` list
9. Add case to View switch for rendering
10. Add `render<View>()` function
11. Add fetch to batch calls when refreshing the tab
12. Update help.go, README.md keybindings

**Filter vs Search** - Use the right pattern for the content type:

- **Tables (structured data)**: Use filter (`/`) - instantly hides non-matching rows
- **Text content (logs, code)**: Use search (`Ctrl+F`) - highlights matches, navigate with n/N

Tables in Cluster tab use filter because users want to narrow down to specific nodes/settings. Workbench response uses search because users want to find text within the JSON while seeing surrounding context.

**Table column sizing** - All tables should use automatic column sizing:

```go
headers := []string{"name", "value", "status"}
widths := AutoColumnWidths(headers, rows, m.width)
rows = FitColumns(rows, widths)

t := table.New().Headers(headers...).Rows(rows...)
```

This ensures columns fit their content up to terminal width, and handles ANSI escape codes correctly (e.g., progress bars with colors).

**Creating modals with huh** - When creating modals using `huh.NewForm()`, always use pointer receivers to avoid value copy issues:

```go
func newModal() *MyModal {
    m := &MyModal{}  // Pointer, not value
    m.form = huh.NewForm(
        huh.NewGroup(
            huh.NewSelect[string]().
                Value(&m.selected),  // Form binds to field via pointer
        ),
    )
    return m  // Return pointer
}

func (m *MyModal) Init() tea.Cmd { /* pointer receiver */ }
func (m *MyModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) { /* pointer receiver */ }
func (m *MyModal) View() string { /* pointer receiver */ }
```

If you create a value (`MyModal{}`) and return it, the form's pointer to `&m.selected` becomes invalid because you're returning a copy. Always use pointers for huh-based modals. See `internal/ui/modal.go` and the cluster picker in `main.go` for examples.

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

**Keep tab click handlers in sync** - When adding new views to tabs (e.g., Cluster tab), update BOTH the `renderTabs()` function AND the mouse click handler in `Update()`. The tabs list must be identical in both places:

```go
// In renderTabs() - defines what's rendered
tabs := []struct {
    key   string
    label string
    view  NodesView
}{
    {"1", "Memory", ViewMemory},
    {"2", "Disk", ViewDisk},
    // ... all views
}

// In Update() tea.MouseMsg handler - must match renderTabs()
tabs := []struct {
    label string
    view  NodesView
}{
    {"[1:Memory]", ViewMemory},
    {"[2:Disk]", ViewDisk},
    // ... all views (same order, same count)
}
```

### Verify UI Changes

Always verify UI changes using the render flag before committing:

```bash
./stoptail --render overview --width 120 --height 40 [cluster]
./stoptail --render workbench --width 120 --height 30 [cluster]
./stoptail --render mappings --width 120 --height 40 [cluster]
./stoptail --render cluster --width 120 --height 40 [cluster]
./stoptail --render cluster --view disk --width 120 --height 40 [cluster]
./stoptail --render cluster --view fielddata --width 120 --height 40 [cluster]
./stoptail --render cluster --view settings --width 120 --height 40 [cluster]
./stoptail --render cluster --view threadpools --width 120 --height 40 [cluster]
./stoptail --render cluster --view hotthreads --width 120 --height 40 [cluster]
./stoptail --render cluster --view templates --width 120 --height 40 [cluster]
./stoptail --render cluster --view deprecations --width 120 --height 40 [cluster]
./stoptail --render cluster --view shardhealth --width 120 --height 40 [cluster]
./stoptail --render tasks --width 120 --height 40 [cluster]
```

This renders the UI to stdout without starting the full TUI, allowing visual verification of layout, borders, and styling.

### Update Demo GIF When UI Changes

When making UI changes (new tabs, new keybindings, layout changes), update `demo.tape` and regenerate `demo.gif`:

1. **Update demo.tape** to showcase the new functionality
2. **Build and regenerate**: `go build . && rm -f ~/.stoptail/history.json && vhs demo.tape`
3. **Verify and commit** both `demo.tape` and `demo.gif`

**IMPORTANT**: Always build before running vhs - the demo uses `./stoptail` (local binary), not the installed version.

### Verify Demo GIF Before Committing

**CRITICAL: NEVER commit demo.gif without visual verification.** A broken demo (showing shell prompts instead of the app) is worse than no update.

When modifying `demo.tape`, always verify the generated `demo.gif` before committing:

1. **Regenerate the demo**: `go build . && rm -f ~/.stoptail/history.json && vhs demo.tape`
2. **Extract frames for verification**:
   ```bash
   mkdir -p /tmp/demo-frames
   ffmpeg -i demo.gif -vf "select=not(mod(n\\,30))" -vsync vfr /tmp/demo-frames/frame_%03d.png
   ```
3. **Check key frames** to verify:
   - **NO shell prompts visible** (if you see `$` prompt, the app crashed - demo is broken)
   - Overview tab shows `es-node-1` (not Docker container ID)
   - Filter shows filtered indices
   - Workbench shows `[REST]` label and correct layout
   - Cluster tab shows views (Memory, Disk, etc.)
   - Help overlay displays correctly

**Before recording demo**:
- Always use: `go build . && rm -f ~/.stoptail/history.json && vhs demo.tape`
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
  # AWS OpenSearch (auto-detected from URL)
  aws-prod:
    url: https://search-mycluster.us-east-1.es.amazonaws.com
    aws_profile: production  # optional
```

Note: `url_command` executes shell commands - this is intentional for secrets management integration.
