package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestValidationDebounce(t *testing.T) {
	e := NewEditor()
	e.SetContent(`{"query": {}}`)

	cmd := e.triggerValidation()
	if cmd == nil {
		t.Error("expected validation command")
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
	e.Focus()
	e.SelectAll()

	if !strings.Contains(e.View(), "\x1b[7m") {
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
			e.textarea.SetCursorColumn(len(tt.content))
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

	if got := e.GetSelectedText(); got != "hello world" {
		t.Errorf("selected text = %q, want %q", got, "hello world")
	}

	e.DeleteSelection()
	if e.Content() != "" {
		t.Errorf("content should be empty after delete, got %q", e.Content())
	}
	if e.HasSelection() {
		t.Error("selection should be inactive after delete")
	}
}

func TestDeleteSelectionMultiLine(t *testing.T) {
	e := NewEditor()
	e.SetContent("line1\nline2\nline3")
	e.SetSize(60, 10)
	e.BeginMouseSelection(2, 0)
	e.ExtendMouseSelection(3, 1)
	e.EndMouseSelection()

	e.DeleteSelection()
	if e.Content() != "lie2\nline3" {
		t.Errorf("unexpected content after delete: %q", e.Content())
	}
}

func TestEditorSelectionKeys(t *testing.T) {
	tests := []struct {
		key  tea.KeyPressMsg
		want string
	}{
		{tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}, "hello world\nline two"},
		{tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}, "h"},
		{tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl | tea.ModShift}, "hello"},
		{tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt | tea.ModShift}, "hello"},
		{tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModShift}, "hello world"},
		{tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}, "hello world\n"},
	}
	for _, tt := range tests {
		t.Run(tt.key.String(), func(t *testing.T) {
			e := NewEditor()
			e.SetContent("hello world\nline two")
			e.SetSize(60, 10)
			e.Focus()
			e.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl})
			e.Update(tt.key)
			if got := e.GetSelectedText(); got != tt.want {
				t.Errorf("selected %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditorKeyBindings(t *testing.T) {
	e := NewEditor()
	if e.textarea.KeyMap.CopySelection.Enabled() {
		t.Error("textarea copy should be disabled; the workbench copies via OSC52")
	}

	e.SetContent("hello world")
	e.SetSize(60, 10)
	e.Focus()
	e.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if e.textarea.Column() != 0 {
		t.Errorf("home should move to line start, column = %d", e.textarea.Column())
	}
	if e.HasSelection() {
		t.Error("home should not select")
	}
}

func TestShiftHomeEndOnWrappedLine(t *testing.T) {
	long := strings.Repeat("abcdefghij", 5)
	e := NewEditor()
	e.SetContent(long)
	e.SetSize(20, 10)
	e.Focus()
	e.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl})
	e.Update(tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModShift})
	if got := e.GetSelectedText(); got != long {
		t.Errorf("shift+end selected %q, want the whole logical line", got)
	}
	e.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModShift})
	if e.HasSelection() {
		t.Errorf("shift+home back to anchor should leave no selection, got %q", e.GetSelectedText())
	}
}

func TestUndoRestoresContentReplacedBySelectionEdits(t *testing.T) {
	keys := []tea.KeyPressMsg{
		{Code: tea.KeySpace, Text: " "},
		{Code: 'k', Mod: tea.ModCtrl},
		{Code: 'u', Mod: tea.ModCtrl},
		{Code: 'w', Mod: tea.ModCtrl},
		{Code: 'd', Mod: tea.ModCtrl},
		{Code: tea.KeyBackspace, Mod: tea.ModAlt},
		{Code: 'é', Text: "é"},
	}
	for _, k := range keys {
		t.Run(k.String(), func(t *testing.T) {
			e := NewEditor()
			e.SetContent("hello")
			e.SetSize(60, 10)
			e.Focus()
			e.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
			e.SelectAll()
			e.Update(k)
			if e.Content() == "hellox" {
				t.Fatalf("%s did not edit the selection", k.String())
			}
			e.Undo()
			if e.Content() != "hellox" {
				t.Errorf("undo after %s = %q, want %q", k.String(), e.Content(), "hellox")
			}
		})
	}
}

func TestCursorMovementDoesNotCreateUndoStep(t *testing.T) {
	e := NewEditor()
	e.SetContent("hello")
	e.Focus()
	e.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	e.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if e.Undo() {
		t.Error("cursor movement and selection should not push undo state")
	}
}
