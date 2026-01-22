package ui

import "testing"

func TestGetCompletionsForPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		wantKeys []string
	}{
		{"root", []string{}, []string{"query", "aggs", "size", "from", "sort", "_source", "highlight"}},
		{"query", []string{"query"}, []string{"bool", "match", "match_phrase", "term", "terms", "range", "exists"}},
		{"bool", []string{"query", "bool"}, []string{"must", "should", "must_not", "filter"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := GetCompletionsForPath(tt.path)
			for _, want := range tt.wantKeys {
				found := false
				for _, item := range items {
					if item.Text == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected key %q in completions", want)
				}
			}
		})
	}
}
