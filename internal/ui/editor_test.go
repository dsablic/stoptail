package ui

import (
	"strings"
	"testing"
)

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

func TestHighlightContent(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"key": "value"}`)
	highlighted := e.highlightContent()
	if !strings.Contains(highlighted, "\x1b[") {
		t.Error("expected ANSI color codes in highlighted output")
	}
}

func TestValidationDebounce(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)

	cmd := e.triggerValidation()
	if cmd == nil {
		t.Error("expected validation command")
	}
}

func TestGetASTPath(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		cursor   int
		wantPath []string
	}{
		{"root", `{""}`, 2, []string{}},
		{"in query", `{"query": {""}}`, 12, []string{"query"}},
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
		{"first char", 6, 0, 0, 0},
		{"second line", 6, 1, 1, 0},
		{"with offset", 8, 1, 1, 2},
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

func TestRenderPlainWithCursor(t *testing.T) {
	e := NewEditor()
	content := "hello"

	result := e.renderPlainWithCursor(content, 0, 2)
	if !strings.Contains(result, "\x1b[7m") {
		t.Error("expected reverse video ANSI code for cursor")
	}
	if !strings.Contains(result, "l") {
		t.Error("expected cursor character 'l' in output")
	}
}

func TestEditorViewStateTransitions(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)
	e.SetSize(60, 10)

	unfocusedView := e.View()
	if !strings.Contains(unfocusedView, "\x1b[") {
		t.Error("unfocused view should have syntax highlighting")
	}

	e.Focus()
	focusedView := e.View()
	if !strings.Contains(focusedView, "\x1b[7m") {
		t.Error("focused view should show cursor (reverse video)")
	}

	e.selection = Selection{
		StartLine: 0, StartCol: 0,
		EndLine: 0, EndCol: 5,
		Active: true,
	}
	selectionView := e.View()
	if !strings.Contains(selectionView, "\x1b[7m") {
		t.Error("selection view should show selection (reverse video)")
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

func TestIsKeyCompletionPosition(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"after open brace", "{", true},
		{"after comma in object", `{"a": 1,`, true},
		{"after colon", `{"a":`, false},
		{"after value", `{"a": 1`, false},
		{"after close brace", `{"a": 1}`, false},
		{"after open bracket", "[", false},
		{"inside array", `["a",`, false},
		{"after bracket then brace", "[{", true},
		{"in object inside array", `[{"a": 1,`, true},
		{"after object in array", `[{}`, false},
		{"between objects in array", `[{},`, false},
		{"new object in array", `[{}, {`, true},
		{"after close bracket", `[1, 2]`, false},
		{"nested array", `{"a": [`, false},
		{"nested array with object", `{"a": [{`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEditor()
			e.SetContent(tt.content)
			lines := strings.Split(tt.content, "\n")
			lastLine := len(lines) - 1
			lastCol := len(lines[lastLine])
			e.textarea.SetCursor(len(tt.content))
			e.cursorLine = lastLine
			e.cursorCol = lastCol
			e.cursorSet = true
			got := e.IsKeyCompletionPosition()
			if got != tt.want {
				t.Errorf("IsKeyCompletionPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}
