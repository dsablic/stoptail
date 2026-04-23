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
		got := ParseSizeOrZero(tt.input)
		if got != tt.expected {
			t.Errorf("ParseSizeOrZero(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0.0 MB"},
		{1024 * 1024, "1.0 MB"},
		{500 * 1024 * 1024, "500.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{int64(1.5 * 1024 * 1024 * 1024), "1.5 GB"},
		{10 * 1024 * 1024 * 1024, "10.0 GB"},
	}

	for _, tt := range tests {
		got := formatSize(tt.input)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDocs(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{5500000, "5.5M"},
	}

	for _, tt := range tests {
		got := formatDocs(tt.input)
		if got != tt.want {
			t.Errorf("formatDocs(%d) = %q, want %q", tt.input, got, tt.want)
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
