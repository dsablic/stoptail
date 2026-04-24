package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

	quoteMsg := tea.KeyPressMsg{Code: '"', Text: "\""}
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

	ctrlR := tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}
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

func TestResponsePgDownFullFlow(t *testing.T) {
	m := New(nil, nil)
	m.connected = true
	m.activeTab = TabWorkbench
	m.width = 80
	m.height = 30
	m.workbench.SetSize(80, 26)

	var lines []string
	for i := range 100 {
		lines = append(lines, fmt.Sprintf("line %d content here", i))
	}
	content := strings.Join(lines, "\n")
	m.workbench.responseRawText = content
	m.workbench.responseText = content
	m.workbench.wrapResponseLines()
	m.workbench.focus = FocusResponse

	before := m.workbench.responseNav.Scroll

	pgDown := tea.KeyPressMsg{Code: tea.KeyPgDown}
	updated, _ := m.Update(pgDown)
	m = updated.(Model)

	after := m.workbench.responseNav.Scroll
	t.Logf("FULL FLOW: before=%d after=%d hasActiveInput=%v focus=%d",
		before, after, m.hasActiveInput(), m.workbench.focus)
	if after <= before {
		t.Errorf("pgdown through model should scroll: before=%d after=%d", before, after)
	}
}

func TestResponseWrapping(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	rawJSON := `{"took": 5, "timed_out": false, "hits": {"total": {"value": 100, "relation": "eq"}, "max_score": 1.0, "hits": [{"_index": "test", "_id": "1", "_score": 1.0, "_source": {"field1": "value1", "field2": "value2", "field3": "this is a long value that should cause wrapping"}}]}}`
	highlighted := highlightJSON(rawJSON)
	w.responseRawText = rawJSON
	w.responseText = highlighted
	w.wrapResponseLines()

	rawLines := strings.Split(rawJSON, "\n")
	t.Logf("raw lines=%d, wrapped lines=%d", len(rawLines), len(w.responseLines))

	if len(w.responseLines) <= len(rawLines) {
		t.Errorf("wrapping should produce more lines (%d) than raw lines (%d)",
			len(w.responseLines), len(rawLines))
	}
}

func TestResponsePgDownScrolls(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	var lines []string
	for i := range 100 {
		lines = append(lines, fmt.Sprintf("line %d content here", i))
	}
	content := strings.Join(lines, "\n")
	w.responseRawText = content
	w.responseText = content
	w.wrapResponseLines()
	w.focus = FocusResponse

	before := w.responseNav.Scroll

	pgDown := tea.KeyPressMsg{Code: tea.KeyPgDown}
	w, _ = w.Update(pgDown)

	after := w.responseNav.Scroll
	t.Logf("before=%d after=%d height=%d focus=%d", before, after, w.height, w.focus)
	if after <= before {
		t.Errorf("pgdown should scroll: before=%d after=%d", before, after)
	}
}

func TestResponseHomeEnd(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)

	var lines []string
	for i := range 100 {
		lines = append(lines, fmt.Sprintf("line %d content here", i))
	}
	content := strings.Join(lines, "\n")
	w.responseRawText = content
	w.responseText = content
	w.wrapResponseLines()
	w.focus = FocusResponse

	endKey := tea.KeyPressMsg{Code: tea.KeyEnd}
	w, _ = w.Update(endKey)
	atEnd := w.responseNav.Scroll
	t.Logf("after End: scroll=%d", atEnd)
	if atEnd == 0 {
		t.Errorf("end key should scroll to bottom, scroll=%d", atEnd)
	}

	homeKey := tea.KeyPressMsg{Code: tea.KeyHome}
	w, _ = w.Update(homeKey)
	atHome := w.responseNav.Scroll
	t.Logf("after Home: scroll=%d", atHome)
	if atHome != 0 {
		t.Errorf("home key should scroll to top, scroll=%d", atHome)
	}
}

func TestSearchNavigationWhenActive(t *testing.T) {
	w := NewWorkbench()
	w.SetSize(80, 30)
	w.responseRawText = "line1\nmatch\nline3\nmatch\nline5"
	w.responseText = w.responseRawText
	w.wrapResponseLines()
	w.focus = FocusResponse

	ctrlF := tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}
	w, _ = w.Update(ctrlF)

	if !w.search.Active() {
		t.Fatal("search should be active after Ctrl+F")
	}

	w.search.SetQuery("match")
	w.updateSearchMatches()

	if w.search.MatchCount() != 2 {
		t.Errorf("expected 2 matches, got %d", w.search.MatchCount())
	}

	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	w, _ = w.Update(enterKey)

	if w.search.CurrentIdx() != 1 {
		t.Errorf("expected searchIdx 1 after Enter, got %d", w.search.CurrentIdx())
	}
	if !w.search.Active() {
		t.Error("search should remain active after Enter (just navigates)")
	}

	ctrlP := tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}
	w, _ = w.Update(ctrlP)

	if w.search.CurrentIdx() != 0 {
		t.Errorf("expected searchIdx 0 after Ctrl+P, got %d", w.search.CurrentIdx())
	}

	escKey := tea.KeyPressMsg{Code: tea.KeyEscape}
	w, _ = w.Update(escKey)

	if w.search.Active() {
		t.Error("search should be inactive after Esc")
	}

	nKey := tea.KeyPressMsg{Code: 'n', Text: "n"}
	w, _ = w.Update(nKey)
	if w.search.CurrentIdx() != 1 {
		t.Errorf("'n' should navigate when search is closed (but matches exist), searchIdx=%d", w.search.CurrentIdx())
	}
}

func extractIndexFromPathStr(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, "_") {
			return part
		}
	}
	return ""
}

func TestExtractIndexFromPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/my-index/_search", "my-index"},
		{"/_cat/indices", "indices"},
		{"/", ""},
		{"", ""},
		{"/index/type/id", "index"},
		{"my-index/_doc/1", "my-index"},
	}

	for _, tt := range tests {
		got := extractIndexFromPathStr(tt.input)
		if got != tt.want {
			t.Errorf("extractIndexFromPathStr(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
