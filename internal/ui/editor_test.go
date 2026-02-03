package ui

import (
	"strings"
	"testing"
)

func TestValidationDebounce(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)

	cmd := e.triggerValidation()
	if cmd == nil {
		t.Error("expected validation command")
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
	if !strings.Contains(view, "query") {
		t.Error("expected content in view")
	}
}

func TestEditorViewWithSelection(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)
	e.SetSize(60, 10)

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
			e.textarea.SetCursor(len(tt.content))
			got := e.IsKeyCompletionPosition()
			if got != tt.want {
				t.Errorf("IsKeyCompletionPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectAllAndDelete(t *testing.T) {
	e := NewEditor()
	e.SetContent("hello world")
	e.SelectAll()

	if !e.selection.Active {
		t.Error("selection should be active after SelectAll")
	}
	if e.selection.StartLine != 0 || e.selection.StartCol != 0 {
		t.Error("selection should start at 0,0")
	}
	if e.selection.EndLine != 0 || e.selection.EndCol != 11 {
		t.Errorf("selection should end at 0,11, got %d,%d", e.selection.EndLine, e.selection.EndCol)
	}

	e.DeleteSelection()
	if e.Content() != "" {
		t.Errorf("content should be empty after delete, got %q", e.Content())
	}
	if e.selection.Active {
		t.Error("selection should be inactive after delete")
	}
}

func TestDeleteSelectionMultiLine(t *testing.T) {
	e := NewEditor()
	e.SetContent("line1\nline2\nline3")
	e.SetSelection(0, 2, 1, 3)

	e.DeleteSelection()
	if e.Content() != "lie2\nline3" {
		t.Errorf("unexpected content after delete: %q", e.Content())
	}
}
