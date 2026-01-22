# Index and Alias Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add keyboard shortcuts to Overview tab for creating/deleting indices and adding/removing aliases via centered modal dialogs.

**Architecture:** Create reusable Modal component for text input dialogs. Add ES client methods for index/alias operations. Extend OverviewModel with modal state and action handlers.

**Tech Stack:** Go, Bubble Tea, Lipgloss, go-elasticsearch/v8

---

### Task 1: Add ES Client Methods for Index Operations

**Files:**
- Modify: `internal/es/cluster.go`

**Step 1: Add CreateIndex method**

Add after `CancelTask` method (around line 744):

```go
func (c *Client) CreateIndex(ctx context.Context, name string, shards, replicas int) error {
	body := fmt.Sprintf(`{"settings":{"number_of_shards":%d,"number_of_replicas":%d}}`, shards, replicas)
	res, err := c.es.Indices.Create(
		name,
		c.es.Indices.Create.WithContext(ctx),
		c.es.Indices.Create.WithBody(strings.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}
	return nil
}

func (c *Client) DeleteIndex(ctx context.Context, name string) error {
	res, err := c.es.Indices.Delete(
		[]string{name},
		c.es.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("deleting index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}
	return nil
}
```

**Step 2: Build to verify**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add internal/es/cluster.go
git commit -m "feat(es): add CreateIndex and DeleteIndex methods"
```

---

### Task 2: Add ES Client Methods for Alias Operations

**Files:**
- Modify: `internal/es/cluster.go`

**Step 1: Add AddAlias and RemoveAlias methods**

Add after DeleteIndex:

```go
func (c *Client) AddAlias(ctx context.Context, index, alias string) error {
	body := fmt.Sprintf(`{"actions":[{"add":{"index":"%s","alias":"%s"}}]}`, index, alias)
	res, err := c.es.Indices.UpdateAliases(
		strings.NewReader(body),
		c.es.Indices.UpdateAliases.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("adding alias: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}
	return nil
}

func (c *Client) RemoveAlias(ctx context.Context, index, alias string) error {
	body := fmt.Sprintf(`{"actions":[{"remove":{"index":"%s","alias":"%s"}}]}`, index, alias)
	res, err := c.es.Indices.UpdateAliases(
		strings.NewReader(body),
		c.es.Indices.UpdateAliases.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("removing alias: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}
	return nil
}
```

**Step 2: Build to verify**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add internal/es/cluster.go
git commit -m "feat(es): add AddAlias and RemoveAlias methods"
```

---

### Task 3: Create Modal Component

**Files:**
- Create: `internal/ui/modal.go`

**Step 1: Create modal.go**

```go
package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Modal struct {
	title    string
	prompt   string
	input    textinput.Model
	err      string
	done     bool
	cancelled bool
}

func NewModal(title, prompt string) *Modal {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 30

	return &Modal{
		title:  title,
		prompt: prompt,
		input:  ti,
	}
}

func (m *Modal) SetError(err string) {
	m.err = err
}

func (m *Modal) ClearError() {
	m.err = ""
}

func (m *Modal) Value() string {
	return m.input.Value()
}

func (m *Modal) SetValue(v string) {
	m.input.SetValue(v)
}

func (m *Modal) Done() bool {
	return m.done
}

func (m *Modal) Cancelled() bool {
	return m.cancelled
}

func (m *Modal) Reset(title, prompt string) {
	m.title = title
	m.prompt = prompt
	m.input.SetValue("")
	m.err = ""
	m.done = false
	m.cancelled = false
	m.input.Focus()
}

func (m *Modal) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.done = true
			return nil
		case "esc":
			m.cancelled = true
			return nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *Modal) View(width, height int) string {
	boxWidth := 50

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		MarginBottom(1)

	promptStyle := lipgloss.NewStyle().
		MarginBottom(1)

	errorStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		MarginTop(1)

	var content string
	content += titleStyle.Render(m.title) + "\n"
	content += promptStyle.Render(m.prompt) + "\n"
	content += m.input.View() + "\n"

	if m.err != "" {
		content += errorStyle.Render(m.err) + "\n"
	}

	content += helpStyle.Render("Enter: confirm | Esc: cancel")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
```

**Step 2: Build to verify**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add internal/ui/modal.go
git commit -m "feat(ui): add reusable Modal component"
```

---

### Task 4: Add Modal State to OverviewModel

**Files:**
- Modify: `internal/ui/overview.go`

**Step 1: Add state fields to OverviewModel struct**

Add after `height int` (line 24):

```go
	modal        *Modal
	modalAction  string
	modalStep    int
	createName   string
	createShards string
```

**Step 2: Add message types for async operations**

Add after the imports, before OverviewModel:

```go
type indexCreatedMsg struct{ err error }
type indexDeletedMsg struct{ err error }
type aliasAddedMsg struct{ err error }
type aliasRemovedMsg struct{ err error }
```

**Step 3: Build to verify**

Run: `go build .`
Expected: Success

**Step 4: Commit**

```bash
git add internal/ui/overview.go
git commit -m "feat(ui): add modal state to OverviewModel"
```

---

### Task 5: Add Keybindings for Index/Alias Actions

**Files:**
- Modify: `internal/ui/overview.go`

**Step 1: Add modal check at start of Update**

At the start of `func (m OverviewModel) Update`, before the switch statement:

```go
	if m.modal != nil {
		cmd := m.modal.Update(msg)
		if m.modal.Cancelled() {
			m.modal = nil
			m.modalAction = ""
			m.modalStep = 0
			return m, nil
		}
		if m.modal.Done() {
			return m.handleModalDone()
		}
		return m, cmd
	}
```

**Step 2: Add keybindings in the switch statement**

Add after the "1"-"9" case (around line 135):

```go
		case "c":
			m.modal = NewModal("Create Index", "Index name:")
			m.modalAction = "create"
			m.modalStep = 1
			return m, textinput.Blink
		case "d":
			if m.SelectedIndex() != "" {
				m.modal = NewModal("Delete Index", "Type '"+m.SelectedIndex()+"' to confirm:")
				m.modalAction = "delete"
				return m, textinput.Blink
			}
		case "a":
			if m.SelectedIndex() != "" {
				m.modal = NewModal("Add Alias", "Alias name:")
				m.modalAction = "addAlias"
				return m, textinput.Blink
			}
		case "A":
			if m.SelectedIndex() != "" {
				m.modal = NewModal("Remove Alias", "Alias name:")
				m.modalAction = "removeAlias"
				return m, textinput.Blink
			}
```

**Step 3: Build to verify**

Run: `go build .`
Expected: Success (modal handling method will be added next)

**Step 4: Commit**

```bash
git add internal/ui/overview.go
git commit -m "feat(ui): add keybindings for index/alias actions"
```

---

### Task 6: Add Modal Done Handler

**Files:**
- Modify: `internal/ui/overview.go`

**Step 1: Add handleModalDone method**

Add after the Update method:

```go
func (m OverviewModel) handleModalDone() (OverviewModel, tea.Cmd) {
	value := m.modal.Value()

	switch m.modalAction {
	case "create":
		switch m.modalStep {
		case 1:
			if value == "" {
				m.modal.SetError("Index name required")
				m.modal.done = false
				return m, nil
			}
			m.createName = value
			m.modal.Reset("Create Index", "Number of shards (default 1):")
			m.modalStep = 2
			return m, textinput.Blink
		case 2:
			m.createShards = value
			if m.createShards == "" {
				m.createShards = "1"
			}
			m.modal.Reset("Create Index", "Number of replicas (default 1):")
			m.modalStep = 3
			return m, textinput.Blink
		case 3:
			replicas := value
			if replicas == "" {
				replicas = "1"
			}
			m.modal = nil
			m.modalAction = ""
			m.modalStep = 0
			return m, m.createIndexCmd(m.createName, m.createShards, replicas)
		}

	case "delete":
		if value != m.SelectedIndex() {
			m.modal.SetError("Name does not match")
			m.modal.done = false
			return m, nil
		}
		indexName := m.SelectedIndex()
		m.modal = nil
		m.modalAction = ""
		return m, m.deleteIndexCmd(indexName)

	case "addAlias":
		if value == "" {
			m.modal.SetError("Alias name required")
			m.modal.done = false
			return m, nil
		}
		indexName := m.SelectedIndex()
		m.modal = nil
		m.modalAction = ""
		return m, m.addAliasCmd(indexName, value)

	case "removeAlias":
		if value == "" {
			m.modal.SetError("Alias name required")
			m.modal.done = false
			return m, nil
		}
		indexName := m.SelectedIndex()
		m.modal = nil
		m.modalAction = ""
		return m, m.removeAliasCmd(indexName, value)
	}

	m.modal = nil
	m.modalAction = ""
	return m, nil
}
```

**Step 2: Build to verify**

Run: `go build .`
Expected: Fails (command methods not yet added)

**Step 3: Commit anyway (partial)**

```bash
git add internal/ui/overview.go
git commit -m "feat(ui): add modal done handler"
```

---

### Task 7: Add Command Methods and Wire to Model

**Files:**
- Modify: `internal/ui/overview.go`
- Modify: `internal/ui/model.go`

**Step 1: Add client field and command methods to OverviewModel**

Add client field to OverviewModel struct:

```go
	client *es.Client
```

Add SetClient method:

```go
func (m *OverviewModel) SetClient(client *es.Client) {
	m.client = client
}
```

Add command methods after handleModalDone:

```go
func (m OverviewModel) createIndexCmd(name, shards, replicas string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return indexCreatedMsg{err: fmt.Errorf("no client")}
		}
		s, _ := strconv.Atoi(shards)
		r, _ := strconv.Atoi(replicas)
		if s < 1 {
			s = 1
		}
		if r < 0 {
			r = 0
		}
		err := m.client.CreateIndex(context.Background(), name, s, r)
		return indexCreatedMsg{err: err}
	}
}

func (m OverviewModel) deleteIndexCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return indexDeletedMsg{err: fmt.Errorf("no client")}
		}
		err := m.client.DeleteIndex(context.Background(), name)
		return indexDeletedMsg{err: err}
	}
}

