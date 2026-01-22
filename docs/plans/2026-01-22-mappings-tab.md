# Mappings Tab Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Mappings tab to browse index mappings and custom analyzers with two-pane layout.

**Architecture:** New `MappingsModel` in `internal/ui/mappings.go` following existing tab patterns. ES client methods fetch mappings and settings per-index on selection. Two-pane split with index list (left) and mappings view (right).

**Tech Stack:** Go, Bubble Tea, Lipgloss, Elasticsearch GET `/{index}/_mapping` and `/{index}/_settings` APIs.

---

### Task 0: Create test index with custom analyzers

**Step 1: Create index with custom analyzers and nested mappings**

```bash
curl -X PUT 'localhost:9200/test_mappings' -H 'Content-Type: application/json' -d '{
  "settings": {
    "analysis": {
      "analyzer": {
        "custom_text": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "snowball_filter"]
        },
        "edge_ngram_analyzer": {
          "type": "custom",
          "tokenizer": "edge_ngram_tokenizer",
          "filter": ["lowercase"]
        }
      },
      "tokenizer": {
        "edge_ngram_tokenizer": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 10,
          "token_chars": ["letter", "digit"]
        }
      },
      "filter": {
        "snowball_filter": {
          "type": "snowball",
          "language": "English"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "title": {
        "type": "text",
        "analyzer": "custom_text",
        "fields": {
          "keyword": { "type": "keyword" },
          "autocomplete": { "type": "text", "analyzer": "edge_ngram_analyzer" }
        }
      },
      "description": {
        "type": "text",
        "norms": false
      },
      "price": {
        "type": "float",
        "doc_values": false
      },
      "in_stock": {
        "type": "boolean",
        "index": false
      },
      "metadata": {
        "type": "object",
        "properties": {
          "created_at": { "type": "date" },
          "updated_at": { "type": "date" },
          "tags": { "type": "keyword" }
        }
      },
      "address": {
        "properties": {
          "street": { "type": "text" },
          "city": { "type": "keyword" },
          "zip": { "type": "keyword", "index": false },
          "geo": {
            "properties": {
              "lat": { "type": "float" },
              "lon": { "type": "float" }
            }
          }
        }
      }
    }
  }
}'
```

**Step 2: Verify it was created**

```bash
curl -s 'localhost:9200/test_mappings/_mapping' | jq .
curl -s 'localhost:9200/test_mappings/_settings' | jq '.test_mappings.settings.index.analysis'
```

---

### Task 1: Add data structures for mappings

**Files:**
- Modify: `internal/es/cluster.go`

**Step 1: Add MappingField struct**

Add after `TaskInfo` struct (~line 95):

```go
type MappingField struct {
	Name       string
	Type       string
	Properties map[string]string
	Children   []MappingField
}

type AnalyzerInfo struct {
	Name     string
	Kind     string
	Settings map[string]string
}

type IndexMappings struct {
	IndexName  string
	FieldCount int
	Fields     []MappingField
	Analyzers  []AnalyzerInfo
}
```

**Step 2: Commit**

```bash
git add internal/es/cluster.go
git commit -m "feat(es): add mapping data structures"
```

---

### Task 2: Add FetchIndexMappings method

**Files:**
- Modify: `internal/es/cluster.go`
- Create: `internal/es/mappings_test.go`

**Step 1: Write the test**

Create `internal/es/mappings_test.go`:

```go
package es

import (
	"encoding/json"
	"testing"
)

func TestParseMappingResponse(t *testing.T) {
	raw := `{
		"products": {
			"mappings": {
				"properties": {
					"name": {"type": "text"},
					"price": {"type": "float"},
					"category": {
						"type": "text",
						"analyzer": "custom_analyzer",
						"fields": {
							"keyword": {"type": "keyword"}
						}
					},
					"address": {
						"properties": {
							"city": {"type": "keyword"},
							"zip": {"type": "keyword", "index": false}
						}
					}
				}
			}
		}
	}`

	var response map[string]struct {
		Mappings struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	props := response["products"].Mappings.Properties
	if len(props) != 4 {
		t.Errorf("got %d properties, want 4", len(props))
	}
}

func TestParseMappingFields(t *testing.T) {
	fields := parseMappingProperties(map[string]json.RawMessage{
		"name":  json.RawMessage(`{"type": "text"}`),
		"price": json.RawMessage(`{"type": "float"}`),
	}, "")

	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(fields))
	}

	nameFound := false
	for _, f := range fields {
		if f.Name == "name" && f.Type == "text" {
			nameFound = true
		}
	}
	if !nameFound {
		t.Error("name field not found or wrong type")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/es/... -run TestParseMapping -v
```

