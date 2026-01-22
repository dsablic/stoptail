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
