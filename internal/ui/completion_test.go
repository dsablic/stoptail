package ui

import (
	"testing"
)

func TestCompletionStateFilter(t *testing.T) {
	items := []CompletionItem{
		{Text: "match", Kind: "query"},
		{Text: "match_all", Kind: "query"},
		{Text: "match_phrase", Kind: "query"},
		{Text: "term", Kind: "query"},
		{Text: "terms", Kind: "query"},
	}

	t.Run("filter by prefix", func(t *testing.T) {
		c := &CompletionState{Active: true, Items: items}
		c.Filter("mat")
		if len(c.Filtered) != 3 {
			t.Errorf("expected 3 matches, got %d", len(c.Filtered))
		}
		if c.SelectedIdx != 0 {
			t.Errorf("SelectedIdx should be reset to 0, got %d", c.SelectedIdx)
		}
	})

	t.Run("filter case insensitive", func(t *testing.T) {
		c := &CompletionState{Active: true, Items: items}
		c.Filter("MAT")
		if len(c.Filtered) != 3 {
			t.Errorf("expected 3 matches, got %d", len(c.Filtered))
		}
	})

	t.Run("no matches deactivates", func(t *testing.T) {
		c := &CompletionState{Active: true, Items: items}
		c.Filter("xyz")
		if c.Active {
			t.Error("expected Active=false when no matches")
		}
		if len(c.Filtered) != 0 {
			t.Errorf("expected 0 filtered, got %d", len(c.Filtered))
		}
	})

	t.Run("empty query matches all", func(t *testing.T) {
		c := &CompletionState{Active: true, Items: items}
		c.Filter("")
		if len(c.Filtered) != len(items) {
			t.Errorf("expected %d matches, got %d", len(items), len(c.Filtered))
		}
	})

	t.Run("filter resets selection", func(t *testing.T) {
		c := &CompletionState{Active: true, Items: items, SelectedIdx: 3}
		c.Filter("term")
		if c.SelectedIdx != 0 {
			t.Errorf("SelectedIdx should be 0 after filter, got %d", c.SelectedIdx)
		}
	})
}

func TestCompletionStateMoveUpDown(t *testing.T) {
	items := []CompletionItem{
		{Text: "a", Kind: "test"},
		{Text: "b", Kind: "test"},
		{Text: "c", Kind: "test"},
	}

	c := &CompletionState{Active: true, Items: items, Filtered: items}

	c.MoveDown()
	if c.SelectedIdx != 1 {
		t.Errorf("after MoveDown, SelectedIdx = %d, want 1", c.SelectedIdx)
	}

	c.MoveDown()
	if c.SelectedIdx != 2 {
		t.Errorf("after second MoveDown, SelectedIdx = %d, want 2", c.SelectedIdx)
	}

	c.MoveDown()
	if c.SelectedIdx != 2 {
		t.Errorf("MoveDown at end should clamp, SelectedIdx = %d, want 2", c.SelectedIdx)
	}

	c.MoveUp()
	if c.SelectedIdx != 1 {
		t.Errorf("after MoveUp, SelectedIdx = %d, want 1", c.SelectedIdx)
	}

	c.MoveUp()
	c.MoveUp()
	if c.SelectedIdx != 0 {
		t.Errorf("MoveUp at start should clamp, SelectedIdx = %d, want 0", c.SelectedIdx)
	}
}

func TestCompletionStateSelected(t *testing.T) {
	items := []CompletionItem{
		{Text: "first", Kind: "test"},
		{Text: "second", Kind: "test"},
	}

	t.Run("valid selection", func(t *testing.T) {
		c := &CompletionState{Filtered: items, SelectedIdx: 1}
		got := c.Selected()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Text != "second" {
			t.Errorf("Selected().Text = %q, want %q", got.Text, "second")
		}
	})

	t.Run("empty filtered", func(t *testing.T) {
		c := &CompletionState{Filtered: nil, SelectedIdx: 0}
		if c.Selected() != nil {
			t.Error("expected nil for empty filtered")
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		c := &CompletionState{Filtered: items, SelectedIdx: 5}
		if c.Selected() != nil {
			t.Error("expected nil for out of bounds index")
		}
	})
}

func TestCompletionStateClose(t *testing.T) {
	c := &CompletionState{
		Active:      true,
		Items:       []CompletionItem{{Text: "a"}},
		Filtered:    []CompletionItem{{Text: "a"}},
		SelectedIdx: 2,
		Query:       "test",
	}

	c.Close()

	if c.Active {
		t.Error("Active should be false")
	}
	if c.Items != nil {
		t.Error("Items should be nil")
	}
	if c.Filtered != nil {
		t.Error("Filtered should be nil")
	}
	if c.SelectedIdx != 0 {
		t.Errorf("SelectedIdx should be 0, got %d", c.SelectedIdx)
	}
	if c.Query != "" {
		t.Errorf("Query should be empty, got %q", c.Query)
	}
}

func TestParseJSONContext(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantKey bool
		wantVal bool
		path    []string
	}{
		{
			name:    "empty object",
			input:   `{`,
			wantKey: true,
			wantVal: false,
			path:    nil,
		},
		{
			name:    "after key colon",
			input:   `{"query":`,
			wantKey: false,
			wantVal: true,
			path:    []string{"query"},
		},
		{
			name:    "nested key",
			input:   `{"query":{"bool":{"must":[{"`,
			wantKey: true,
			wantVal: false,
			path:    []string{"query", "bool", "must"},
		},
		{
			name:    "typing key",
			input:   `{"query":{"mat`,
			wantKey: true,
			wantVal: false,
			path:    []string{"query"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ParseJSONContext(tt.input)
			if ctx.InKey != tt.wantKey {
				t.Errorf("InKey = %v, want %v", ctx.InKey, tt.wantKey)
			}
			if ctx.InValue != tt.wantVal {
				t.Errorf("InValue = %v, want %v", ctx.InValue, tt.wantVal)
			}
			if len(ctx.Path) != len(tt.path) {
				t.Errorf("Path length = %d, want %d (path=%v)", len(ctx.Path), len(tt.path), ctx.Path)
			}
			for i, p := range tt.path {
				if i < len(ctx.Path) && ctx.Path[i] != p {
					t.Errorf("Path[%d] = %q, want %q", i, ctx.Path[i], p)
				}
			}
		})
	}
}

func TestGetKeywordsForContext(t *testing.T) {
	t.Run("root level", func(t *testing.T) {
		keywords := GetKeywordsForContext(nil)
		if len(keywords) == 0 {
			t.Error("expected root-level keywords")
		}
		found := false
		for _, k := range keywords {
			if k.Text == "query" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'query' in root-level keywords")
		}
	})

	t.Run("query context", func(t *testing.T) {
		keywords := GetKeywordsForContext([]string{"query"})
		if len(keywords) == 0 {
			t.Error("expected query keywords")
		}
		found := false
		for _, k := range keywords {
			if k.Text == "bool" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'bool' in query keywords")
		}
	})

	t.Run("unknown falls back to parent", func(t *testing.T) {
		keywords := GetKeywordsForContext([]string{"query", "unknown_field"})
		if len(keywords) == 0 {
			t.Error("expected fallback keywords")
		}
	})
}
