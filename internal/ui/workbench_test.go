package ui

import (
	"strings"
	"testing"
)

func TestOffsetToLineCol(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		offset   int
		wantLine int
		wantCol  int
	}{
		{"start of text", "hello", 0, 1, 1},
		{"middle of line", "hello", 3, 1, 4},
		{"end of line", "hello", 5, 1, 6},
		{"after newline", "hello\nworld", 6, 2, 1},
		{"second line middle", "hello\nworld", 8, 2, 3},
		{"empty string", "", 0, 1, 1},
		{"multiple newlines", "a\nb\nc", 4, 3, 1},
		{"offset beyond text", "hi", 10, 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLine, gotCol := offsetToLineCol(tt.text, tt.offset)
			if gotLine != tt.wantLine || gotCol != tt.wantCol {
				t.Errorf("offsetToLineCol(%q, %d) = (%d, %d), want (%d, %d)",
					tt.text, tt.offset, gotLine, gotCol, tt.wantLine, tt.wantCol)
			}
		})
	}
}

func TestOverlayErrorMarker(t *testing.T) {
	m := WorkbenchModel{}

	tests := []struct {
		name      string
		bodyView  string
		errorLine int
		wantSame  bool
	}{
		{"zero error line returns unchanged", "line1\nline2", 0, true},
		{"negative error line returns unchanged", "line1\nline2", -1, true},
		{"error line beyond view returns unchanged", "line1\nline2", 10, true},
		{"valid error line modifies view", "┃ line1\n┃ line2", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.overlayErrorMarker(tt.bodyView, tt.errorLine)
			if tt.wantSame && result != tt.bodyView {
				t.Errorf("expected unchanged view, got modified")
			}
			if !tt.wantSame && result == tt.bodyView {
				t.Errorf("expected modified view, got unchanged")
			}
		})
	}
}

func TestWorkbenchEditorIntegration(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	view := w.View()

	if !strings.Contains(view, "1") {
		t.Error("expected line numbers in view")
	}

	if !strings.Contains(view, "\x1b[") {
		t.Error("expected ANSI color codes (syntax highlighting) in view")
	}

	if !strings.Contains(view, "Body") {
		t.Error("expected Body header in view")
	}
}
