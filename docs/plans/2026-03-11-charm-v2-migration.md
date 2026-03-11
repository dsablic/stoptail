# Charm v2 Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate all Charm libraries (bubbletea, lipgloss, bubbles, huh, glamour) from v1 to v2.

**Architecture:** Mechanical migration in phases - imports first, then fix compilation errors by category (types, APIs, patterns), then verify. Sub-models keep returning `string` from View(); only top-level models passed to `tea.NewProgram` return `tea.View`.

**Tech Stack:** Go, charm.land/bubbletea/v2, charm.land/lipgloss/v2, charm.land/bubbles/v2, charm.land/huh/v2, charm.land/glamour/v2

---

### Task 1: Update go.mod and imports

**Files:**
- Modify: `go.mod`
- Modify: ALL `.go` files importing charm libraries

**Step 1: Update go.mod dependencies**

Replace the charm dependencies:
```
github.com/charmbracelet/bubbletea  ->  charm.land/bubbletea/v2
github.com/charmbracelet/lipgloss   ->  charm.land/lipgloss/v2
github.com/charmbracelet/bubbles    ->  charm.land/bubbles/v2
github.com/charmbracelet/huh        ->  charm.land/huh/v2
github.com/charmbracelet/glamour    ->  charm.land/glamour/v2
```

Also update sub-package imports:
```
github.com/charmbracelet/bubbles/spinner    ->  charm.land/bubbles/v2/spinner
github.com/charmbracelet/bubbles/textinput  ->  charm.land/bubbles/v2/textinput
github.com/charmbracelet/bubbles/textarea   ->  charm.land/bubbles/v2/textarea
github.com/charmbracelet/bubbles/viewport   ->  charm.land/bubbles/v2/viewport
github.com/charmbracelet/bubbles/cursor     ->  charm.land/bubbles/v2/cursor
github.com/charmbracelet/lipgloss/table     ->  charm.land/lipgloss/v2/table
github.com/charmbracelet/huh               ->  charm.land/huh/v2
```

Run:
```bash
# First do a find-replace across all .go files for import paths
sed -i '' 's|github.com/charmbracelet/bubbletea|charm.land/bubbletea/v2|g' $(grep -rl 'github.com/charmbracelet/bubbletea' --include='*.go')
sed -i '' 's|github.com/charmbracelet/lipgloss|charm.land/lipgloss/v2|g' $(grep -rl 'github.com/charmbracelet/lipgloss' --include='*.go')
sed -i '' 's|github.com/charmbracelet/bubbles|charm.land/bubbles/v2|g' $(grep -rl 'github.com/charmbracelet/bubbles' --include='*.go')
sed -i '' 's|github.com/charmbracelet/huh|charm.land/huh/v2|g' $(grep -rl 'github.com/charmbracelet/huh' --include='*.go')
sed -i '' 's|github.com/charmbracelet/glamour|charm.land/glamour/v2|g' $(grep -rl 'github.com/charmbracelet/glamour' --include='*.go')
```

**Step 2: Update go.mod and fetch new deps**

```bash
# Remove old deps, add new ones
go get charm.land/bubbletea/v2@latest
go get charm.land/lipgloss/v2@latest
go get charm.land/bubbles/v2@latest
go get charm.land/huh/v2@latest
go get charm.land/glamour/v2@latest
go mod tidy
```

**Step 3: Verify imports resolve**

```bash
go build ./... 2>&1 | head -50
```

Expected: Compilation errors about changed APIs (NOT import errors).

**Step 4: Commit**

```bash
git add -A && git commit -m "chore: update charm library imports to v2 (charm.land)"
```

---

### Task 2: Fix lipgloss.Color type changes

In v2, `lipgloss.Color("xxx")` returns `color.Color` (from `image/color`), not `lipgloss.Color`. The type `lipgloss.Color` no longer exists as a storable type.

**Files:**
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/utils.go`
- Modify: `internal/ui/nodes.go`
- Modify: `internal/ui/mappings.go`
- Modify: `internal/ui/shard_picker.go`
- Modify: `internal/ui/overview.go`

**Step 1: Add `image/color` import and change variable types**

In `styles.go`, change all `lipgloss.Color` variable declarations to `color.Color`:
```go
import "image/color"

