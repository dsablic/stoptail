# Tasks View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a "Tasks" tab showing long-running cancellable ES operations (reindex, snapshot, force merge, update-by-query, delete-by-query) with cancel confirmation.

**Architecture:** New TasksModel sub-model with table view, fetching from `_tasks` API filtered by action type. Cancel via POST to `_tasks/{id}/_cancel`.

**Tech Stack:** Go, Bubble Tea, Lipgloss, go-elasticsearch/v8

---

## Task 1: Add TaskInfo struct and parsing test

**Files:**
- Modify: `internal/es/cluster.go`
- Modify: `internal/es/cluster_test.go`

**Step 1: Add TaskInfo struct to cluster.go**

Add after FielddataByIndex struct (around line 86):

```go
type TaskInfo struct {
	ID            string
	Action        string
	Node          string
	Index         string
	RunningTime   string
	RunningTimeMs int64
	Description   string
	Cancellable   bool
}
```

**Step 2: Add test for parsing tasks API response**

Add to cluster_test.go:

```go
func TestParseTasksResponse(t *testing.T) {
	raw := `{
		"nodes": {
			"node-id-1": {
				"name": "es-node-1",
				"tasks": {
					"node-id-1:12345": {
						"node": "node-id-1",
						"id": 12345,
						"type": "transport",
						"action": "indices:data/write/reindex",
						"description": "reindex from [source] to [dest]",
						"start_time_in_millis": 1700000000000,
						"running_time_in_nanos": 120000000000,
						"cancellable": true,
						"cancelled": false
					},
					"node-id-1:12346": {
						"node": "node-id-1",
						"id": 12346,
						"type": "transport",
						"action": "cluster:monitor/tasks/lists",
						"start_time_in_millis": 1700000001000,
						"running_time_in_nanos": 1000000,
						"cancellable": false
					}
				}
			}
		}
	}`

	tasks, err := parseTasksResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1 (only cancellable reindex)", len(tasks))
	}

	if tasks[0].ID != "node-id-1:12345" {
		t.Errorf("ID = %q, want %q", tasks[0].ID, "node-id-1:12345")
	}
	if tasks[0].Action != "indices:data/write/reindex" {
		t.Errorf("Action = %q, want %q", tasks[0].Action, "indices:data/write/reindex")
	}
	if tasks[0].Node != "es-node-1" {
		t.Errorf("Node = %q, want %q", tasks[0].Node, "es-node-1")
	}
	if tasks[0].RunningTime != "2m 0s" {
		t.Errorf("RunningTime = %q, want %q", tasks[0].RunningTime, "2m 0s")
	}
	if tasks[0].Cancellable != true {
		t.Error("Cancellable should be true")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/es -run TestParseTasksResponse -v`
Expected: FAIL (parseTasksResponse not defined)

**Step 4: Implement parseTasksResponse**

Add to cluster.go:

```go
func parseTasksResponse(data []byte) ([]TaskInfo, error) {
	var response struct {
		Nodes map[string]struct {
			Name  string `json:"name"`
			Tasks map[string]struct {
				Node              string `json:"node"`
				ID                int64  `json:"id"`
				Action            string `json:"action"`
				Description       string `json:"description"`
				RunningTimeNanos  int64  `json:"running_time_in_nanos"`
				Cancellable       bool   `json:"cancellable"`
			} `json:"tasks"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing tasks response: %w", err)
	}

	actionPrefixes := []string{
		"indices:data/write/reindex",
		"indices:data/write/update/byquery",
		"indices:data/write/delete/byquery",
		"indices:admin/forcemerge",
		"cluster:admin/snapshot",
	}

	isTargetAction := func(action string) bool {
		for _, prefix := range actionPrefixes {
			if len(action) >= len(prefix) && action[:len(prefix)] == prefix {
				return true
			}
		}
		return false
	}

	var tasks []TaskInfo
	for nodeID, nodeData := range response.Nodes {
		for taskID, task := range nodeData.Tasks {
			if !task.Cancellable || !isTargetAction(task.Action) {
				continue
			}

			runningMs := task.RunningTimeNanos / 1_000_000
			tasks = append(tasks, TaskInfo{
				ID:            taskID,
				Action:        task.Action,
				Node:          nodeData.Name,
				Description:   task.Description,
				RunningTime:   formatDuration(runningMs),
				RunningTimeMs: runningMs,
				Cancellable:   task.Cancellable,
			})
			_ = nodeID
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].RunningTimeMs > tasks[j].RunningTimeMs
	})

	return tasks, nil
}

