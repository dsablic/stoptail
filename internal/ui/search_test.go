package ui

import "testing"

func TestNewSearchBar(t *testing.T) {
	s := NewSearchBar()
	if s.Active() {
		t.Error("new search bar should not be active")
	}
	if s.Query() != "" {
		t.Error("new search bar should have empty query")
	}
	if len(s.Matches()) != 0 {
		t.Error("new search bar should have no matches")
	}
}

func TestSearchBarActivateDeactivate(t *testing.T) {
	s := NewSearchBar()

	s.Activate()
	if !s.Active() {
		t.Error("search bar should be active after Activate()")
	}

	s.Deactivate()
	if s.Active() {
		t.Error("search bar should not be active after Deactivate()")
	}
}

func TestSearchBarFindMatches(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		query     string
		wantCount int
	}{
		{"empty query", []string{"foo", "bar"}, "", 0},
		{"no matches", []string{"foo", "bar"}, "baz", 0},
		{"single match", []string{"foo", "bar", "baz"}, "bar", 1},
		{"multiple matches", []string{"foo", "foobar", "barfoo"}, "foo", 3},
		{"case insensitive", []string{"FOO", "foo", "Foo"}, "foo", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSearchBar()
			s.Activate()
			s.input.SetValue(tt.query)
			s.FindMatches(tt.lines)

			if len(s.Matches()) != tt.wantCount {
				t.Errorf("got %d matches, want %d", len(s.Matches()), tt.wantCount)
			}
		})
	}
}

func TestSearchBarCurrentMatch(t *testing.T) {
	s := NewSearchBar()

	if s.CurrentMatch() != -1 {
		t.Error("CurrentMatch() should return -1 when no matches")
	}

	s.Activate()
	s.input.SetValue("foo")
	s.FindMatches([]string{"foo", "bar", "foo"})

	if s.CurrentMatch() != 0 {
		t.Errorf("CurrentMatch() = %d, want 0", s.CurrentMatch())
	}
}

func TestSearchBarNextPrevMatch(t *testing.T) {
	s := NewSearchBar()

	if s.NextMatch() != -1 {
		t.Error("NextMatch() should return -1 when no matches")
	}
	if s.PrevMatch() != -1 {
		t.Error("PrevMatch() should return -1 when no matches")
	}

	s.Activate()
	s.input.SetValue("x")
	s.FindMatches([]string{"x", "y", "x", "z", "x"})

	if got := s.NextMatch(); got != 2 {
		t.Errorf("first NextMatch() = %d, want 2", got)
	}
	if got := s.NextMatch(); got != 4 {
		t.Errorf("second NextMatch() = %d, want 4", got)
	}
	if got := s.NextMatch(); got != 0 {
		t.Errorf("third NextMatch() should wrap to 0, got %d", got)
	}

	if got := s.PrevMatch(); got != 4 {
		t.Errorf("PrevMatch() from 0 should wrap to 4, got %d", got)
	}
	if got := s.PrevMatch(); got != 2 {
		t.Errorf("PrevMatch() = %d, want 2", got)
	}
}