var (
    ColorGreen  color.Color = lipgloss.Color("#22c55e")
    ColorYellow color.Color = lipgloss.Color("#eab308")
    // ... etc for all color vars
)
```

**Step 2: Change function return types**

In `utils.go`:
```go
func HealthColor(health string) color.Color {
```

In `nodes.go`:
```go
func threadTypeColor(threadType string) color.Color {
```

In `mappings.go`:
```go
func (m MappingsModel) typeColor(fieldType string) color.Color {
```

**Step 3: Change local variable types**

In `shard_picker.go:82`: `var bgColor color.Color`
In `overview.go:1049`: `var shardColor color.Color`
In `utils.go:59`: `var color color.Color` (rename to avoid shadowing the package)

**Step 4: Verify compilation**

```bash
go build ./... 2>&1 | grep -i color | head -20
```

**Step 5: Commit**

```bash
git add -A && git commit -m "fix: update lipgloss.Color type to color.Color for v2"
```

---

### Task 3: Fix lipgloss.HasDarkBackground() signature

In v2, `HasDarkBackground()` requires `(io.Reader, io.Writer)` args.

**Files:**
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/help.go`

**Step 1: Update calls**

In `styles.go`:
```go
import "os"

isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
```

In `help.go`:
```go
if !lipgloss.HasDarkBackground(os.Stdin, os.Stdout) {
```

**Step 2: Verify**

```bash
go build ./internal/ui/...
```

**Step 3: Commit**

```bash
git add -A && git commit -m "fix: pass stdin/stdout to HasDarkBackground for lipgloss v2"
```

---

### Task 4: Fix tea.KeyMsg -> tea.KeyPressMsg

In v2, `tea.KeyMsg` is an interface. Use `tea.KeyPressMsg` for key press events in type switches.

**Files:** ALL files with `case tea.KeyMsg:` in Update():
- `main.go:239,324`
- `internal/ui/model.go:349`
- `internal/ui/workbench.go:317`
- `internal/ui/overview.go:118`
- `internal/ui/nodes.go:270`
- `internal/ui/mappings.go:110`
- `internal/ui/browser.go:165`
- `internal/ui/tasks.go:62`
- `internal/ui/shardcalc.go:61`
- `internal/ui/editor.go:357` (type assertion)

**Step 1: Replace all type switch cases**

In every `Update()` method, change:
```go
case tea.KeyMsg:
```
to:
```go
case tea.KeyPressMsg:
```

In `editor.go`, change the type assertion:
```go
if keyMsg, ok := msg.(tea.KeyMsg); ok {
```
to:
```go
if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
```

**Step 2: Fix direct KeyMsg construction**

These are the trickiest changes. In v2, you cannot construct `tea.KeyMsg` directly. Use `tea.KeyPressMsg` instead.

In `editor.go` (lines 409-419), change:
```go
tea.KeyMsg{Type: tea.KeyLeft}   -> tea.KeyPressMsg{Code: tea.KeyLeft}
tea.KeyMsg{Type: tea.KeyRight}  -> tea.KeyPressMsg{Code: tea.KeyRight}
tea.KeyMsg{Type: tea.KeyUp}     -> tea.KeyPressMsg{Code: tea.KeyUp}
tea.KeyMsg{Type: tea.KeyDown}   -> tea.KeyPressMsg{Code: tea.KeyDown}
tea.KeyMsg{Type: tea.KeyHome}   -> tea.KeyPressMsg{Code: tea.KeyHome}
tea.KeyMsg{Type: tea.KeyEnd}    -> tea.KeyPressMsg{Code: tea.KeyEnd}
```

In `workbench.go:510`:
```go
tea.KeyMsg{Type: tea.KeyLeft}   -> tea.KeyPressMsg{Code: tea.KeyLeft}
```

In `workbench.go:1204`:
```go
tea.KeyMsg{Type: tea.KeyDelete} -> tea.KeyPressMsg{Code: tea.KeyDelete}
```

**Note:** Check the actual v2 key constant names - they may have changed (e.g., `tea.KeyLeft` might now be a rune constant). Verify by checking the bubbletea v2 source or trying to compile.

**Step 3: Fix test file**

In `workbench_test.go`, update all KeyMsg constructions:
```go
// tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'"'}}
// becomes:
tea.KeyPressMsg{Code: '"', Text: "\""}

// tea.KeyMsg{Type: tea.KeyCtrlR}
// becomes:
tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}

// tea.KeyMsg{Type: tea.KeyCtrlF}
// becomes:
tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}

// tea.KeyMsg{Type: tea.KeyEnter}
// becomes:
tea.KeyPressMsg{Code: tea.KeyEnter}

// tea.KeyMsg{Type: tea.KeyCtrlP}
// becomes:
tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}

// tea.KeyMsg{Type: tea.KeyEsc}
// becomes:
tea.KeyPressMsg{Code: tea.KeyEscape}

// tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
// becomes:
tea.KeyPressMsg{Code: 'n', Text: "n"}
```

**Step 4: Verify**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add -A && git commit -m "fix: migrate tea.KeyMsg to tea.KeyPressMsg for bubbletea v2"
```

---

### Task 5: Fix mouse event handling

In v2, `tea.MouseMsg` is an interface. Mouse events are split into specific types: `tea.MouseClickMsg`, `tea.MouseReleaseMsg`, `tea.MouseWheelMsg`, `tea.MouseMotionMsg`. Coordinates accessed via `.Mouse()` method (returns `tea.Mouse` struct with X, Y fields).

**Files:**
- `internal/ui/model.go`
- `internal/ui/workbench.go`
- `internal/ui/overview.go`
- `internal/ui/nodes.go`
- `internal/ui/mappings.go`
- `internal/ui/browser.go`
- `internal/ui/tasks.go`

**Step 1: Identify patterns and their v2 replacements**

Pattern A - Click release detection:
```go
// v1:
case tea.MouseMsg:
    if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
        // use msg.X, msg.Y

// v2:
case tea.MouseReleaseMsg:
    mouse := msg.Mouse()
    if mouse.Button == tea.MouseLeft {
        // use mouse.X, mouse.Y
```

Pattern B - Wheel scroll:
```go
// v1:
case tea.MouseMsg:
    if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
        if msg.Button == tea.MouseButtonWheelUp { ... }

// v2:
case tea.MouseWheelMsg:
    mouse := msg.Mouse()
    if mouse.Button == tea.MouseWheelUp { ... }
```

Pattern C - Combined click + wheel (workbench.go, overview.go, nodes.go):
These need to be split into separate cases in the type switch:
```go
// v2:
case tea.MouseReleaseMsg:
    mouse := msg.Mouse()
    // handle clicks using mouse.X, mouse.Y, mouse.Button

case tea.MouseWheelMsg:
    mouse := msg.Mouse()
    // handle wheel using mouse.X, mouse.Y, mouse.Button
```

**Step 2: Fix model.go mouse handling**

The coordinate delegation pattern changes too:
```go
// v1:
delegateMsg := msg
if mouseMsg, ok := msg.(tea.MouseMsg); ok {
    mouseMsg.Y -= 2
    delegateMsg = mouseMsg
}

// v2: Need to check actual v2 API for coordinate adjustment.
// May need to create a new mouse message or adjust differently.
```

**Step 3: Update each file systematically**

Go through each file and apply the patterns above. The key changes:
- `tea.MouseMsg` type switch -> split into `tea.MouseReleaseMsg` + `tea.MouseWheelMsg`
- `msg.Action == tea.MouseActionRelease` -> handled by the type (MouseReleaseMsg)
- `msg.Button == tea.MouseButtonLeft` -> `mouse.Button == tea.MouseLeft`
- `msg.Button == tea.MouseButtonWheelUp` -> `mouse.Button == tea.MouseWheelUp`
- `msg.X` / `msg.Y` -> `msg.Mouse().X` / `msg.Mouse().Y` (or extract to local var)

**Step 4: Verify**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add -A && git commit -m "fix: migrate mouse events to v2 split types"
```

---

### Task 6: Fix View() return type for top-level models

Only models passed directly to `tea.NewProgram()` need to return `tea.View`. Sub-models keep returning `string`.

**Files:**
- Modify: `internal/ui/model.go` (main UI model)
- Modify: `main.go` (clusterPickerModal, urlResolverModel)

**Step 1: Update Model.View() in model.go**

```go
func (m Model) View() tea.View {
    var v tea.View
    v.SetContent(m.renderContent()) // extract current View body to helper
    return v
}
```

Or more simply using the declarative API:
```go
func (m Model) View() tea.View {
    content := // ... existing view logic returning string ...
    return tea.NewView(content)
}
```

The key addition: set `AltScreen` and `MouseMode` in the View:
```go
func (m Model) View() tea.View {
    content := m.viewContent()
    v := tea.NewView(content)
    v.AltScreen = true
    v.MouseMode = tea.MouseCellMotion
    return v
}
```

This replaces the `tea.WithAltScreen()` and `tea.WithMouseCellMotion()` program options.

**Step 2: Update main.go models**

For `clusterPickerModal`:
```go
func (m *clusterPickerModal) View() tea.View {
    return tea.NewView(m.viewContent())  // move current body to viewContent()
}
```

For `urlResolverModel`:
```go
func (m urlResolverModel) View() tea.View {
    return tea.NewView(m.viewContent())
}
```

**Step 3: Remove deprecated program options**

In `main.go:109`:
```go
// v1:
p := tea.NewProgram(ui.New(client, cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
// v2:
p := tea.NewProgram(ui.New(client, cfg))
```

The alt screen and mouse mode are now set declaratively in `View()`.

**Step 4: Update Init() signatures if needed**

Check if Init() return type changed. In v2 the signature is still `Init() tea.Cmd` (same as v1).

**Step 5: Verify**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add -A && git commit -m "fix: return tea.View from top-level models, use declarative terminal features"
```

---

### Task 7: Fix bubbles component API changes

**Files:**
- `internal/ui/workbench.go` (textinput, viewport, spinner)
- `internal/ui/overview.go` (textinput, spinner)
- `internal/ui/nodes.go` (textinput, spinner)
- `internal/ui/editor.go` (textarea)
- `internal/ui/search.go` (textinput)
- `internal/ui/bookmark.go` (textinput)
- `internal/ui/model.go` (spinner)
- `main.go` (spinner)

**Step 1: Fix viewport constructor**

In `workbench.go:97`:
```go
// v1:
vp := viewport.New(40, 10)
// v2:
vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))
```

**Step 2: Fix viewport field access**

In `workbench.go`:
```go
// v1:
m.response.Width = paneInnerWidth
m.response.Height = bodyHeight - 2
m.response.YOffset
// v2:
m.response.SetWidth(paneInnerWidth)
m.response.SetHeight(bodyHeight - 2)
m.response.YOffset()  // getter
```

Check all `m.response.YOffset` reads and replace with `m.response.YOffset()`.
Check all `m.response.Width` and `m.response.Height` assignments.

**Step 3: Fix textinput field access**

Fields that became methods in v2:
```go
// v1:
path.Width = 40
path.Cursor.Style = ...
path.TextStyle = ...
path.PlaceholderStyle = ...
// v2:
path.SetWidth(40)
// For style fields, check v2 API - they moved to Styles struct
```

For `Placeholder`, `CharLimit`, `Prompt` - check if these are still direct fields or became methods in v2. Based on the upgrade guide, Width became a method but Placeholder/CharLimit may still be fields.

**Step 4: Fix textarea changes**

In `editor.go`:
```go
// v1:
e.textarea.SetCursor(pos)  // This may have been renamed
// v2:
e.textarea.SetCursorColumn(pos)  // if SetCursor was renamed
```

Check: `SetCursor(col)` -> `SetCursorColumn(col)` per upgrade guide.

**Step 5: Fix spinner field access**

```go
// v1:
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)
// v2: Check if these are still fields or became methods
```

The spinner `Tick` method should still work as `m.spinner.Tick` (was already method-based).

Check if `spinner.TickMsg` type name changed.

**Step 6: Verify**

```bash
go build ./...
```

**Step 7: Commit**

```bash
git add -A && git commit -m "fix: update bubbles component APIs for v2 (getters/setters)"
```

---

### Task 8: Fix huh v2 changes

**Files:**
- Modify: `main.go`
- Modify: `internal/ui/modal.go`

**Step 1: Update ThemeBase call**

In `main.go`:
```go
// v1:
theme := huh.ThemeBase()
// v2:
theme := huh.ThemeBase(isDark)  // needs bool param
```

Need to determine `isDark` - use `lipgloss.HasDarkBackground(os.Stdin, os.Stdout)`.

**Step 2: Check form state and other huh APIs**

Verify `huh.StateCompleted` still exists, and form methods (`.WithShowHelp()`, `.WithShowErrors()`, `.WithTheme()`) still work. These likely haven't changed.

**Step 3: Verify**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add -A && git commit -m "fix: update huh theme to pass isDark for v2"
```

---

### Task 9: Fix glamour v2 changes

**Files:**
- Modify: `internal/ui/help.go`

**Step 1: Update glamour renderer**

In v2, `WithStandardStyle()` may have been removed. Use `WithStylePath()` or the new equivalent:
```go
// v1:
r, _ := glamour.NewTermRenderer(
    glamour.WithStandardStyle(styleName),
    glamour.WithWordWrap(40),
)
// v2: WithAutoStyle removed, WithStandardStyle may have changed
// Check actual v2 API - likely need WithStylePath("dark") or WithStylePath("light")
r, _ := glamour.NewTermRenderer(
    glamour.WithStylePath(styleName),
    glamour.WithWordWrap(40),
)
```

**Step 2: Verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add -A && git commit -m "fix: update glamour renderer for v2"
```

---

### Task 10: Fix remaining compilation errors

After all targeted fixes, there will likely be remaining compilation issues.

**Step 1: Iterate on build errors**

```bash
go build ./... 2>&1
```

Fix each error. Common remaining issues:
- Renamed constants or types
- Changed method signatures
- Removed deprecated functions
- `tea.Sequentially()` -> `tea.Sequence()` (if used)

**Step 2: Run tests**

```bash
go test ./...
```

Fix any test compilation or runtime failures.

**Step 3: Run linters**

```bash
go vet ./...
staticcheck ./...
```

**Step 4: Commit**

```bash
git add -A && git commit -m "fix: resolve remaining charm v2 compilation issues"
```

---

### Task 11: Verify UI rendering

**Step 1: Build and test render modes**

```bash
go build .
./stoptail --render overview --width 120 --height 40 localhost:9200
./stoptail --render workbench --width 120 --height 30 localhost:9200
./stoptail --render mappings --width 120 --height 40 localhost:9200
./stoptail --render cluster --width 120 --height 40 localhost:9200
./stoptail --render tasks --width 120 --height 40 localhost:9200
```

**Step 2: Test interactive mode**

```bash
./stoptail localhost:9200
```

Verify:
- Tab switching works
- Keyboard shortcuts work
- Mouse clicks work (tab switching, buttons)
- Mouse scroll works
- Search (Ctrl+F) works
- Filter (/) works
- Help (?) overlay works

**Step 3: Run editor tests**

```bash
go run cmd/editor-test/main.go -test
```

**Step 4: Final commit if any fixes needed**

```bash
git add -A && git commit -m "fix: post-migration UI verification fixes"
```

---

### Task 12: Update go.mod cleanup and final verification

**Step 1: Clean up go.mod**

```bash
go mod tidy
```

Verify old `github.com/charmbracelet/*` dependencies are gone from go.mod.

**Step 2: Full test suite**

```bash
go test ./...
go vet ./...
staticcheck ./...
```

**Step 3: Commit**

```bash
git add -A && git commit -m "chore: clean up go.mod after charm v2 migration"
```
