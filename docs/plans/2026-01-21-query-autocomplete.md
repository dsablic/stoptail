# Query Autocomplete Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add auto-completion to the Workbench body editor for ES query DSL keywords and index field names.

**Architecture:** New completion.go with context detection and DSL keywords, integrated into workbench.go with dropdown overlay. FetchMapping added to ES client for field names.

**Tech Stack:** Go, Bubble Tea, Lipgloss, go-elasticsearch/v8

---

## Task 1: Add CompletionItem and DSL keywords map

**Files:**
- Create: `internal/ui/completion.go`

**Step 1: Create completion.go with types and keywords**

```go
package ui

type CompletionItem struct {
	Text string
	Kind string
}

type JSONContext struct {
	Path    []string
	InKey   bool
	InValue bool
}

var dslKeywords = map[string][]CompletionItem{
	"": {
		{Text: "query", Kind: "keyword"},
		{Text: "aggs", Kind: "keyword"},
		{Text: "aggregations", Kind: "keyword"},
		{Text: "size", Kind: "keyword"},
		{Text: "from", Kind: "keyword"},
		{Text: "sort", Kind: "keyword"},
		{Text: "_source", Kind: "keyword"},
		{Text: "highlight", Kind: "keyword"},
		{Text: "track_total_hits", Kind: "keyword"},
	},
	"query": {
		{Text: "bool", Kind: "keyword"},
		{Text: "match", Kind: "keyword"},
		{Text: "match_all", Kind: "keyword"},
		{Text: "match_phrase", Kind: "keyword"},
		{Text: "multi_match", Kind: "keyword"},
		{Text: "term", Kind: "keyword"},
		{Text: "terms", Kind: "keyword"},
		{Text: "range", Kind: "keyword"},
		{Text: "exists", Kind: "keyword"},
		{Text: "prefix", Kind: "keyword"},
		{Text: "wildcard", Kind: "keyword"},
		{Text: "regexp", Kind: "keyword"},
		{Text: "fuzzy", Kind: "keyword"},
		{Text: "nested", Kind: "keyword"},
		{Text: "ids", Kind: "keyword"},
	},
	"bool": {
		{Text: "must", Kind: "keyword"},
		{Text: "should", Kind: "keyword"},
		{Text: "must_not", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "minimum_should_match", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"aggs": {
		{Text: "terms", Kind: "keyword"},
		{Text: "avg", Kind: "keyword"},
		{Text: "sum", Kind: "keyword"},
		{Text: "min", Kind: "keyword"},
		{Text: "max", Kind: "keyword"},
		{Text: "cardinality", Kind: "keyword"},
		{Text: "value_count", Kind: "keyword"},
		{Text: "stats", Kind: "keyword"},
		{Text: "date_histogram", Kind: "keyword"},
		{Text: "histogram", Kind: "keyword"},
		{Text: "range", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "nested", Kind: "keyword"},
	},
	"aggregations": {
		{Text: "terms", Kind: "keyword"},
		{Text: "avg", Kind: "keyword"},
		{Text: "sum", Kind: "keyword"},
		{Text: "min", Kind: "keyword"},
		{Text: "max", Kind: "keyword"},
		{Text: "cardinality", Kind: "keyword"},
		{Text: "value_count", Kind: "keyword"},
		{Text: "stats", Kind: "keyword"},
		{Text: "date_histogram", Kind: "keyword"},
		{Text: "histogram", Kind: "keyword"},
		{Text: "range", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "nested", Kind: "keyword"},
	},
	"match": {
		{Text: "query", Kind: "keyword"},
		{Text: "operator", Kind: "keyword"},
		{Text: "fuzziness", Kind: "keyword"},
		{Text: "analyzer", Kind: "keyword"},
	},
	"range": {
		{Text: "gte", Kind: "keyword"},
		{Text: "gt", Kind: "keyword"},
		{Text: "lte", Kind: "keyword"},
		{Text: "lt", Kind: "keyword"},
		{Text: "format", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"sort": {
		{Text: "order", Kind: "keyword"},
		{Text: "mode", Kind: "keyword"},
		{Text: "nested", Kind: "keyword"},
		{Text: "unmapped_type", Kind: "keyword"},
	},
	"highlight": {
		{Text: "fields", Kind: "keyword"},
		{Text: "pre_tags", Kind: "keyword"},
		{Text: "post_tags", Kind: "keyword"},
		{Text: "number_of_fragments", Kind: "keyword"},
	},
}

func GetKeywordsForContext(path []string) []CompletionItem {
	if len(path) == 0 {
		return dslKeywords[""]
	}
	lastKey := path[len(path)-1]
	if items, ok := dslKeywords[lastKey]; ok {
		return items
	}
	for i := len(path) - 2; i >= 0; i-- {
		if items, ok := dslKeywords[path[i]]; ok {
			return items
		}
	}
	return dslKeywords[""]
}
```

