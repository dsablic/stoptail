package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestParseKeys(t *testing.T) {
	tests := []struct {
		keys string
		want []string
	}{
		{"down,enter", []string{"down", "enter"}},
		{"a,/", []string{"a", "/"}},
		{"ctrl+a", []string{"ctrl+a"}},
		{"shift+down", []string{"shift+down"}},
		{"ctrl+shift+right", []string{"ctrl+shift+right"}},
		{"alt+shift+left,shift+end", []string{"alt+shift+left", "shift+end"}},
		{"ctrl+home", []string{"ctrl+home"}},
		{"+", []string{"+"}},
		{"bogus+down", nil},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			msgs := parseKeys(tt.keys)
			var got []string
			for _, m := range msgs {
				got = append(got, m.String())
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseKeys(%q) = %v, want %v", tt.keys, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseKeys(%q)[%d] = %q, want %q", tt.keys, i, got[i], tt.want[i])
				}
			}
		})
	}
	if msgs := parseKeys("a"); msgs[0].Text != "a" {
		t.Errorf("plain key should carry Text, got %+v", msgs[0])
	}
	if msgs := parseKeys("ctrl+a"); msgs[0].Mod != tea.ModCtrl {
		t.Errorf("ctrl+a should carry ModCtrl, got %+v", msgs[0])
	}
}