func (m OverviewModel) addAliasCmd(index, alias string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return aliasAddedMsg{err: fmt.Errorf("no client")}
		}
		err := m.client.AddAlias(context.Background(), index, alias)
		return aliasAddedMsg{err: err}
	}
}

func (m OverviewModel) removeAliasCmd(index, alias string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return aliasRemovedMsg{err: fmt.Errorf("no client")}
		}
		err := m.client.RemoveAlias(context.Background(), index, alias)
		return aliasRemovedMsg{err: err}
	}
}
```

**Step 2: Add imports to overview.go**

Add to imports:
```go
	"context"
	"strconv"
```

**Step 3: Wire client in model.go**

In `internal/ui/model.go`, find where overview is initialized in `New()` and add after `overview.SetSize`:

```go
	overview.SetClient(client)
```

**Step 4: Build to verify**

Run: `go build .`
Expected: Success

**Step 5: Commit**

```bash
git add internal/ui/overview.go internal/ui/model.go
git commit -m "feat(ui): add command methods and wire client to overview"
```

---

### Task 8: Handle Async Messages and Refresh

**Files:**
- Modify: `internal/ui/overview.go`
- Modify: `internal/ui/model.go`

**Step 1: Handle messages in Overview Update**

Add cases in the Update switch for the message types (after the tea.MouseMsg case):

```go
	case indexCreatedMsg:
		if msg.err != nil {
			m.modal = NewModal("Error", msg.err.Error())
			m.modalAction = "error"
		}
		return m, nil
	case indexDeletedMsg:
		if msg.err != nil {
			m.modal = NewModal("Error", msg.err.Error())
			m.modalAction = "error"
		}
		m.selectedIndex = 0
		return m, nil
	case aliasAddedMsg:
		if msg.err != nil {
			m.modal = NewModal("Error", msg.err.Error())
			m.modalAction = "error"
		}
		return m, nil
	case aliasRemovedMsg:
		if msg.err != nil {
			m.modal = NewModal("Error", msg.err.Error())
			m.modalAction = "error"
		}
		return m, nil
