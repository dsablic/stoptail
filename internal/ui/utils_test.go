package ui

import "testing"

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"-", "-"},
		{"0", "0"},
		{"999", "999"},
		{"1000", "1,000"},
		{"1234", "1,234"},
		{"12345", "12,345"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
		{"12345678", "12,345,678"},
		{"123456789", "123,456,789"},
		{"1234567890", "1,234,567,890"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