Expected: FAIL with `parseMappingProperties` undefined

**Step 3: Implement parseMappingProperties helper**

Add to `internal/es/cluster.go`:

```go
func parseMappingProperties(props map[string]json.RawMessage, prefix string) []MappingField {
	var fields []MappingField

	for name, raw := range props {
		var prop struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
			Analyzer   string                     `json:"analyzer"`
			Index      *bool                      `json:"index"`
			DocValues  *bool                      `json:"doc_values"`
			Norms      *bool                      `json:"norms"`
			Store      *bool                      `json:"store"`
			NullValue  interface{}                `json:"null_value"`
			Fields     map[string]json.RawMessage `json:"fields"`
		}
		if err := json.Unmarshal(raw, &prop); err != nil {
			continue
		}

		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}

		field := MappingField{
			Name:       fullName,
			Type:       prop.Type,
			Properties: make(map[string]string),
		}

		if prop.Type == "" && prop.Properties != nil {
			field.Type = "object"
		}

		if prop.Analyzer != "" {
			field.Properties["analyzer"] = prop.Analyzer
		}
		if prop.Index != nil && !*prop.Index {
			field.Properties["index"] = "false"
		}
		if prop.DocValues != nil && !*prop.DocValues {
			field.Properties["doc_values"] = "false"
		}
		if prop.Norms != nil && !*prop.Norms {
			field.Properties["norms"] = "false"
		}
		if prop.Store != nil && *prop.Store {
			field.Properties["store"] = "true"
		}
		if prop.NullValue != nil {
			field.Properties["null_value"] = fmt.Sprintf("%v", prop.NullValue)
		}

		if prop.Properties != nil {
			field.Children = parseMappingProperties(prop.Properties, fullName)
		}

		if prop.Fields != nil {
			for subName, subRaw := range prop.Fields {
				subFields := parseMappingProperties(map[string]json.RawMessage{subName: subRaw}, fullName)
				field.Children = append(field.Children, subFields...)
			}
		}

		fields = append(fields, field)
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields
}
```

**Step 4: Run tests**

```bash
go test ./internal/es/... -run TestParseMapping -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/es/cluster.go internal/es/mappings_test.go
git commit -m "feat(es): add mapping properties parser"
```

---

### Task 3: Add FetchIndexMappings client method

**Files:**
- Modify: `internal/es/cluster.go`

**Step 1: Add FetchIndexMappings method**

```go
func (c *Client) FetchIndexMappings(ctx context.Context, indexName string) (*IndexMappings, error) {
	res, err := c.es.Indices.GetMapping(
		c.es.Indices.GetMapping.WithContext(ctx),
		c.es.Indices.GetMapping.WithIndex(indexName),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching mappings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading mappings response: %w", err)
	}

	var response map[string]struct {
		Mappings struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing mappings: %w", err)
	}

	indexData, ok := response[indexName]
	if !ok {
		return nil, fmt.Errorf("index %s not found in response", indexName)
	}

	fields := parseMappingProperties(indexData.Mappings.Properties, "")
	flatFields := flattenFields(fields)

	return &IndexMappings{
		IndexName:  indexName,
		FieldCount: len(flatFields),
		Fields:     fields,
	}, nil
}

func flattenFields(fields []MappingField) []MappingField {
	var result []MappingField
	for _, f := range fields {
		result = append(result, f)
		if len(f.Children) > 0 {
			result = append(result, flattenFields(f.Children)...)
		}
	}
	return result
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/es/cluster.go
git commit -m "feat(es): add FetchIndexMappings method"
```

---

### Task 4: Add FetchIndexAnalyzers method

**Files:**
- Modify: `internal/es/cluster.go`

**Step 1: Add FetchIndexAnalyzers method**

