# Enhanced Query Editor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a VS Code-like query editor with line numbers, syntax highlighting, ES validation, context-aware completion, and mouse selection.

**Architecture:** Tree-sitter parses JSON for AST-based highlighting and completion context. Validation pipeline: instant JSON syntax check + debounced ES _validate API. Custom editor component wraps textarea with gutter and selection overlay.

**Tech Stack:** go-tree-sitter, tree-sitter-json grammar, bubbles textarea, lipgloss styling

---

## Task 1: Add Tree-sitter Dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add tree-sitter dependency**

Run:
```bash
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/json
```

**Step 2: Verify dependency installed**

Run: `go mod tidy && go build .`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add tree-sitter dependency"
```

---

## Task 2: Create Editor Model with Line Numbers

**Files:**
- Create: `internal/ui/editor.go`
- Create: `internal/ui/editor_test.go`

**Step 1: Write test for line number rendering**

```go
// internal/ui/editor_test.go
package ui

import "testing"

func TestRenderLineNumbers(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		width     int
		wantLines int
	}{
		{"empty", "", 3, 1},
		{"single line", "{}", 3, 1},
		{"multi line", "{\n  \"a\": 1\n}", 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEditor()
			e.SetContent(tt.content)
			gutter := e.renderGutter(tt.width, 10)
			lines := len(splitLines(gutter))
			if lines != tt.wantLines {
				t.Errorf("got %d lines, want %d", lines, tt.wantLines)
			}
		})
	}
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestRenderLineNumbers -v`
Expected: FAIL with "NewEditor not defined"

**Step 3: Write minimal Editor struct**

```go
// internal/ui/editor.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

type Editor struct {
	textarea   textarea.Model
	width      int
	height     int
	gutterWidth int
}

func NewEditor() Editor {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = 50000
	return Editor{
		textarea:    ta,
		gutterWidth: 4,
	}
}

func (e *Editor) SetContent(content string) {
	e.textarea.SetValue(content)
}

func (e *Editor) Content() string {
	return e.textarea.Value()
}