```

**Step 2: Handle error modal dismiss**

In the modal check at start of Update, update the cancelled check to also handle error modals:

```go
		if m.modal.Cancelled() || (m.modal.Done() && m.modalAction == "error") {
			m.modal = nil
			m.modalAction = ""
			m.modalStep = 0
			return m, nil
		}
```

**Step 3: Handle messages in model.go and trigger refresh**

In `internal/ui/model.go`, find the Update method and add handling for these messages (they should trigger a cluster refresh):

```go
	case indexCreatedMsg, indexDeletedMsg, aliasAddedMsg, aliasRemovedMsg:
		m.overview, cmd = m.overview.Update(msg)
		if hasNoError(msg) {
			return m, tea.Batch(cmd, m.fetchCluster())
		}
		return m, cmd
```

Add helper function:

```go
func hasNoError(msg tea.Msg) bool {
	switch m := msg.(type) {
	case indexCreatedMsg:
		return m.err == nil
	case indexDeletedMsg:
		return m.err == nil
	case aliasAddedMsg:
		return m.err == nil
	case aliasRemovedMsg:
		return m.err == nil
	}
	return false
}
```

**Step 4: Export message types from overview.go**

Rename the message types to be exported (capital first letter):

```go
type IndexCreatedMsg struct{ Err error }
type IndexDeletedMsg struct{ Err error }
type AliasAddedMsg struct{ Err error }
type AliasRemovedMsg struct{ Err error }
```

Update all references in overview.go to use the new names and `.Err` field.

**Step 5: Build and test**

Run: `go build . && go test ./...`
Expected: Success

**Step 6: Commit**

```bash
git add internal/ui/overview.go internal/ui/model.go
git commit -m "feat(ui): handle async messages and trigger refresh"
```

---

### Task 9: Render Modal in Overview View

**Files:**
- Modify: `internal/ui/overview.go`

**Step 1: Update View method to render modal**

At the end of the View method, before `return b.String()`:

```go
	if m.modal != nil {
		return m.modal.View(m.width, m.height)
	}
