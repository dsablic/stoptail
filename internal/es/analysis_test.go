package es

import (
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"", 0},
		{"0", 0},
		{"100b", 100},
		{"1kb", 1024},
		{"1.5kb", 1536},
		{"1mb", BytesMB},
		{"2gb", 2 * BytesGB},
		{"1tb", BytesTB},
		{"1.5gb", int64(1.5 * float64(BytesGB))},
		{"100KB", 100 * BytesKB},
		{"  2gb  ", 2 * BytesGB},
		{"1g", BytesGB},
		{"2m", 2 * BytesMB},
		{"3t", 3 * BytesTB},
		{"invalid", 0},
		{"abc", 0},
		{"gb", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseSize(tt.input)
			if got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0b"},
		{500, "500b"},
		{1023, "1023b"},
		{BytesKB, "1.0kb"},
		{BytesMB, "1.0mb"},
		{BytesGB, "1.0gb"},
		{BytesTB, "1.0tb"},
		{int64(1.5 * float64(BytesGB)), "1.5gb"},
		{2 * BytesTB, "2.0tb"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBytes(tt.input)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUniqueAliases(t *testing.T) {
	tests := []struct {
		name    string
		aliases []AliasInfo
		want    int
	}{
		{"empty", nil, 0},
		{"single", []AliasInfo{{Alias: "a1"}}, 1},
		{"duplicates", []AliasInfo{
			{Alias: "a1", Index: "idx1"},
			{Alias: "a1", Index: "idx2"},
			{Alias: "a2", Index: "idx1"},
		}, 2},
		{"all unique", []AliasInfo{
			{Alias: "a1"},
			{Alias: "a2"},
			{Alias: "a3"},
		}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ClusterState{Aliases: tt.aliases}
			got := s.UniqueAliases()
			if len(got) != tt.want {
				t.Errorf("UniqueAliases() returned %d aliases, want %d", len(got), tt.want)
			}
			seen := make(map[string]bool)
			for _, a := range got {
				if seen[a] {
					t.Errorf("UniqueAliases() returned duplicate alias %q", a)
				}
				seen[a] = true
			}
		})
	}
}

func TestFormatSettingValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"array", []any{"a", "b", "c"}, "[a, b, c]"},
		{"empty array", []any{}, "[]"},
		{"nil", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSettingValue(tt.input)
			if got != tt.want {
				t.Errorf("formatSettingValue(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0s"},
		{500, "0s"},
		{1000, "1s"},
		{30000, "30s"},
		{59999, "59s"},
		{60000, "1m 0s"},
		{90000, "1m 30s"},
		{3599999, "59m 59s"},
		{3600000, "1h 0m"},
		{5400000, "1h 30m"},
		{7200000, "2h 0m"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}
