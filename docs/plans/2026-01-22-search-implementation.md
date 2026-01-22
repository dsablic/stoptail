# Search in Text Views Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Ctrl+F search to Mappings, Nodes, and Tasks views with consistent UX.

**Architecture:** Create reusable SearchBar component wrapping textinput.Model with match tracking. Integrate into each view by adding search state, handling Ctrl+F activation, and rendering search bar when active.

**Tech Stack:** Go, Bubble Tea, Lipgloss

---

### Task 1: Create SearchBar Component

**Files:**
- Create: `internal/ui/search.go`

**Step 1: Create search.go with SearchBar struct**

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SearchBar struct {
	input      textinput.Model
	matches    []int
	currentIdx int
	active     bool
}

func NewSearchBar() SearchBar {
	input := textinput.New()
	input.Placeholder = "Search..."
	input.CharLimit = 100
	input.Width = 30
	return SearchBar{input: input}
}

func (s *SearchBar) Active() bool {
	return s.active
}

func (s *SearchBar) Activate() {
	s.active = true
	s.input.Focus()
	s.input.SetValue("")
	s.matches = nil
	s.currentIdx = 0
}

func (s *SearchBar) Deactivate() {
	s.active = false
	s.input.Blur()
}

func (s *SearchBar) Query() string {
	return s.input.Value()
}

func (s *SearchBar) Matches() []int {
	return s.matches
}

func (s *SearchBar) CurrentMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	return s.matches[s.currentIdx]
}

func (s *SearchBar) FindMatches(lines []string) {
	query := strings.ToLower(s.input.Value())
	s.matches = nil
	s.currentIdx = 0

	if query == "" {
		return
	}

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			s.matches = append(s.matches, i)
		}
	}
}

func (s *SearchBar) NextMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	s.currentIdx = (s.currentIdx + 1) % len(s.matches)
	return s.matches[s.currentIdx]
}

func (s *SearchBar) PrevMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	s.currentIdx--
	if s.currentIdx < 0 {
		s.currentIdx = len(s.matches) - 1
	}
	return s.matches[s.currentIdx]
}

func (s *SearchBar) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

func (s *SearchBar) View(width int) string {
	status := ""
	if len(s.matches) > 0 {
		status = lipgloss.NewStyle().Foreground(ColorGray).Render(
			fmt.Sprintf(" %d/%d ", s.currentIdx+1, len(s.matches)))
	} else if s.input.Value() != "" {
		status = lipgloss.NewStyle().Foreground(ColorRed).Render(" No matches ")
	}

	navBtns := ""
	if len(s.matches) > 0 {
		navBtns = " [<] [>]"
	}

	return lipgloss.NewStyle().
		Background(ActiveBg).
		Padding(0, 1).
		Width(width).
		Render("/" + s.input.View() + status + navBtns + " [x]")
}
```

**Step 2: Add missing import**

Add `"fmt"` to imports.

**Step 3: Build to verify**

Run: `go build .`
Expected: Success

**Step 4: Commit**

```bash
git add internal/ui/search.go
git commit -m "feat(ui): add reusable SearchBar component"
```

---

### Task 2: Add Search to Mappings View

**Files:**
- Modify: `internal/ui/mappings.go`

**Step 1: Add search field to MappingsModel**

In the struct definition around line 20, add:
```go
search SearchBar
```

**Step 2: Initialize search in NewMappings**

After creating the model, add:
```go
search: NewSearchBar(),
```

**Step 3: Add Ctrl+F handler in Update**

In the `tea.KeyMsg` switch, add before the existing cases:
```go
case "ctrl+f":
	if !m.filterActive {
		m.search.Activate()
		return m, nil
	}
```

**Step 4: Add search key handling when active**

At the start of the `tea.KeyMsg` handler, add:
```go
if m.search.Active() {
	switch msg.String() {
	case "esc":
		m.search.Deactivate()
		return m, nil
	case "enter":
		if match := m.search.NextMatch(); match >= 0 {
			m.mappingScroll = match
		}
		return m, nil
	case "shift+enter":
		if match := m.search.PrevMatch(); match >= 0 {
			m.mappingScroll = match
		}
		return m, nil
	default:
		cmd := m.search.Update(msg)
		m.updateMappingSearch()
		return m, cmd
	}
}
```

**Step 5: Add updateMappingSearch method**

```go
func (m *MappingsModel) updateMappingSearch() {
	if m.mappings == nil {
		return
	}
	var lines []string
	if m.treeView {
		lines = m.renderFieldsTree(m.mappings.Fields, 0, 1000)
	} else {
		lines = m.renderFieldsFlat(1000)
	}
	if len(m.analyzers) > 0 {
		lines = append(m.renderAnalyzers(1000), lines...)
	}
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.mappingScroll = match
	}
}
```

**Step 6: Render search bar in View**

In `renderMappingPane`, after the field lines loop and before returning, add:
```go
if m.search.Active() {
	b.WriteString(m.search.View(width - 4))
	b.WriteString("\n")
}
```

**Step 7: Build and test**

Run: `go build . && go test ./...`
Expected: Success

**Step 8: Commit**

```bash
git add internal/ui/mappings.go
git commit -m "feat(ui): add Ctrl+F search to Mappings view"
```

---

### Task 3: Add Search to Nodes View

**Files:**
- Modify: `internal/ui/nodes.go`

**Step 1: Add search field to NodesModel**

In the struct definition, add:
```go
search SearchBar
```

**Step 2: Initialize search in NewNodes**

```go
search: NewSearchBar(),
```

**Step 3: Add Ctrl+F handler**

In the `tea.KeyMsg` switch:
```go
case "ctrl+f":
	m.search.Activate()
	return m, nil