```

**Step 2: Build and test**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add internal/ui/overview.go
git commit -m "feat(ui): render modal overlay in Overview"
```

---

### Task 10: Update Help and Documentation

**Files:**
- Modify: `internal/ui/help.go`
- Modify: `README.md`

**Step 1: Update help.go Overview section**

Find the Overview section and add keybindings:

```go
{"c", "Create index"},
{"d", "Delete selected index"},
{"a", "Add alias to index"},
{"A", "Remove alias from index"},
```

**Step 2: Update README.md**

Add to Overview Tab keybindings table:

```markdown
| `c` | Create new index |
| `d` | Delete selected index |
| `a` | Add alias to selected index |
| `A` | Remove alias from selected index |
```

**Step 3: Build and test**

Run: `go build . && go test ./...`
Expected: Success

**Step 4: Commit**

```bash
git add internal/ui/help.go README.md
git commit -m "docs: add index/alias management keybindings"
```

---

### Task 11: Manual Testing

**Step 1: Test create index**

1. Run `./stoptail http://localhost:9200`
2. Press `c`
3. Enter name "test-index", press Enter
4. Enter shards "2", press Enter
5. Enter replicas "1", press Enter
6. Verify index appears in list after refresh

**Step 2: Test delete index**

1. Select "test-index"
2. Press `d`
3. Type "test-index" to confirm
4. Verify index is removed

**Step 3: Test add alias**

1. Select an index
2. Press `a`
3. Enter alias name "test-alias"
4. Verify alias appears

**Step 4: Test remove alias**

1. Select the index with alias
2. Press `A`
3. Enter "test-alias"
4. Verify alias is removed

**Step 5: Test error handling**

1. Try to delete an index that doesn't exist
2. Verify error modal appears
3. Press Esc to dismiss

**Step 6: Commit any fixes**

If issues found, fix and commit.