func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/es -run TestParseTasksResponse -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/es/cluster.go internal/es/cluster_test.go
git commit -m "feat(es): add TaskInfo struct and parsing"
```

---

## Task 2: Add FetchTasks and CancelTask methods

**Files:**
- Modify: `internal/es/cluster.go`

**Step 1: Add FetchTasks method**

Add to cluster.go:

```go
func (c *Client) FetchTasks(ctx context.Context) ([]TaskInfo, error) {
	res, err := c.es.Tasks.List(
		c.es.Tasks.List.WithContext(ctx),
		c.es.Tasks.List.WithDetailed(true),
		c.es.Tasks.List.WithActions("*reindex*", "*byquery*", "*forcemerge*", "*snapshot*"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching tasks: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tasks response: %w", err)
	}

	return parseTasksResponse(body)
}
```

**Step 2: Add CancelTask method**

Add to cluster.go:

```go
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	res, err := c.es.Tasks.Cancel(
		c.es.Tasks.Cancel.WithContext(ctx),
		c.es.Tasks.Cancel.WithTaskID(taskID),
	)
	if err != nil {
		return fmt.Errorf("cancelling task: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	return nil
}
```

**Step 3: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/es/cluster.go
git commit -m "feat(es): add FetchTasks and CancelTask methods"
```

---

## Task 3: Create TasksModel UI component

**Files:**
- Create: `internal/ui/tasks.go`

**Step 1: Create tasks.go with struct and constructor**

```go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type TasksModel struct {
	tasks       []es.TaskInfo
	selectedRow int
	scrollY     int
	width       int
	height      int
	loading     bool
	confirming  string
}

func NewTasks() TasksModel {
	return TasksModel{
		loading: true,
	}
}

func (m *TasksModel) SetTasks(tasks []es.TaskInfo) {
	m.tasks = tasks
	m.loading = false
	if m.selectedRow >= len(tasks) {
		m.selectedRow = max(0, len(tasks)-1)
	}
}

func (m *TasksModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m TasksModel) SelectedTaskID() string {
	if m.selectedRow >= 0 && m.selectedRow < len(m.tasks) {
		return m.tasks[m.selectedRow].ID
	}
	return ""
}

func (m *TasksModel) ClearConfirming() {
	m.confirming = ""
}
```

**Step 2: Verify build**

Run: `go build .`
Expected: FAIL (unused imports, Update/View not implemented)

**Step 3: Add Update method**

Add to tasks.go:

```go
func (m TasksModel) Update(msg tea.Msg) (TasksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming != "" {
			switch msg.String() {
			case "y", "Y":
				taskID := m.confirming
				m.confirming = ""
				return m, func() tea.Msg {
					return taskCancelRequestMsg{taskID: taskID}
				}
			case "n", "N", "esc":
				m.confirming = ""
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.selectedRow > 0 {
				m.selectedRow--
				if m.selectedRow < m.scrollY {
					m.scrollY = m.selectedRow
				}
			}
		case "down", "j":
			if m.selectedRow < len(m.tasks)-1 {
				m.selectedRow++
				maxVisible := m.maxVisibleRows()
				if m.selectedRow >= m.scrollY+maxVisible {
					m.scrollY = m.selectedRow - maxVisible + 1
				}
			}
		case "c":
			if m.selectedRow >= 0 && m.selectedRow < len(m.tasks) {
				m.confirming = m.tasks[m.selectedRow].ID
			}
		}
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			scrollAmount := 3
			maxScroll := m.maxScroll()
			if msg.Button == tea.MouseButtonWheelUp {
				m.scrollY = max(0, m.scrollY-scrollAmount)
			} else {
				m.scrollY = min(maxScroll, m.scrollY+scrollAmount)
			}
		}
	}
	return m, nil
}

func (m TasksModel) maxVisibleRows() int {
	rows := m.height - 6
	if rows < 1 {
		return 10
	}
	return rows
}

func (m TasksModel) maxScroll() int {
	maxScroll := len(m.tasks) - m.maxVisibleRows()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

type taskCancelRequestMsg struct {
	taskID string
}
```

**Step 4: Add View method**

Add to tasks.go:

```go
func (m TasksModel) View() string {
	if m.loading {
		return "Loading tasks..."
	}

	if len(m.tasks) == 0 {
		return lipgloss.NewStyle().
			Foreground(ColorGray).
			Padding(2).
			Render("No long-running tasks found.\n\nThis view shows: reindex, update-by-query, delete-by-query,\nforce merge, and snapshot operations.")
	}

	var b strings.Builder

	colWidths := []int{35, 25, 15, 12, 8}
	headers := []string{"action", "node", "description", "running", "cancel"}
	b.WriteString(m.renderHeader(headers, colWidths))

	maxVisible := m.maxVisibleRows()
	endIdx := m.scrollY + maxVisible
	if endIdx > len(m.tasks) {
		endIdx = len(m.tasks)
	}

	for i := m.scrollY; i < endIdx; i++ {
		task := m.tasks[i]
		isSelected := i == m.selectedRow
		isConfirming := m.confirming == task.ID

		rowStyle := lipgloss.NewStyle()
		if isConfirming {
			rowStyle = rowStyle.Background(ColorRed).Foreground(ColorOnAccent)
		} else if isSelected {
			rowStyle = rowStyle.Background(ColorBlue).Foreground(ColorOnAccent)
		}

		action := m.truncateAction(task.Action)
		desc := task.Description
		if len(desc) > colWidths[2]-2 {
			desc = desc[:colWidths[2]-5] + "..."
		}

		cancelText := "[c]"
		if isConfirming {
			cancelText = "y/n?"
		}

		row := fmt.Sprintf("%-*s %-*s %-*s %*s %*s",
			colWidths[0], action,
			colWidths[1], m.truncate(task.Node, colWidths[1]),
			colWidths[2], m.truncate(desc, colWidths[2]),
			colWidths[3], task.RunningTime,
			colWidths[4], cancelText,
		)

		b.WriteString(rowStyle.Render(row))
		b.WriteString("\n")
	}

	if m.confirming != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(ColorYellow).Render("Cancel this task? Press 'y' to confirm, 'n' or Esc to abort"))
	}

	return b.String()
}

func (m TasksModel) renderHeader(headers []string, widths []int) string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	var parts []string
	for i, h := range headers {
		parts = append(parts, fmt.Sprintf("%-*s", widths[i], h))
	}
	header := headerStyle.Render(strings.Join(parts, " "))

	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}
	totalWidth += len(widths) - 1

	return header + "\n" + strings.Repeat("-", totalWidth) + "\n"
}

func (m TasksModel) truncateAction(action string) string {
	parts := strings.Split(action, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return action
}

func (m TasksModel) truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-3]) + "..."
}
```

**Step 5: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add internal/ui/tasks.go
git commit -m "feat(ui): add TasksModel component"
```

---

## Task 4: Integrate Tasks tab into main model

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Add TabTasks constant**

Change the const block:

```go
const (
	TabOverview = iota
	TabWorkbench
	TabNodes
	TabTasks
)
```

**Step 2: Add tasks field and message types**

Add to Model struct (after nodes field):

```go
tasks     TasksModel
```

Add new message types (after nodesStateMsg):

```go
type tasksMsg struct{ tasks []es.TaskInfo }
type taskCancelledMsg struct{ err error }
```

**Step 3: Initialize tasks in New function**

Add to New function:

```go
tasks:     NewTasks(),
```

**Step 4: Add fetchTasks command**

Add after fetchNodes:

```go
func (m Model) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tasks, err := m.client.FetchTasks(ctx)
		if err != nil {
			return errMsg{err}
		}
		return tasksMsg{tasks}
	}
}