```go
func (c *Client) FetchIndexAnalyzers(ctx context.Context, indexName string) ([]AnalyzerInfo, error) {
	res, err := c.es.Indices.GetSettings(
		c.es.Indices.GetSettings.WithContext(ctx),
		c.es.Indices.GetSettings.WithIndex(indexName),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading settings response: %w", err)
	}

	var response map[string]struct {
		Settings struct {
			Index struct {
				Analysis struct {
					Analyzer  map[string]json.RawMessage `json:"analyzer"`
					Tokenizer map[string]json.RawMessage `json:"tokenizer"`
					Filter    map[string]json.RawMessage `json:"filter"`
				} `json:"analysis"`
			} `json:"index"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	var analyzers []AnalyzerInfo
	indexData := response[indexName]

	for name, raw := range indexData.Settings.Index.Analysis.Analyzer {
		analyzers = append(analyzers, parseAnalyzerInfo(name, "analyzer", raw))
	}
	for name, raw := range indexData.Settings.Index.Analysis.Tokenizer {
		analyzers = append(analyzers, parseAnalyzerInfo(name, "tokenizer", raw))
	}
	for name, raw := range indexData.Settings.Index.Analysis.Filter {
		analyzers = append(analyzers, parseAnalyzerInfo(name, "filter", raw))
	}

	sort.Slice(analyzers, func(i, j int) bool {
		if analyzers[i].Kind != analyzers[j].Kind {
			kindOrder := map[string]int{"analyzer": 0, "tokenizer": 1, "filter": 2}
			return kindOrder[analyzers[i].Kind] < kindOrder[analyzers[j].Kind]
		}
		return analyzers[i].Name < analyzers[j].Name
	})

	return analyzers, nil
}

func parseAnalyzerInfo(name, kind string, raw json.RawMessage) AnalyzerInfo {
	var settings map[string]interface{}
	json.Unmarshal(raw, &settings)

	info := AnalyzerInfo{
		Name:     name,
		Kind:     kind,
		Settings: make(map[string]string),
	}

	for k, v := range settings {
		switch val := v.(type) {
		case string:
			info.Settings[k] = val
		case []interface{}:
			strs := make([]string, len(val))
			for i, s := range val {
				strs[i] = fmt.Sprintf("%v", s)
			}
			info.Settings[k] = strings.Join(strs, ", ")
		default:
			info.Settings[k] = fmt.Sprintf("%v", v)
		}
	}

	return info
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/es/cluster.go
git commit -m "feat(es): add FetchIndexAnalyzers method"
```

---

### Task 5: Create MappingsModel skeleton

**Files:**
- Create: `internal/ui/mappings.go`

**Step 1: Create mappings.go with basic structure**

```go
package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type MappingsPane int

const (
	PaneIndices MappingsPane = iota
	PaneMappings
)

type MappingsModel struct {
	indices       []es.IndexInfo
	selectedIndex int
	scrollY       int
	width         int
	height        int
	activePane    MappingsPane
	filterActive  bool
	filterText    string
	treeView      bool

	mappings      *es.IndexMappings
	analyzers     []es.AnalyzerInfo
	mappingScroll int
	loadingIndex  string
}

func NewMappings() MappingsModel {
	return MappingsModel{
		activePane: PaneIndices,
	}
}

func (m *MappingsModel) SetIndices(indices []es.IndexInfo) {
	m.indices = indices
	if m.selectedIndex >= len(indices) {
		m.selectedIndex = max(0, len(indices)-1)
	}
}

func (m *MappingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MappingsModel) SetMappings(mappings *es.IndexMappings, analyzers []es.AnalyzerInfo) {
	m.mappings = mappings
	m.analyzers = analyzers
	m.loadingIndex = ""
	m.mappingScroll = 0
}

func (m *MappingsModel) SelectedIndexName() string {
	filtered := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(filtered) {
		return filtered[m.selectedIndex].Name
	}
	return ""
}

func (m *MappingsModel) IsLoading() bool {
	return m.loadingIndex != ""
}

func (m *MappingsModel) SetLoading(indexName string) {
	m.loadingIndex = indexName
}

func (m MappingsModel) filteredIndices() []es.IndexInfo {
	if m.filterText == "" {
		return m.indices
	}
	var filtered []es.IndexInfo
	for _, idx := range m.indices {
		if strings.Contains(strings.ToLower(idx.Name), strings.ToLower(m.filterText)) {
			filtered = append(filtered, idx)
		}
	}
	return filtered
}

func (m MappingsModel) Update(msg tea.Msg) (MappingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "esc":
				m.filterActive = false
				m.filterText = ""
				m.selectedIndex = 0
			case "enter":
				m.filterActive = false
			case "backspace":
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.selectedIndex = 0
				}
			default:
				if len(msg.String()) == 1 {
					m.filterText += msg.String()
					m.selectedIndex = 0
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			if m.activePane == PaneIndices {
				m.filterActive = true
			}
		case "t":
			if m.activePane == PaneMappings {
				m.treeView = !m.treeView
			}
		case "left", "h":
			m.activePane = PaneIndices
		case "right", "l", "enter":
			if m.activePane == PaneIndices {
				m.activePane = PaneMappings
				indexName := m.SelectedIndexName()
				if indexName != "" && (m.mappings == nil || m.mappings.IndexName != indexName) {
					m.loadingIndex = indexName
					return m, func() tea.Msg {
						return fetchMappingsMsg{indexName: indexName}
					}
				}
			}
		case "up", "k":
			if m.activePane == PaneIndices {
				if m.selectedIndex > 0 {
					m.selectedIndex--
				}
			} else {
				if m.mappingScroll > 0 {
					m.mappingScroll--
				}
			}
		case "down", "j":
			if m.activePane == PaneIndices {
				filtered := m.filteredIndices()
				if m.selectedIndex < len(filtered)-1 {
					m.selectedIndex++
				}
			} else {
				m.mappingScroll++
			}
		}
	}
	return m, nil
}

