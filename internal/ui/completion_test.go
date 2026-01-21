package ui

import "testing"

func TestParseJSONContext(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantCtx JSONContext
	}{
		{
			name:    "empty",
			text:    `{`,
			wantCtx: JSONContext{Path: nil, InKey: true, InValue: false},
		},
		{
			name:    "after opening brace and quote",
			text:    `{"`,
			wantCtx: JSONContext{Path: nil, InKey: true, InValue: false},
		},
		{
			name:    "after key and colon",
			text:    `{"query":`,
			wantCtx: JSONContext{Path: []string{"query"}, InKey: false, InValue: true},
		},
		{
			name:    "nested in query",
			text:    `{"query":{"`,
			wantCtx: JSONContext{Path: []string{"query"}, InKey: true, InValue: false},
		},
		{
			name:    "nested in bool",
			text:    `{"query":{"bool":{"`,
			wantCtx: JSONContext{Path: []string{"query", "bool"}, InKey: true, InValue: false},
		},
		{
			name:    "inside array",
			text:    `{"query":{"bool":{"must":[{"`,
			wantCtx: JSONContext{Path: []string{"query", "bool", "must"}, InKey: true, InValue: false},
		},
		{
			name:    "after field in match",
			text:    `{"query":{"match":{"title":`,
			wantCtx: JSONContext{Path: []string{"query", "match", "title"}, InKey: false, InValue: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseJSONContext(tt.text)
			if got.InKey != tt.wantCtx.InKey {
				t.Errorf("InKey = %v, want %v", got.InKey, tt.wantCtx.InKey)
			}
			if got.InValue != tt.wantCtx.InValue {
				t.Errorf("InValue = %v, want %v", got.InValue, tt.wantCtx.InValue)
			}
			if len(got.Path) != len(tt.wantCtx.Path) {
				t.Errorf("Path = %v, want %v", got.Path, tt.wantCtx.Path)
			} else {
				for i := range got.Path {
					if got.Path[i] != tt.wantCtx.Path[i] {
						t.Errorf("Path[%d] = %v, want %v", i, got.Path[i], tt.wantCtx.Path[i])
					}
				}
			}
		})
	}
}