func (e Editor) renderGutter(width, height int) string {
	content := e.textarea.Value()
	lineCount := 1 + strings.Count(content, "\n")
	if content == "" {
		lineCount = 1
	}

	gutterStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(width).
		Align(lipgloss.Right)

	var lines []string
	for i := 1; i <= lineCount && i <= height; i++ {
		lines = append(lines, gutterStyle.Render(fmt.Sprintf("%d", i)))
	}
	return strings.Join(lines, "\n")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestRenderLineNumbers -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add Editor model with line number gutter"
```

---

## Task 3: Add Tree-sitter JSON Parsing

**Files:**
- Modify: `internal/ui/editor.go`
- Modify: `internal/ui/editor_test.go`

**Step 1: Write test for tree-sitter parsing**

```go
// Add to internal/ui/editor_test.go
func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantOK  bool
	}{
		{"valid simple", "{}", true},
		{"valid nested", `{"query": {"match": {}}}`, true},
		{"invalid", `{"query":}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEditor()
			e.SetContent(tt.content)
			tree := e.parse()
			hasError := tree != nil && tree.RootNode().HasError()
			gotOK := tree != nil && !hasError
			if gotOK != tt.wantOK {
				t.Errorf("parse() ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestParseJSON -v`
Expected: FAIL with "parse method not defined"

**Step 3: Implement tree-sitter parsing**

```go
// Add to internal/ui/editor.go
import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/json"
)

var jsonParser *sitter.Parser

func init() {
	jsonParser = sitter.NewParser()
	jsonParser.SetLanguage(json.GetLanguage())
}

func (e Editor) parse() *sitter.Tree {
	content := e.textarea.Value()
	if content == "" {
		return nil
	}
	return jsonParser.Parse(nil, []byte(content))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestParseJSON -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add tree-sitter JSON parsing"
```

---

## Task 4: Add Syntax Highlighting

**Files:**
- Modify: `internal/ui/editor.go`
- Modify: `internal/ui/editor_test.go`

**Step 1: Write test for syntax highlighting**

```go
// Add to internal/ui/editor_test.go
func TestHighlightContent(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"key": "value"}`)
	highlighted := e.highlightContent()
	// Should contain ANSI color codes
	if !strings.Contains(highlighted, "\x1b[") {
		t.Error("expected ANSI color codes in highlighted output")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestHighlightContent -v`
Expected: FAIL with "highlightContent not defined"

**Step 3: Implement syntax highlighting using tree-sitter AST**

```go
// Add to internal/ui/editor.go
func (e Editor) highlightContent() string {
	content := e.textarea.Value()
	if content == "" {
		return ""
	}

	tree := e.parse()
	if tree == nil {
		return content
	}

	return e.applyHighlighting(content, tree.RootNode())
}

func (e Editor) applyHighlighting(content string, root *sitter.Node) string {
	type highlight struct {
		start, end int
		color      lipgloss.Color
	}
	var highlights []highlight

	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		nodeType := node.Type()
		start := int(node.StartByte())
		end := int(node.EndByte())

		switch nodeType {
		case "string":
			parent := node.Parent()
			if parent != nil && parent.Type() == "pair" {
				firstChild := parent.Child(0)
				if firstChild != nil && firstChild.Equal(node) {
					highlights = append(highlights, highlight{start, end, ColorBlue})
				} else {
					highlights = append(highlights, highlight{start, end, ColorGreen})
				}
			} else {
				highlights = append(highlights, highlight{start, end, ColorGreen})
			}
		case "number":
			highlights = append(highlights, highlight{start, end, ColorYellow})
		case "true", "false":
			highlights = append(highlights, highlight{start, end, lipgloss.Color("#c586c0")})
		case "null":
			highlights = append(highlights, highlight{start, end, lipgloss.Color("#c586c0")})
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(root)

	if len(highlights) == 0 {
		return content
	}

	var result strings.Builder
	lastEnd := 0
	for _, h := range highlights {
		if h.start > lastEnd {
			result.WriteString(content[lastEnd:h.start])
		}
		style := lipgloss.NewStyle().Foreground(h.color)
		result.WriteString(style.Render(content[h.start:h.end]))
		lastEnd = h.end
	}
	if lastEnd < len(content) {
		result.WriteString(content[lastEnd:])
	}
	return result.String()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestHighlightContent -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add tree-sitter based syntax highlighting"
```

---

## Task 5: Add ES _validate API

**Files:**
- Create: `internal/es/validate.go`
- Create: `internal/es/validate_test.go`

**Step 1: Write test for validate API**

```go
// internal/es/validate_test.go
package es

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		wantValid  bool
		wantErr    string
	}{
		{
			name:       "valid query",
			response:   `{"valid": true}`,
			statusCode: 200,
			wantValid:  true,
		},
		{
			name:       "invalid query",
			response:   `{"valid": false, "error": "unknown field [matchh]"}`,
			statusCode: 200,
			wantValid:  false,
			wantErr:    "unknown field [matchh]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, _ := NewClient(server.URL)
			result, err := client.ValidateQuery("test-index", json.RawMessage(`{}`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tt.wantValid {
				t.Errorf("got valid=%v, want %v", result.Valid, tt.wantValid)
			}
			if result.Error != tt.wantErr {
				t.Errorf("got error=%q, want %q", result.Error, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/es -run TestValidateQuery -v`
Expected: FAIL with "ValidateQuery not defined"

**Step 3: Implement validate API**

```go
// internal/es/validate.go
package es

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type ValidateResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func (c *Client) ValidateQuery(index string, query json.RawMessage) (*ValidateResult, error) {
	body := map[string]json.RawMessage{"query": query}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling query: %w", err)
	}

	path := fmt.Sprintf("/%s/_validate/query", index)
	if index == "" {
		path = "/_validate/query"
	}

	res, err := c.es.Perform(&http.Request{
		Method: "POST",
		URL:    &url.URL{Path: path},
		Body:   io.NopCloser(bytes.NewReader(bodyBytes)),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	if err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result ValidateResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/es -run TestValidateQuery -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/es/validate.go internal/es/validate_test.go
git commit -m "feat(es): add _validate query API"
```

---

## Task 6: Add Debounced Validation to Editor

**Files:**
- Modify: `internal/ui/editor.go`
- Modify: `internal/ui/editor_test.go`

**Step 1: Write test for debounce timer**

```go
// Add to internal/ui/editor_test.go
func TestValidationDebounce(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)

	// Trigger validation
	cmd := e.triggerValidation()
	if cmd == nil {
		t.Error("expected validation command")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestValidationDebounce -v`
Expected: FAIL with "triggerValidation not defined"

**Step 3: Implement debounced validation**

```go
// Add to internal/ui/editor.go
import (
	"time"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/labtiva/stoptail/internal/es"
)

type ValidationState int

const (
	ValidationIdle ValidationState = iota
	ValidationPending
	ValidationValid
	ValidationInvalid
)

type validateMsg struct {
	result *es.ValidateResult
	err    error
}

type validateTickMsg struct{}

func (e *Editor) SetClient(client *es.Client) {
	e.client = client
}

func (e *Editor) SetIndex(index string) {
	e.index = index
}

func (e Editor) triggerValidation() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return validateTickMsg{}
	})
}

func (e Editor) executeValidation() tea.Cmd {
	if e.client == nil || e.index == "" {
		return nil
	}
	content := e.textarea.Value()
	if content == "" {
		return nil
	}

	var query map[string]interface{}
	if err := json.Unmarshal([]byte(content), &query); err != nil {
		return nil
	}

	queryPart, ok := query["query"]
	if !ok {
		return nil
	}

	queryBytes, _ := json.Marshal(queryPart)
	return func() tea.Msg {
		result, err := e.client.ValidateQuery(e.index, queryBytes)
		return validateMsg{result: result, err: err}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestValidationDebounce -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add debounced ES validation"
```

---

## Task 7: Add AST Path Detection for Completion

**Files:**
- Modify: `internal/ui/editor.go`
- Modify: `internal/ui/editor_test.go`

**Step 1: Write test for AST path detection**

```go
// Add to internal/ui/editor_test.go
func TestGetASTPath(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		cursor   int
		wantPath []string
	}{
		{"root", `{""}`, 2, []string{}},
		{"in query", `{"query": {"}}`, 12, []string{"query"}},
		{"in bool", `{"query": {"bool": {""}}}`, 21, []string{"query", "bool"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEditor()
			e.SetContent(tt.content)
			path := e.getASTPath(tt.cursor)
			if len(path) != len(tt.wantPath) {
				t.Errorf("got path %v, want %v", path, tt.wantPath)
				return
			}
			for i, p := range path {
				if p != tt.wantPath[i] {
					t.Errorf("got path %v, want %v", path, tt.wantPath)
					return
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestGetASTPath -v`
Expected: FAIL with "getASTPath not defined"

**Step 3: Implement AST path detection**

```go
// Add to internal/ui/editor.go
func (e Editor) getASTPath(cursorOffset int) []string {
	tree := e.parse()
	if tree == nil {
		return nil
	}

	var path []string
	node := tree.RootNode().NamedDescendantForPointRange(
		sitter.Point{Row: 0, Column: 0},
		sitter.Point{Row: uint32(cursorOffset), Column: uint32(cursorOffset)},
	)

	content := []byte(e.textarea.Value())
	for node != nil {
		if node.Type() == "pair" {
			keyNode := node.ChildByFieldName("key")
			if keyNode != nil {
				key := string(content[keyNode.StartByte()+1 : keyNode.EndByte()-1])
				path = append([]string{key}, path...)
			}
		}
		node = node.Parent()
	}

	if len(path) > 0 {
		path = path[:len(path)-1]
	}
	return path
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestGetASTPath -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add tree-sitter AST path detection"
```

---

## Task 8: Create DSL Schema for Context-Aware Completion

**Files:**
- Create: `internal/ui/dsl_schema.go`
- Create: `internal/ui/dsl_schema_test.go`

**Step 1: Write test for schema lookup**

```go
// internal/ui/dsl_schema_test.go
package ui

import "testing"

func TestGetCompletionsForPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		wantKeys []string
	}{
		{"root", []string{}, []string{"query", "aggs", "size", "from", "sort", "_source", "highlight"}},
		{"query", []string{"query"}, []string{"bool", "match", "match_phrase", "term", "terms", "range", "exists"}},
		{"bool", []string{"query", "bool"}, []string{"must", "should", "must_not", "filter"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := GetCompletionsForPath(tt.path)
			for _, want := range tt.wantKeys {
				found := false
				for _, item := range items {
					if item.Text == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected key %q in completions", want)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestGetCompletionsForPath -v`
Expected: FAIL with "GetCompletionsForPath not defined"

**Step 3: Implement DSL schema**

```go
// internal/ui/dsl_schema.go
package ui

var dslSchema = map[string][]CompletionItem{
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
		{Text: "nested", Kind: "keyword"},
		{Text: "ids", Kind: "keyword"},
	},
	"query.bool": {
		{Text: "must", Kind: "keyword"},
		{Text: "should", Kind: "keyword"},
		{Text: "must_not", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "minimum_should_match", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"query.match.*": {
		{Text: "query", Kind: "keyword"},
		{Text: "operator", Kind: "keyword"},
		{Text: "fuzziness", Kind: "keyword"},
		{Text: "analyzer", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"query.range.*": {
		{Text: "gte", Kind: "keyword"},
		{Text: "gt", Kind: "keyword"},
		{Text: "lte", Kind: "keyword"},
		{Text: "lt", Kind: "keyword"},
		{Text: "format", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"aggs.*": {
		{Text: "terms", Kind: "keyword"},
		{Text: "avg", Kind: "keyword"},
		{Text: "sum", Kind: "keyword"},
		{Text: "min", Kind: "keyword"},
		{Text: "max", Kind: "keyword"},
		{Text: "cardinality", Kind: "keyword"},
		{Text: "date_histogram", Kind: "keyword"},
		{Text: "histogram", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "aggs", Kind: "keyword"},
	},
	"aggs.*.terms": {
		{Text: "field", Kind: "keyword"},
		{Text: "size", Kind: "keyword"},
		{Text: "order", Kind: "keyword"},
		{Text: "missing", Kind: "keyword"},
	},
	"sort.*": {
		{Text: "order", Kind: "keyword"},
		{Text: "mode", Kind: "keyword"},
		{Text: "unmapped_type", Kind: "keyword"},
	},
	"highlight": {
		{Text: "fields", Kind: "keyword"},
		{Text: "pre_tags", Kind: "keyword"},
		{Text: "post_tags", Kind: "keyword"},
	},
}

func GetCompletionsForPath(path []string) []CompletionItem {
	if len(path) == 0 {
		return dslSchema[""]
	}

	key := joinPath(path)
	if items, ok := dslSchema[key]; ok {
		return items
	}

	wildcardKey := joinPathWithWildcard(path)
	if items, ok := dslSchema[wildcardKey]; ok {
		return items
	}

	if len(path) > 1 {
		return GetCompletionsForPath(path[1:])
	}

	return dslSchema[""]
}

func joinPath(path []string) string {
	result := ""
	for i, p := range path {
		if i > 0 {
			result += "."
		}
		result += p
	}
	return result
}

func joinPathWithWildcard(path []string) string {
	if len(path) < 2 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		result += ".*"
	}
	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestGetCompletionsForPath -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/dsl_schema.go internal/ui/dsl_schema_test.go
git commit -m "feat(editor): add ES DSL schema for context-aware completion"
```

---

## Task 9: Add Mouse Selection Support

**Files:**
- Modify: `internal/ui/editor.go`
- Modify: `internal/ui/editor_test.go`

**Step 1: Write test for coordinate to position conversion**

```go
// Add to internal/ui/editor_test.go
func TestScreenToPosition(t *testing.T) {
	e := NewEditor()
	e.SetContent("{\n  \"key\": 1\n}")
	e.SetSize(40, 10)

	tests := []struct {
		name     string
		x, y     int
		wantLine int
		wantCol  int
	}{
		{"first char", 5, 0, 0, 0},
		{"second line", 5, 1, 1, 0},
		{"with offset", 7, 1, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, col := e.screenToPosition(tt.x, tt.y)
			if line != tt.wantLine || col != tt.wantCol {
				t.Errorf("got (%d,%d), want (%d,%d)", line, col, tt.wantLine, tt.wantCol)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestScreenToPosition -v`
Expected: FAIL with "screenToPosition not defined"

**Step 3: Implement mouse position conversion**

```go
// Add to internal/ui/editor.go
type Selection struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	Active              bool
	Dragging            bool
}

func (e *Editor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.textarea.SetWidth(width - e.gutterWidth - 2)
	e.textarea.SetHeight(height)
}

func (e Editor) screenToPosition(x, y int) (line, col int) {
	adjustedX := x - e.gutterWidth - 1
	if adjustedX < 0 {
		adjustedX = 0
	}
	return y, adjustedX
}

func (e *Editor) HandleClick(x, y int) {
	line, col := e.screenToPosition(x, y)
	e.setCursorPosition(line, col)
	e.selection.Active = false
}

func (e *Editor) HandleDragStart(x, y int) {
	line, col := e.screenToPosition(x, y)
	e.selection.StartLine = line
	e.selection.StartCol = col
	e.selection.EndLine = line
	e.selection.EndCol = col
	e.selection.Active = true
	e.selection.Dragging = true
}

func (e *Editor) HandleDrag(x, y int) {
	if !e.selection.Dragging {
		return
	}
	line, col := e.screenToPosition(x, y)
	e.selection.EndLine = line
	e.selection.EndCol = col
}

func (e *Editor) HandleDragEnd() {
	e.selection.Dragging = false
}

func (e *Editor) setCursorPosition(line, col int) {
	lines := strings.Split(e.textarea.Value(), "\n")
	offset := 0
	for i := 0; i < line && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	if line < len(lines) {
		lineLen := len(lines[line])
		if col > lineLen {
			col = lineLen
		}
		offset += col
	}
	e.textarea.SetCursor(offset)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestScreenToPosition -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add mouse click and selection support"
```

---

## Task 10: Add Selection Rendering

**Files:**
- Modify: `internal/ui/editor.go`
- Modify: `internal/ui/editor_test.go`

**Step 1: Write test for selection rendering**

```go
// Add to internal/ui/editor_test.go
func TestRenderSelection(t *testing.T) {
	e := NewEditor()
	e.SetContent("hello world")
	e.selection = Selection{
		StartLine: 0, StartCol: 0,
		EndLine: 0, EndCol: 5,
		Active: true,
	}
	rendered := e.renderWithSelection("hello world")
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("expected ANSI codes for selection highlight")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestRenderSelection -v`
Expected: FAIL with "renderWithSelection not defined"

**Step 3: Implement selection rendering**

```go
// Add to internal/ui/editor.go
func (e Editor) renderWithSelection(content string) string {
	if !e.selection.Active {
		return content
	}

	lines := strings.Split(content, "\n")
	selStyle := lipgloss.NewStyle().Reverse(true)

	startLine, startCol := e.selection.StartLine, e.selection.StartCol
	endLine, endCol := e.selection.EndLine, e.selection.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	var result []string
	for i, line := range lines {
		if i < startLine || i > endLine {
			result = append(result, line)
			continue
		}

		var lineResult string
		runes := []rune(line)

		selStart := 0
		selEnd := len(runes)

		if i == startLine {
			selStart = startCol
		}
		if i == endLine {
			selEnd = endCol
		}

		if selStart > len(runes) {
			selStart = len(runes)
		}
		if selEnd > len(runes) {
			selEnd = len(runes)
		}

		if selStart > 0 {
			lineResult += string(runes[:selStart])
		}
		if selEnd > selStart {
			lineResult += selStyle.Render(string(runes[selStart:selEnd]))
		}
		if selEnd < len(runes) {
			lineResult += string(runes[selEnd:])
		}

		result = append(result, lineResult)
	}

	return strings.Join(result, "\n")
}

func (e Editor) GetSelectedText() string {
	if !e.selection.Active {
		return ""
	}

	content := e.textarea.Value()
	lines := strings.Split(content, "\n")

	startLine, startCol := e.selection.StartLine, e.selection.StartCol
	endLine, endCol := e.selection.EndLine, e.selection.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	if startLine == endLine {
		if startLine >= len(lines) {
			return ""
		}
		runes := []rune(lines[startLine])
		if startCol > len(runes) {
			startCol = len(runes)
		}
		if endCol > len(runes) {
			endCol = len(runes)
		}
		return string(runes[startCol:endCol])
	}

	var result []string
	for i := startLine; i <= endLine && i < len(lines); i++ {
		runes := []rune(lines[i])
		if i == startLine {
			if startCol < len(runes) {
				result = append(result, string(runes[startCol:]))
			}
		} else if i == endLine {
			if endCol > len(runes) {
				endCol = len(runes)
			}
			result = append(result, string(runes[:endCol]))
		} else {
			result = append(result, lines[i])
		}
	}

	return strings.Join(result, "\n")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestRenderSelection -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add selection rendering and text extraction"
```

---

## Task 11: Add Editor View Method

**Files:**
- Modify: `internal/ui/editor.go`

**Step 1: Write test for complete View**

```go
// Add to internal/ui/editor_test.go
func TestEditorView(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)
	e.SetSize(60, 10)
	view := e.View()
	if !strings.Contains(view, "1") {
		t.Error("expected line numbers in view")
	}
	if !strings.Contains(view, "query") {
		t.Error("expected content in view")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestEditorView -v`
Expected: FAIL with "View not defined"

**Step 3: Implement View method**

```go
// Add to internal/ui/editor.go
func (e Editor) View() string {
	content := e.textarea.Value()
	lines := strings.Split(content, "\n")
	lineCount := len(lines)

	gutterWidth := 3
	if lineCount >= 100 {
		gutterWidth = 4
	}

	highlighted := e.highlightContent()
	if e.selection.Active {
		highlighted = e.renderWithSelection(highlighted)
	}

	highlightedLines := strings.Split(highlighted, "\n")

	gutterStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(gutterWidth).
		Align(lipgloss.Right)

	separatorStyle := lipgloss.NewStyle().Foreground(ColorGray)

	var resultLines []string
	visibleLines := e.height
	if visibleLines > lineCount {
		visibleLines = lineCount
	}

	for i := 0; i < visibleLines; i++ {
		lineNum := gutterStyle.Render(fmt.Sprintf("%d", i+1))
		separator := separatorStyle.Render(" │ ")
		lineContent := ""
		if i < len(highlightedLines) {
			lineContent = highlightedLines[i]
		}
		resultLines = append(resultLines, lineNum+separator+lineContent)
	}

	return strings.Join(resultLines, "\n")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestEditorView -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/editor.go internal/ui/editor_test.go
git commit -m "feat(editor): add complete View method with gutter and highlighting"
```

---

## Task 12: Integrate Editor into Workbench

**Files:**
- Modify: `internal/ui/workbench.go`

**Step 1: Replace textarea with Editor in WorkbenchModel**

Modify `internal/ui/workbench.go`:
- Replace `body textarea.Model` with `editor Editor`
- Update `NewWorkbench()` to create Editor
- Update all references from `m.body` to `m.editor`
- Wire up mouse events to editor
- Wire up validation state display

**Step 2: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 3: Manual verification**

Run: `go build . && ./stoptail --render workbench --width 80 --height 30 localhost`
Expected: Editor renders with line numbers and syntax highlighting

**Step 4: Commit**

```bash
git add internal/ui/workbench.go
git commit -m "feat(workbench): integrate new Editor component"
```

---

## Task 13: Wire Up ES Validation Display

**Files:**
- Modify: `internal/ui/workbench.go`

**Step 1: Add validation state to header display**

Update the body header rendering to show:
- `Body ✓` when JSON valid and ES valid
- `Body ✓ ⋯` when JSON valid and ES pending
- `Body ✓ ✗ <error>` when JSON valid but ES invalid
- `Body ✗ line:col` when JSON invalid

**Step 2: Run manual test**

Run: `./stoptail`
- Type valid query, verify ✓ appears
- Type invalid ES query (typo), verify error appears after 500ms

**Step 3: Commit**

```bash
git add internal/ui/workbench.go
git commit -m "feat(workbench): display ES validation status in header"
```

---

## Task 14: Final Integration Test

**Files:**
- Modify: `internal/ui/workbench_test.go`

**Step 1: Add integration test**

```go
// Add to internal/ui/workbench_test.go
func TestWorkbenchEditorIntegration(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	// Verify editor has line numbers
	view := w.View()
	if !strings.Contains(view, "1") {
		t.Error("expected line numbers")
	}

	// Verify syntax highlighting (ANSI codes present)
	if !strings.Contains(view, "\x1b[") {
		t.Error("expected syntax highlighting")
	}
}
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 3: Final commit**

```bash
git add internal/ui/workbench_test.go
git commit -m "test(workbench): add editor integration test"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add tree-sitter dependency | go.mod |
| 2 | Create Editor model with line numbers | editor.go, editor_test.go |
| 3 | Add tree-sitter JSON parsing | editor.go, editor_test.go |
| 4 | Add syntax highlighting | editor.go, editor_test.go |
| 5 | Add ES _validate API | validate.go, validate_test.go |
| 6 | Add debounced validation | editor.go, editor_test.go |
| 7 | Add AST path detection | editor.go, editor_test.go |
| 8 | Create DSL schema | dsl_schema.go, dsl_schema_test.go |
| 9 | Add mouse selection | editor.go, editor_test.go |
| 10 | Add selection rendering | editor.go, editor_test.go |
| 11 | Add Editor View method | editor.go, editor_test.go |
| 12 | Integrate into Workbench | workbench.go |
| 13 | Wire up validation display | workbench.go |
| 14 | Final integration test | workbench_test.go |
