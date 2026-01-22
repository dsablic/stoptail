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