func (m Model) cancelTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.client.CancelTask(ctx, taskID)
		return taskCancelledMsg{err}
	}
}
```

**Step 5: Handle new messages in Update**

Add cases in Update switch:

```go
case tasksMsg:
	m.loading = false
	m.tasks.SetTasks(msg.tasks)
case taskCancelledMsg:
	m.tasks.ClearConfirming()
	if msg.err != nil {
		m.err = msg.err
	} else {
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
	}
case taskCancelRequestMsg:
	return m, m.cancelTask(msg.taskID)
```

**Step 6: Update tab cycling logic**

Update the tab key handling:

```go
case "tab":
	if m.activeTab == TabOverview && !m.overview.filterActive {
		m.activeTab = TabWorkbench
		m.workbench.Focus()
		return m, nil
	}
	if m.activeTab == TabWorkbench && m.workbench.focus != FocusPath && m.workbench.focus != FocusBody {
		m.activeTab = TabNodes
		m.workbench.Blur()
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
	}
	if m.activeTab == TabNodes {
		m.activeTab = TabTasks
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
	}
	if m.activeTab == TabTasks {
		m.activeTab = TabOverview
		return m, nil
	}
case "shift+tab":
	if m.activeTab == TabWorkbench {
		m.activeTab = TabOverview
		m.workbench.Blur()
		return m, nil
	}
	if m.activeTab == TabOverview && !m.overview.filterActive {
		m.activeTab = TabTasks
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
	}
	if m.activeTab == TabNodes {
		m.activeTab = TabWorkbench
		m.workbench.Focus()
		return m, nil
	}
	if m.activeTab == TabTasks {
		m.activeTab = TabNodes
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
	}