**Step 2: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/ui/completion.go
git commit -m "feat(ui): add completion types and DSL keywords"
```

---

## Task 2: Add JSON context detection

**Files:**
- Modify: `internal/ui/completion.go`
- Create: `internal/ui/completion_test.go`

**Step 1: Add test for context detection**

```go
package ui

import "testing"

func TestParseJSONContext(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantCtx JSONContext
	}{
		{
			name:    "empty",
			text:    `{`,
			wantCtx: JSONContext{Path: nil, InKey: true, InValue: false},
		},
		{
			name:    "after opening brace and quote",
			text:    `{"`,
			wantCtx: JSONContext{Path: nil, InKey: true, InValue: false},
		},
		{
			name:    "after key and colon",
			text:    `{"query":`,
			wantCtx: JSONContext{Path: []string{"query"}, InKey: false, InValue: true},
		},
		{
			name:    "nested in query",
			text:    `{"query":{"`,
			wantCtx: JSONContext{Path: []string{"query"}, InKey: true, InValue: false},
		},
		{
			name:    "nested in bool",
			text:    `{"query":{"bool":{"`,
			wantCtx: JSONContext{Path: []string{"query", "bool"}, InKey: true, InValue: false},
		},
		{
			name:    "inside array",
			text:    `{"query":{"bool":{"must":[{"`,
			wantCtx: JSONContext{Path: []string{"query", "bool", "must"}, InKey: true, InValue: false},
		},
		{
			name:    "after field in match",
			text:    `{"query":{"match":{"title":`,
			wantCtx: JSONContext{Path: []string{"query", "match", "title"}, InKey: false, InValue: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseJSONContext(tt.text)
			if got.InKey != tt.wantCtx.InKey {
				t.Errorf("InKey = %v, want %v", got.InKey, tt.wantCtx.InKey)
			}
			if got.InValue != tt.wantCtx.InValue {
				t.Errorf("InValue = %v, want %v", got.InValue, tt.wantCtx.InValue)
			}
			if len(got.Path) != len(tt.wantCtx.Path) {
				t.Errorf("Path = %v, want %v", got.Path, tt.wantCtx.Path)
			} else {
				for i := range got.Path {
					if got.Path[i] != tt.wantCtx.Path[i] {
						t.Errorf("Path[%d] = %v, want %v", i, got.Path[i], tt.wantCtx.Path[i])
					}
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestParseJSONContext -v`
Expected: FAIL (ParseJSONContext not defined)

**Step 3: Implement ParseJSONContext**

Add to completion.go:

```go
func ParseJSONContext(text string) JSONContext {
	ctx := JSONContext{}
	var path []string
	var currentKey string
	var inString bool
	var afterColon bool
	var depth int

	for i := 0; i < len(text); i++ {
		c := text[i]

		if inString {
			if c == '"' && (i == 0 || text[i-1] != '\\') {
				inString = false
				if afterColon {
					afterColon = false
				}
			} else {
				currentKey += string(c)
			}
			continue
		}

		switch c {
		case '"':
			inString = true
			currentKey = ""
		case ':':
			if currentKey != "" {
				path = append(path, currentKey)
			}
			afterColon = true
			currentKey = ""
		case '{':
			depth++
			afterColon = false
		case '}':
			depth--
			if len(path) > 0 {
				path = path[:len(path)-1]
			}
			afterColon = false
		case '[':
			afterColon = false
		case ']':
			afterColon = false
		case ',':
			afterColon = false
			currentKey = ""
		}
	}

	ctx.Path = path
	ctx.InKey = !afterColon && inString
	ctx.InValue = afterColon

	return ctx
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestParseJSONContext -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/completion.go internal/ui/completion_test.go
git commit -m "feat(ui): add JSON context detection"
```

---

## Task 3: Add FetchMapping to ES client

**Files:**
- Modify: `internal/es/cluster.go`
- Modify: `internal/es/cluster_test.go`

**Step 1: Add test for parsing mapping response**

Add to cluster_test.go:

```go
func TestParseMappingResponse(t *testing.T) {
	raw := `{
		"products": {
			"mappings": {
				"properties": {
					"title": {"type": "text"},
					"price": {"type": "float"},
					"category": {
						"type": "object",
						"properties": {
							"name": {"type": "keyword"},
							"id": {"type": "integer"}
						}
					},
					"tags": {"type": "keyword"}
				}
			}
		}
	}`

	fields, err := parseMappingResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	expected := []string{"title", "price", "category.name", "category.id", "tags"}
	if len(fields) != len(expected) {
		t.Fatalf("got %d fields, want %d: %v", len(fields), len(expected), fields)
	}

	fieldMap := make(map[string]bool)
	for _, f := range fields {
		fieldMap[f] = true
	}

	for _, e := range expected {
		if !fieldMap[e] {
			t.Errorf("missing field %q", e)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/es -run TestParseMappingResponse -v`
Expected: FAIL (parseMappingResponse not defined)

**Step 3: Implement parseMappingResponse and FetchMapping**

Add to cluster.go:

```go
func parseMappingResponse(data []byte) ([]string, error) {
	var response map[string]struct {
		Mappings struct {
			Properties map[string]interface{} `json:"properties"`
		} `json:"mappings"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing mapping response: %w", err)
	}

	var fields []string
	for _, indexData := range response {
		fields = extractFields(indexData.Mappings.Properties, "")
		break
	}

	return fields, nil
}

func extractFields(properties map[string]interface{}, prefix string) []string {
	var fields []string

	for name, prop := range properties {
		fieldName := name
		if prefix != "" {
			fieldName = prefix + "." + name
		}

		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}

		if nested, ok := propMap["properties"].(map[string]interface{}); ok {
			fields = append(fields, extractFields(nested, fieldName)...)
		} else {
			fields = append(fields, fieldName)
		}
	}

	return fields
}

func (c *Client) FetchMapping(ctx context.Context, index string) ([]string, error) {
	res, err := c.es.Indices.GetMapping(
		c.es.Indices.GetMapping.WithContext(ctx),
		c.es.Indices.GetMapping.WithIndex(index),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching mapping: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES error %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading mapping response: %w", err)
	}

	return parseMappingResponse(body)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/es -run TestParseMappingResponse -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/es/cluster.go internal/es/cluster_test.go
git commit -m "feat(es): add FetchMapping for index field names"
```

---

## Task 4: Add CompletionState to WorkbenchModel

**Files:**
- Modify: `internal/ui/workbench.go`

**Step 1: Add completion state fields**

Add to WorkbenchModel struct:

```go
completion     CompletionState
fieldCache     map[string][]CompletionItem
lastIndex      string
```

Add CompletionState type to completion.go:

```go
type CompletionState struct {
	Active      bool
	Items       []CompletionItem
	Filtered    []CompletionItem
	SelectedIdx int
	TriggerCol  int
	Query       string
}

func (c *CompletionState) Filter(query string) {
	c.Query = query
	c.Filtered = nil
	c.SelectedIdx = 0
	query = strings.ToLower(query)
	for _, item := range c.Items {
		if strings.HasPrefix(strings.ToLower(item.Text), query) {
			c.Filtered = append(c.Filtered, item)
		}
	}
	if len(c.Filtered) == 0 {
		c.Active = false
	}
}

func (c *CompletionState) MoveUp() {
	if c.SelectedIdx > 0 {
		c.SelectedIdx--
	}
}

func (c *CompletionState) MoveDown() {
	if c.SelectedIdx < len(c.Filtered)-1 {
		c.SelectedIdx++
	}
}

func (c *CompletionState) Selected() *CompletionItem {
	if c.SelectedIdx >= 0 && c.SelectedIdx < len(c.Filtered) {
		return &c.Filtered[c.SelectedIdx]
	}
	return nil
}

func (c *CompletionState) Close() {
	c.Active = false
	c.Items = nil
	c.Filtered = nil
	c.SelectedIdx = 0
	c.Query = ""
}
```

**Step 2: Initialize in NewWorkbench**

Add to NewWorkbench return:

```go
fieldCache: make(map[string][]CompletionItem),
```

**Step 3: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/ui/completion.go internal/ui/workbench.go
git commit -m "feat(ui): add CompletionState to workbench"
```

---

## Task 5: Trigger completions on `"` and `:`

**Files:**
- Modify: `internal/ui/workbench.go`

**Step 1: Add trigger logic in Update**

In workbench.go Update function, in the FocusBody case after `m.body, cmd = m.body.Update(msg)`, add:

```go
if keyMsg, ok := msg.(tea.KeyMsg); ok {
	key := keyMsg.String()
	if !m.completion.Active {
		if key == `"` || key == ":" || key == "shift+;" {
			m.triggerCompletion()
		}
	} else {
		switch key {
		case "up":
			m.completion.MoveUp()
			return m, nil
		case "down":
			m.completion.MoveDown()
			return m, nil
		case "enter", "tab":
			m.acceptCompletion()
			return m, nil
		case "esc":
			m.completion.Close()
			return m, nil
		default:
			if len(key) == 1 || key == "backspace" {
				col := m.body.Col()
				if col > m.completion.TriggerCol {
					query := m.getCompletionQuery()
					m.completion.Filter(query)
				} else {
					m.completion.Close()
				}
			}
		}
	}
}
```

**Step 2: Add helper methods**

Add to workbench.go:

```go
func (m *WorkbenchModel) triggerCompletion() {
	text := m.body.Value()
	lines := strings.Split(text, "\n")
	row := m.body.Line()
	col := m.body.Col()

	textUpToCursor := ""
	for i := 0; i < row && i < len(lines); i++ {
		textUpToCursor += lines[i] + "\n"
	}
	if row < len(lines) {
		if col <= len(lines[row]) {
			textUpToCursor += lines[row][:col]
		} else {
			textUpToCursor += lines[row]
		}
	}

	ctx := ParseJSONContext(textUpToCursor)

	var items []CompletionItem
	keywords := GetKeywordsForContext(ctx.Path)
	items = append(items, keywords...)

	if fields, ok := m.fieldCache[m.lastIndex]; ok {
		items = append(items, fields...)
	}

	if len(items) == 0 {
		return
	}

	m.completion.Active = true
	m.completion.Items = items
	m.completion.Filtered = items
	m.completion.SelectedIdx = 0
	m.completion.TriggerCol = col
	m.completion.Query = ""
}

func (m *WorkbenchModel) getCompletionQuery() string {
	lines := strings.Split(m.body.Value(), "\n")
	row := m.body.Line()
	col := m.body.Col()

	if row >= len(lines) {
		return ""
	}

	line := lines[row]
	if col > len(line) {
		col = len(line)
	}

	start := m.completion.TriggerCol
	if start > col {
		return ""
	}

	return line[start:col]
}

func (m *WorkbenchModel) acceptCompletion() {
	selected := m.completion.Selected()
	if selected == nil {
		m.completion.Close()
		return
	}

	query := m.getCompletionQuery()
	insertion := selected.Text[len(query):]

	lines := strings.Split(m.body.Value(), "\n")
	row := m.body.Line()
	col := m.body.Col()

	if row < len(lines) {
		line := lines[row]
		if col <= len(line) {
			lines[row] = line[:col] + insertion + `": ` + line[col:]
		}
	}

	m.body.SetValue(strings.Join(lines, "\n"))
	newCol := col + len(insertion) + 3
	m.body.SetCursor(newCol)

	m.completion.Close()
}

func (m *WorkbenchModel) extractIndexFromPath() string {
	path := m.path.Value()
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, "_") {
			return part
		}
	}
	return ""
}
```

**Step 3: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/ui/workbench.go
git commit -m "feat(ui): trigger completions on quote and colon"
```

---

## Task 6: Render completion dropdown

**Files:**
- Modify: `internal/ui/workbench.go`

**Step 1: Add dropdown rendering in View**

In the body pane section, after `bodyPaneContent := ...`, add overlay logic:

```go
func (m WorkbenchModel) renderCompletionDropdown() string {
	if !m.completion.Active || len(m.completion.Filtered) == 0 {
		return ""
	}

	maxVisible := 8
	items := m.completion.Filtered
	if len(items) > maxVisible {
		items = items[:maxVisible]
	}

	var lines []string
	for i, item := range items {
		text := fmt.Sprintf(" %s ", item.Text)
		if item.Kind != "" {
			text = fmt.Sprintf(" %s (%s) ", item.Text, item.Kind)
		}

		style := lipgloss.NewStyle().Background(InactiveBg)
		if i == m.completion.SelectedIdx {
			style = style.Background(ColorBlue).Foreground(ColorOnAccent)
		}
		lines = append(lines, style.Render(text))
	}

	dropdown := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(strings.Join(lines, "\n"))

	return dropdown
}
```

**Step 2: Position dropdown in View**

In the View function, after building bodyPane, check if completion is active and overlay the dropdown. This requires modifying how the body pane is rendered to include the dropdown positioned at cursor.

For simplicity, render dropdown below the body pane initially:

```go
if m.completion.Active {
	dropdown := m.renderCompletionDropdown()
	bodyPaneContent = lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Body"),
		m.body.View(),
		dropdown)
}
```

**Step 3: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/ui/workbench.go
git commit -m "feat(ui): render completion dropdown"
```

---

## Task 7: Fetch and cache field mappings

**Files:**
- Modify: `internal/ui/workbench.go`
- Modify: `internal/ui/model.go`

**Step 1: Add message type for mapping result**

Add to workbench.go:

```go
type mappingResultMsg struct {
	index  string
	fields []string
}
```

**Step 2: Add fetch command**

Add to workbench.go:

```go
func (m WorkbenchModel) fetchMapping(index string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}
		ctx := context.Background()
		fields, err := m.client.FetchMapping(ctx, index)
		if err != nil {
			return nil
		}
		return mappingResultMsg{index: index, fields: fields}
	}
}
```

**Step 3: Handle mapping result in Update**

Add case in Update:

```go
case mappingResultMsg:
	items := make([]CompletionItem, len(msg.fields))
	for i, f := range msg.fields {
		items[i] = CompletionItem{Text: f, Kind: "field"}
	}
	m.fieldCache[msg.index] = items
	return m, nil
```

**Step 4: Trigger fetch when path changes**

In the path input handling, after path is updated, check if index changed:

```go
func (m *WorkbenchModel) checkIndexChange() tea.Cmd {
	index := m.extractIndexFromPath()
	if index != "" && index != m.lastIndex {
		m.lastIndex = index
		if _, ok := m.fieldCache[index]; !ok {
			return m.fetchMapping(index)
		}
	}
	return nil
}
```

Call this after path updates in Update.

**Step 5: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add internal/ui/workbench.go
git commit -m "feat(ui): fetch and cache index field mappings"
```

---

## Task 8: Fix Tab behavior

**Files:**
- Modify: `internal/ui/workbench.go`
- Modify: `internal/ui/model.go`

**Step 1: Add HasActiveInput method**

Add to workbench.go:

```go
func (m WorkbenchModel) HasActiveInput() bool {
	return m.focus == FocusPath || m.focus == FocusBody
}
```

**Step 2: Update model.go tab handling**

In model.go, modify the "tab" case:

```go
case "tab":
	if m.activeTab == TabWorkbench && m.workbench.HasActiveInput() {
		// Let workbench handle tab internally
		break
	}
	// ... existing tab switching logic
```

**Step 3: Update workbench tab handling**

In workbench.go, change tab behavior to not cycle focus when in body:

```go
case "tab":
	if m.completion.Active {
		m.acceptCompletion()
		return m, nil
	}
	if m.focus == FocusBody {
		// Don't cycle, let main model handle tab
		return m, nil
	}
	m.cycleFocus()
	return m, nil
```

**Step 4: Verify build**

Run: `go build .`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add internal/ui/workbench.go internal/ui/model.go
git commit -m "fix(ui): tab switches main tabs when not in text input"
```

---

## Task 9: Update help and documentation

**Files:**
- Modify: `internal/ui/help.go`
- Modify: `README.md`

**Step 1: Update Workbench section in help.go**

Add to Workbench keys:

```go
{"\"/:\"", "Trigger autocomplete"},
{"↑/↓", "Navigate completions"},
{"Enter/Tab", "Accept completion"},
{"Esc", "Dismiss completions"},
```

**Step 2: Update README**

Add to Workbench Tab keybindings:

```markdown
| `"` or `:` | Trigger autocomplete |
| `Up/Down` | Navigate completions |
| `Enter/Tab` | Accept completion |
| `Esc` | Dismiss completions |
```

Add to Features:

```markdown
- **Query Autocomplete**: ES DSL keywords and index field names
```

**Step 3: Commit**

```bash
git add internal/ui/help.go README.md
git commit -m "docs: add autocomplete keybindings"
```

---

## Verification

1. `go test ./...` - all tests pass
2. `docker compose up -d` - start ES
3. `go run .` - launch TUI
4. Navigate to Workbench, set path to `/products/_search`
5. In body, type `{"` - verify dropdown with root keywords
6. Type `q` - verify filtered to `query`
7. Press Enter - verify `query": ` inserted
8. Type `{"` - verify query-level completions
9. Verify field names appear (title, price, etc. from products index)
10. Press Tab on Overview - verify switches to Workbench
11. Type in body, press Tab - verify stays in Workbench (doesn't switch tab)
