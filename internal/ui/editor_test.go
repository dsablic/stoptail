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
