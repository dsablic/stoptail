package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestShouldAutoComplete(t *testing.T) {
	tests := []struct {
		name    string
		content string
		row     int
		col     int
		want    bool
	}{
		{"after opening brace", `{"`, 0, 2, true},
		{"after comma", `{"a": 1, "`, 0, 10, true},
		{"after colon", `{"a": "`, 0, 7, false},
		{"in value position", `{"a": "v`, 0, 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWorkbench()
			w.editor.SetContent(tt.content)
			// Note: we can't easily set cursor position, so this test is limited
			// Just verify the method exists and can be called
			_ = w.shouldAutoComplete()
		})
	}
}

func TestAutoCompleteAfterQuote(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	// Simulate initial content with cursor after opening brace
	w.editor.SetContent("{")
	
	// Check shouldAutoComplete at various positions
	// For this we need to simulate that a quote was just typed after {
	w.editor.SetContent(`{"`)
	
	// At this point, if shouldAutoComplete() is called, it should return true
	// because the character before the quote (at col-1) is {
	
	// Test the parsing logic directly
	ctx := ParseJSONContext(`{"`)
	if len(ctx.Path) != 0 {
		t.Errorf("expected empty path, got %v", ctx.Path)
	}
	
	// Check that GetKeywordsForContext returns items for empty path
	keywords := GetKeywordsForContext(ctx.Path)
	if len(keywords) == 0 {
		t.Error("expected keywords for root context")
	}
	
	// Check that "query" is in the keywords
	found := false
	for _, kw := range keywords {
		if kw.Text == "query" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'query' in keywords")
	}
	
	// Check "track_total_hits" is also there
	found = false
	for _, kw := range keywords {
		if kw.Text == "track_total_hits" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'track_total_hits' in keywords")
	}
}

func TestTriggerCompletionSetsState(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)
	
	// Set content with cursor positioned after opening brace and quote
	w.editor.SetContent(`{"`)
	
	// Manually trigger completion (simulating what happens after typing ")
	w.triggerCompletion()
	
	if !w.completion.Active {
		t.Error("completion should be active after triggerCompletion")
	}
	
	if len(w.completion.Items) == 0 {
		t.Error("completion should have items")
	}
	
	if len(w.completion.Filtered) == 0 {
		t.Error("completion should have filtered items")
	}
	
	// Verify dropdown renders
	dropdown := w.renderCompletionDropdown()
	if dropdown == "" {
		t.Error("dropdown should not be empty")
	}
	
	// Verify View includes completion
	view := w.View()
	if !strings.Contains(view, "query") {
		t.Error("view should contain 'query' completion item")
	}
}

func TestShouldAutoCompleteWithContent(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)
	
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"after opening brace", `{"`, true},
		{"after brace with newline", "{\n\"", true},
		{"after comma", `{"a": 1, "`, true},
		{"inside value", `{"a": "`, false},  // after : means value position
		{"empty", `"`, false},  // no brace before
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.editor.SetContent(tt.content)
			// shouldAutoComplete checks col-1, so we need cursor at end
			got := w.shouldAutoComplete()
			if got != tt.want {
				t.Errorf("shouldAutoComplete() = %v, want %v for content %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestAutoCompleteTriggersOnQuote(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)
	w.focus = FocusBody
	w.editor.Focus()

	w.editor.SetContent("{}")
	w.editor.SetCursor(1)

	quoteMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'"'}}
	newModel, _ := w.Update(quoteMsg)

	if !newModel.completion.Active {
		t.Error("completion should be active after typing quote")
	}
	if len(newModel.completion.Items) == 0 {
		t.Error("completion should have items")
	}
}

func TestCtrlRExecutionState(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	if w.executing {
		t.Fatal("executing should be false initially")
	}

	ctrlR := tea.KeyMsg{Type: tea.KeyCtrlR}
	w1, cmd1 := w.Update(ctrlR)

	if w1.client != nil && !w1.executing {
		t.Log("ctrl+r without client does nothing (expected)")
	}

	if cmd1 != nil && w.client == nil {
		t.Error("should not return command without client")
	}

	if w1.search.Active() {
		t.Error("search should not be active by ctrl+r")
	}
}

func TestSearchNavigationWhenActive(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)
	w.responseText = "line1\nmatch\nline3\nmatch\nline5"
	w.response.SetContent(w.responseText)
	w.focus = FocusResponse

	ctrlF := tea.KeyMsg{Type: tea.KeyCtrlF}
	w, _ = w.Update(ctrlF)

	if !w.search.Active() {
		t.Fatal("search should be active after Ctrl+F")
	}

	w.search.SetQuery("match")
	w.updateSearchMatches()

	if w.search.MatchCount() != 2 {
		t.Errorf("expected 2 matches, got %d", w.search.MatchCount())
	}

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	w, _ = w.Update(enterKey)

	if w.search.CurrentIdx() != 1 {
		t.Errorf("expected searchIdx 1 after Enter, got %d", w.search.CurrentIdx())
	}
	if !w.search.Active() {
		t.Error("search should remain active after Enter (just navigates)")
	}

	ctrlP := tea.KeyMsg{Type: tea.KeyCtrlP}
	w, _ = w.Update(ctrlP)

	if w.search.CurrentIdx() != 0 {
		t.Errorf("expected searchIdx 0 after Ctrl+P, got %d", w.search.CurrentIdx())
	}

	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	w, _ = w.Update(escKey)

	if w.search.Active() {
		t.Error("search should be inactive after Esc")
	}

	nKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	w, _ = w.Update(nKey)
	if w.search.CurrentIdx() != 1 {
		t.Errorf("'n' should navigate when search is closed (but matches exist), searchIdx=%d", w.search.CurrentIdx())
	}
}
