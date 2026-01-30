package ui

import "testing"

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100gb", 100 * 1024 * 1024 * 1024},
		{"100GB", 100 * 1024 * 1024 * 1024},
		{"1.5tb", int64(1.5 * 1024 * 1024 * 1024 * 1024)},
		{"500mb", 500 * 1024 * 1024},
		{"100g", 100 * 1024 * 1024 * 1024},
		{"1t", 1024 * 1024 * 1024 * 1024},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := parseSize(tt.input)
		if got != tt.expected {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestShardCalcCalculate(t *testing.T) {
	tests := []struct {
		name       string
		size       string
		docs       string
		nodes      string
		wantShards int
		wantErr    bool
	}{
		{
			name:       "100gb index",
			size:       "100gb",
			docs:       "50000000",
			wantShards: 3,
		},
		{
			name:       "1tb index",
			size:       "1tb",
			docs:       "500000000",
			wantShards: 34,
		},
		{
			name:       "small index",
			size:       "10gb",
			docs:       "1000000",
			wantShards: 1,
		},
		{
			name:       "with nodes",
			size:       "100gb",
			docs:       "50000000",
			nodes:      "5",
			wantShards: 5,
		},
		{
			name:    "invalid size",
			size:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewShardCalc()
			m.sizeInput.SetValue(tt.size)
			m.docsInput.SetValue(tt.docs)
			m.nodesInput.SetValue(tt.nodes)

			result := m.calculate()

			if tt.wantErr {
				if result.err == "" && result.totalSize > 0 {
					t.Errorf("expected error, got none")
				}
				return
			}

			if result.err != "" {
				t.Errorf("unexpected error: %s", result.err)
				return
			}

			if result.primaryShards != tt.wantShards {
				t.Errorf("primaryShards = %d, want %d", result.primaryShards, tt.wantShards)
			}
		})
	}
}