type fetchMappingsMsg struct {
	indexName string
}

func (m MappingsModel) View() string {
	leftWidth := m.width / 4
	rightWidth := m.width - leftWidth - 3

	leftPane := m.renderIndexList(leftWidth)
	rightPane := m.renderMappingsPane(rightWidth)

	leftStyle := lipgloss.NewStyle().Width(leftWidth).Height(m.height)
	rightStyle := lipgloss.NewStyle().Width(rightWidth).Height(m.height)

	if m.activePane == PaneIndices {
		leftStyle = leftStyle.BorderStyle(lipgloss.RoundedBorder()).BorderForeground(ColorBlue)
		rightStyle = rightStyle.BorderStyle(lipgloss.RoundedBorder()).BorderForeground(ColorGray)
	} else {
		leftStyle = leftStyle.BorderStyle(lipgloss.RoundedBorder()).BorderForeground(ColorGray)
		rightStyle = rightStyle.BorderStyle(lipgloss.RoundedBorder()).BorderForeground(ColorBlue)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(leftPane),
		" ",
		rightStyle.Render(rightPane),
	)
}

func (m MappingsModel) renderIndexList(width int) string {
	var b strings.Builder

	if m.filterActive {
		b.WriteString("/" + m.filterText + "_\n")
	} else if m.filterText != "" {
		b.WriteString("/" + m.filterText + "\n")
	}

	filtered := m.filteredIndices()
	innerWidth := width - 4

	for i, idx := range filtered {
		name := idx.Name
		if len(name) > innerWidth-10 {
			name = name[:innerWidth-13] + "..."
		}

		line := name
		style := lipgloss.NewStyle()
		if i == m.selectedIndex {
			style = style.Background(ColorBlue).Foreground(ColorOnAccent)
		}
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m MappingsModel) renderMappingsPane(width int) string {
	if m.loadingIndex != "" {
		return "Loading mappings for " + m.loadingIndex + "..."
	}

	if m.mappings == nil {
		return lipgloss.NewStyle().Foreground(ColorGray).Render("Select an index to view mappings")
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true)
	b.WriteString(headerStyle.Render(m.mappings.IndexName))
	b.WriteString("\n\n")

	if m.treeView {
		b.WriteString(m.renderFieldsTree(m.mappings.Fields, 0, width))
	} else {
		b.WriteString(m.renderFieldsFlat(width))
	}

	if len(m.analyzers) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render(strings.Repeat("-", width-4) + " Custom Analyzers "))
		b.WriteString("\n\n")
		b.WriteString(m.renderAnalyzers(width))
	}

	return b.String()
}