```

**Step 4: Add search key handling when active**

At the start of the `tea.KeyMsg` handler:
```go
if m.search.Active() {
	switch msg.String() {
	case "esc":
		m.search.Deactivate()
		return m, nil
	case "enter":
		if match := m.search.NextMatch(); match >= 0 {
			m.scrollY = match
		}
		return m, nil
	case "shift+enter":
		if match := m.search.PrevMatch(); match >= 0 {
			m.scrollY = match
		}
		return m, nil
	default:
		cmd := m.search.Update(msg)
		m.updateNodeSearch()
		return m, cmd
	}
}
```

**Step 5: Add updateNodeSearch method**

```go
func (m *NodesModel) updateNodeSearch() {
	if m.state == nil {
		return
	}
	lines := m.getSearchableLines()
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.scrollY = match
	}
}

func (m *NodesModel) getSearchableLines() []string {
	if m.state == nil {
		return nil
	}
	var lines []string
	for _, node := range m.state.Nodes {
		lines = append(lines, node.Name)
	}
	return lines
}
```

**Step 6: Render search bar in View**

At the end of the View method, before returning, when search is active:
```go
if m.search.Active() {
	content = lipgloss.JoinVertical(lipgloss.Left, content, m.search.View(m.width-4))
}
```

**Step 7: Build and test**

Run: `go build . && go test ./...`
Expected: Success

**Step 8: Commit**

```bash
git add internal/ui/nodes.go
git commit -m "feat(ui): add Ctrl+F search to Nodes view"
```

---

### Task 4: Add Search to Tasks View

**Files:**
- Modify: `internal/ui/tasks.go`

**Step 1: Add search field to TasksModel**

In the struct definition:
```go
search SearchBar
```

**Step 2: Initialize search in NewTasks**

```go
search: NewSearchBar(),
```

**Step 3: Add Ctrl+F handler**

In the `tea.KeyMsg` switch:
```go
case "ctrl+f":
	if m.confirming == "" {
		m.search.Activate()
		return m, nil
	}
```

**Step 4: Add search key handling when active**

At the start of the `tea.KeyMsg` handler:
```go
if m.search.Active() {
	switch msg.String() {
	case "esc":
		m.search.Deactivate()
		return m, nil
	case "enter":
		if match := m.search.NextMatch(); match >= 0 {
			m.selectedRow = match
			m.scrollY = max(0, match-5)
		}
		return m, nil
	case "shift+enter":
		if match := m.search.PrevMatch(); match >= 0 {
			m.selectedRow = match
			m.scrollY = max(0, match-5)
		}
		return m, nil
	default:
		cmd := m.search.Update(msg)
		m.updateTaskSearch()
		return m, cmd
	}
}
```

**Step 5: Add updateTaskSearch method**

```go
func (m *TasksModel) updateTaskSearch() {
	var lines []string
	for _, task := range m.tasks {
		lines = append(lines, task.Action+" "+task.Description+" "+task.Index+" "+task.Node)
	}
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.selectedRow = match
		m.scrollY = max(0, match-5)
	}
}
```

**Step 6: Render search bar in View**

At the end of View, before returning:
```go
if m.search.Active() {
	content = lipgloss.JoinVertical(lipgloss.Left, content, m.search.View(m.width-4))
}
```

**Step 7: Build and test**

Run: `go build . && go test ./...`
Expected: Success

**Step 8: Commit**

```bash
git add internal/ui/tasks.go
git commit -m "feat(ui): add Ctrl+F search to Tasks view"
```

---

### Task 5: Update Help and Documentation

**Files:**
- Modify: `internal/ui/help.go`
- Modify: `README.md`

**Step 1: Add Ctrl+F to Mappings section in help.go**

In the Mappings keys slice:
```go
{"Ctrl+F", "Search"},
```

**Step 2: Add Ctrl+F to Nodes section in help.go**

In the Nodes keys slice:
```go
{"Ctrl+F", "Search"},
```

**Step 3: Add Ctrl+F to Tasks section in help.go**

In the Tasks keys slice:
```go
{"Ctrl+F", "Search"},
```

**Step 4: Update README.md keybindings**

Add to Mappings Tab table:
```markdown
| `Ctrl+F` | Search fields |
```

Add to Nodes Tab table:
```markdown
| `Ctrl+F` | Search nodes |
```

Add to Tasks Tab table:
```markdown
| `Ctrl+F` | Search tasks |
```

**Step 5: Build and test**

Run: `go build . && go test ./...`
Expected: Success

**Step 6: Commit**

```bash
git add internal/ui/help.go README.md
git commit -m "docs: add Ctrl+F search to help and README"
```

---

### Task 6: Manual Testing

**Step 1: Test Mappings search**

1. Run `./stoptail`
2. Go to Mappings tab (press `m`)
3. Select an index and press Enter to load mappings
4. Press Ctrl+F
5. Type a field name
6. Verify matches are found and scroll jumps
7. Press Enter to go to next match
8. Press Shift+Enter to go to previous match
9. Press Esc to close search

**Step 2: Test Nodes search**

1. Go to Nodes tab
2. Press Ctrl+F
3. Type a node name
4. Verify navigation works

**Step 3: Test Tasks search**

1. Go to Tasks tab
2. Press Ctrl+F
3. Type an action or index name
4. Verify navigation works

**Step 4: Commit any fixes**

If any issues found, fix and commit.