```

**Step 7: Update 'r' refresh handling**

Add case for TabTasks:

```go
case "r":
	if m.activeTab == TabOverview && !m.overview.filterActive {
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.connect())
	}
	if m.activeTab == TabNodes {
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
	}
	if m.activeTab == TabTasks {
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
	}
```

**Step 8: Update 'q' quit handling**

Add TabTasks case:

```go
if m.activeTab == TabTasks && m.tasks.confirming == "" {
	m.quitting = true
	return m, tea.Quit
}
```

**Step 9: Update View for tabs and content**

Add tasksTab to tabs rendering:

```go
tasksTab := InactiveTabStyle.Render("Tasks")
switch m.activeTab {
case TabOverview:
	overviewTab = ActiveTabStyle.Render("Overview")
case TabWorkbench:
	workbenchTab = ActiveTabStyle.Render("Workbench")
case TabNodes:
	nodesTab = ActiveTabStyle.Render("Nodes")
case TabTasks:
	tasksTab = ActiveTabStyle.Render("Tasks")
}
tabs := lipgloss.JoinHorizontal(lipgloss.Top, overviewTab, workbenchTab, nodesTab, tasksTab)
```

Add content case:

```go
case TabTasks:
	content = m.tasks.View()
```

**Step 10: Update status bar**

Add case:

```go
case TabTasks:
	statusText = "q: quit  Tab: overview  Shift+Tab: nodes  r: refresh  c: cancel  ↑↓: select"
```

**Step 11: Update WindowSizeMsg handler**

Add:

```go
m.tasks.SetSize(msg.Width, msg.Height-4)
```

**Step 12: Update mouse click handler for tab bar**

Update the click detection:

```go
if msg.Y == 1 {
	overviewWidth := lipgloss.Width(InactiveTabStyle.Render("Overview"))
	workbenchWidth := lipgloss.Width(InactiveTabStyle.Render("Workbench"))
	nodesWidth := lipgloss.Width(InactiveTabStyle.Render("Nodes"))

	if msg.X < overviewWidth {
		m.activeTab = TabOverview
		m.workbench.Blur()
	} else if msg.X < overviewWidth+workbenchWidth {
		m.activeTab = TabWorkbench
		m.workbench.Focus()
	} else if msg.X < overviewWidth+workbenchWidth+nodesWidth {
		m.activeTab = TabNodes
		m.workbench.Blur()
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
	} else {
		m.activeTab = TabTasks
		m.workbench.Blur()
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
	}
	return m, nil
}
```

**Step 13: Add delegation to tasks in Update**

Add case in delegation switch:

```go
case TabTasks:
	var cmd tea.Cmd
	m.tasks, cmd = m.tasks.Update(delegateMsg)
	if cmd != nil {
		return m, cmd
	}
```

**Step 14: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 15: Run all tests**

Run: `go test ./...`
Expected: PASS

**Step 16: Commit**

```bash
git add internal/ui/model.go
git commit -m "feat(ui): integrate Tasks tab into main model"
```

---

## Task 5: Update help overlay

**Files:**
- Modify: `internal/ui/help.go`

**Step 1: Add Tasks section to help**

Add new section after Nodes:

```go
{
	header: "Tasks",
	keys: [][]string{
		{"c", "Cancel selected task"},
		{"y/n", "Confirm/abort cancel"},
		{"up/down", "Select task"},
		{"r", "Refresh"},
	},
},
```

**Step 2: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/ui/help.go
git commit -m "docs(ui): add Tasks section to help overlay"
```

---

## Task 6: Update README documentation

**Files:**
- Modify: `README.md`

**Step 1: Update Features section**

Add after Nodes Tab bullet:

```markdown
- **Tasks Tab**: Monitor long-running operations
  - Reindex, update-by-query, delete-by-query tracking
  - Force merge and snapshot operations
  - Cancel with confirmation
```

**Step 2: Add Tasks Tab keybindings section**

Add after Nodes Tab section:

```markdown
### Tasks Tab

| Key | Action |
|-----|--------|
| `c` | Cancel selected task |
| `y` | Confirm cancel |
| `n` / `Esc` | Abort cancel |
| `Up/Down` | Select task |
```

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add Tasks tab to README"
```

---

## Verification

1. `go test ./...` - all tests pass
2. `docker compose up -d` - start ES
3. Start a reindex: `curl -X POST "localhost:9200/_reindex?wait_for_completion=false" -H 'Content-Type: application/json' -d '{"source":{"index":"products"},"dest":{"index":"products-copy"}}'`
4. `go run .` - launch TUI
5. Tab to Tasks - verify task appears in table
6. Press `c` - verify row highlights red, prompt appears
7. Press `n` - verify cancel aborted
8. Press `c` then `y` - verify task cancelled, list refreshes
9. Press `?` - verify Tasks section in help