func (m MappingsModel) renderFieldsFlat(width int) string {
	var b strings.Builder
	flat := flattenMappingFields(m.mappings.Fields)

	nameWidth := 30
	typeWidth := 12

	for _, f := range flat {
		name := f.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		attrs := formatFieldAttrs(f.Properties)
		attrStyle := lipgloss.NewStyle().Foreground(ColorGray)

		line := lipgloss.NewStyle().Width(nameWidth).Render(name) + " " +
			lipgloss.NewStyle().Width(typeWidth).Render(f.Type)
		if attrs != "" {
			line += " " + attrStyle.Render(attrs)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m MappingsModel) renderFieldsTree(fields []es.MappingField, indent int, width int) string {
	var b strings.Builder

	for _, f := range fields {
		prefix := strings.Repeat("  ", indent)
		name := f.Name
		if strings.Contains(name, ".") {
			parts := strings.Split(name, ".")
			name = parts[len(parts)-1]
		}

		marker := "  "
		if len(f.Children) > 0 {
			marker = "▼ "
		}

		attrs := formatFieldAttrs(f.Properties)
		attrStyle := lipgloss.NewStyle().Foreground(ColorGray)

		line := prefix + marker + name + "  " + f.Type
		if attrs != "" {
			line += "  " + attrStyle.Render(attrs)
		}
		b.WriteString(line + "\n")

		if len(f.Children) > 0 {
			b.WriteString(m.renderFieldsTree(f.Children, indent+1, width))
		}
	}

	return b.String()
}

func (m MappingsModel) renderAnalyzers(width int) string {
	var b strings.Builder

	for _, a := range m.analyzers {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(a.Name))
		b.WriteString("  ")
		b.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render(a.Kind))
		b.WriteString("\n")

		for k, v := range a.Settings {
			if k == "type" {
				continue
			}
			b.WriteString("  " + k + ": " + v + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func flattenMappingFields(fields []es.MappingField) []es.MappingField {
	var result []es.MappingField
	for _, f := range fields {
		result = append(result, f)
		if len(f.Children) > 0 {
			result = append(result, flattenMappingFields(f.Children)...)
		}
	}
	return result
}

func formatFieldAttrs(props map[string]string) string {
	var attrs []string
	for k, v := range props {
		attrs = append(attrs, k+"="+v)
	}
	if len(attrs) == 0 {
		return ""
	}
	return strings.Join(attrs, " ")
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/ui/mappings.go
git commit -m "feat(ui): add MappingsModel skeleton"
```

---

### Task 6: Wire up Mappings tab in model.go

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Add TabMappings constant and mappings field**

Update constants (after line 18):

```go
const (
	TabOverview = iota
	TabWorkbench
	TabMappings
	TabNodes
	TabTasks
)
```

Add to Model struct (after line 28):

```go
	mappings     MappingsModel
```

Add message types (after line 44):

```go
type mappingsMsg struct {
	mappings  *es.IndexMappings
	analyzers []es.AnalyzerInfo
}
```

**Step 2: Initialize mappings in New()**

Add after `nodes: NewNodes(),`:

```go
	mappings:  NewMappings(),
```

**Step 3: Add fetchMappings command**

Add new method after `fetchTasks()`:

```go
func (m Model) fetchMappings(indexName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		mappings, err := m.client.FetchIndexMappings(ctx, indexName)
		if err != nil {
			return errMsg{err}
		}
		analyzers, _ := m.client.FetchIndexAnalyzers(ctx, indexName)
		return mappingsMsg{mappings: mappings, analyzers: analyzers}
	}
}
```

**Step 4: Handle mappingsMsg in Update()**

Add case in Update() switch after `case tasksMsg:`:

```go
	case mappingsMsg:
		m.loading = false
		m.mappings.SetMappings(msg.mappings, msg.analyzers)
	case fetchMappingsMsg:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchMappings(msg.indexName))
```

**Step 5: Update tab switching logic**

Update the `case "tab":` section to include Mappings:

```go
		case "tab":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.activeTab = TabWorkbench
				m.workbench.Blur()
				return m, nil
			}
			if m.activeTab == TabWorkbench && !m.workbench.HasActiveInput() {
				m.activeTab = TabMappings
				m.workbench.Blur()
				m.mappings.SetIndices(m.cluster.Indices)
				return m, nil
			}
			if m.activeTab == TabMappings && !m.mappings.filterActive {
				m.activeTab = TabNodes
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
			}
			if m.activeTab == TabNodes {
				m.activeTab = TabTasks
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
			}
			if m.activeTab == TabTasks && m.tasks.confirming == "" {
				m.activeTab = TabOverview
				return m, nil
			}
```

Update `case "shift+tab":` similarly.

**Step 6: Update View() tab rendering**

Add mappingsTab variable and update tabs line:

```go
	mappingsTab := InactiveTabStyle.Render("Mappings")
```

In the switch statement add:

```go
	case TabMappings:
		mappingsTab = ActiveTabStyle.Render("Mappings")
```

Update tabs join:

```go
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, overviewTab, workbenchTab, mappingsTab, nodesTab, tasksTab)
```

**Step 7: Add content rendering and delegation**

In content switch add:

```go
		case TabMappings:
			content = m.mappings.View()
```

In delegation switch add:

```go
		case TabMappings:
			m.mappings, cmd = m.mappings.Update(delegateMsg)
```

**Step 8: Update SetSize call**

Add after other SetSize calls:

```go
		m.mappings.SetSize(msg.Width, msg.Height-4)
```

**Step 9: Update status bar**

Add case for TabMappings:

```go
	case TabMappings:
		statusText = "q: quit  Tab: nodes  Shift+Tab: workbench  /: filter  h/l: switch pane  t: tree view  r: refresh"
```

**Step 10: Update mouse click handling**

Update the click boundaries calculation to include Mappings tab.

**Step 11: Verify build**

```bash
go build ./...
```

**Step 12: Commit**

```bash
git add internal/ui/model.go
git commit -m "feat(ui): wire up Mappings tab in main model"
```

---

### Task 7: Update help.go

**Files:**
- Modify: `internal/ui/help.go`

**Step 1: Add Mappings section**

Add new section after Workbench:

```go
		{
			header: "Mappings",
			keys: [][]string{
				{"/", "Filter indices"},
				{"h/l", "Switch pane"},
				{"t", "Toggle tree view"},
				{"up/down", "Navigate/scroll"},
				{"Enter", "Select index"},
				{"r", "Refresh"},
			},
		},
```

**Step 2: Commit**

```bash
git add internal/ui/help.go
git commit -m "docs(ui): add Mappings tab to help"
```

---

### Task 8: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Add Mappings tab to features**

Add after Workbench Tab section:

```markdown
- **Mappings Tab**: Browse index mappings and analyzers
  - Two-pane layout: index list + mappings view
  - Flat and tree view toggle for field hierarchy
  - Shows non-default field attributes (analyzer, index, doc_values)
  - Custom analyzers section showing tokenizers and filters
```

**Step 2: Add keybindings section**

Add Mappings tab keybindings table.

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add Mappings tab to README"
```

---

### Task 9: Add render flag support for mappings

**Files:**
- Modify: `main.go`

**Step 1: Add mappings case to renderAndExit**

```go
	case "mappings":
		state, err := client.FetchClusterState(ctx)
		if err != nil {
			return fmt.Errorf("fetching cluster state: %w", err)
		}
		mappings := ui.NewMappings()
		mappings.SetSize(width, height)
		mappings.SetIndices(state.Indices)
		fmt.Println(mappings.View())
```

**Step 2: Commit**

```bash
git add main.go
git commit -m "feat(cli): add mappings to render flag"
```

---

### Task 10: Manual testing and polish

**Step 1: Build and test**

```bash
go build .
./stoptail --render mappings --width 120 --height 40 localhost
```

**Step 2: Test interactively**

```bash
./stoptail localhost
```

- Tab to Mappings
- Navigate indices with arrow keys
- Press Enter or `l` to view mappings
- Press `t` to toggle tree view
- Press `/` to filter indices
- Press `h` to go back to index list

**Step 3: Fix any issues found**

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete Mappings tab implementation"
```
