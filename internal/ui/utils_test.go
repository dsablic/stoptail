package ui

import (
	"strings"
	"testing"
)

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		text, query string
		want        bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"", "", true},
		{"anything", "", true},
		{"", "query", false},
	}

	for _, tt := range tests {
		got := MatchesFilter(tt.text, tt.query)
		if got != tt.want {
			t.Errorf("MatchesFilter(%q, %q) = %v, want %v", tt.text, tt.query, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello", 3, "hel"},
		{"hello", 2, "he"},
		{"hello", 0, ""},
		{"", 5, ""},
		{"unicode test", 6, "uni..."},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestTrimANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"trailing spaces", "hello   "},
		{"trailing reset", "hello\x1b[0m"},
		{"trailing short reset", "hello\x1b[m"},
		{"mixed", "hello \x1b[0m  \x1b[m"},
		{"clean", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimANSI(tt.input)
			if !strings.HasSuffix(got, "\x1b[0m") {
				t.Errorf("TrimANSI(%q) should end with reset code, got %q", tt.input, got)
			}
			trimmed := strings.TrimSuffix(got, "\x1b[0m")
			if strings.HasSuffix(trimmed, " ") {
				t.Errorf("TrimANSI(%q) should not have trailing spaces before reset, got %q", tt.input, got)
			}
		})
	}
}

func TestHealthColor(t *testing.T) {
	green := HealthColor("green")
	yellow := HealthColor("yellow")
	red := HealthColor("red")
	unknown := HealthColor("unknown")
	gray := ColorGray

	if green == yellow {
		t.Error("green and yellow should return different colors")
	}
	if green == red {
		t.Error("green and red should return different colors")
	}
	if yellow == red {
		t.Error("yellow and red should return different colors")
	}
	if unknown != gray {
		t.Errorf("unknown health should return ColorGray, got %v", unknown)
	}
	if green != ColorGreen {
		t.Errorf("HealthColor(green) = %v, want %v", green, ColorGreen)
	}
	if yellow != ColorYellow {
		t.Errorf("HealthColor(yellow) = %v, want %v", yellow, ColorYellow)
	}
	if red != ColorRed {
		t.Errorf("HealthColor(red) = %v, want %v", red, ColorRed)
	}
}

func TestSanitizeForTerminal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal text", "hello world", "hello world"},
		{"preserves newlines", "hello\nworld", "hello\nworld"},
		{"preserves tabs", "hello\tworld", "hello\tworld"},
		{"removes null", "hello\x00world", "helloworld"},
		{"removes BEL", "hello\x07world", "helloworld"},
		{"removes DEL", "hello\x7Fworld", "helloworld"},
		{"removes zero-width space", "hello\u200Bworld", "helloworld"},
		{"removes bidi marks", "hello\u202Aworld", "helloworld"},
		{"removes BOM", "hello\uFEFFworld", "helloworld"},
		{"removes C1 control chars", "hello\u0085world", "helloworld"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForTerminal(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeForTerminal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandleFilterKey(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		key        string
		wantText   string
		wantAction FilterAction
	}{
		{"enter", "query", "enter", "query", FilterConfirm},
		{"esc", "query", "esc", "query", FilterClose},
		{"backspace", "hello", "backspace", "hell", FilterNone},
		{"backspace empty", "", "backspace", "", FilterNone},
		{"type char", "hel", "l", "hell", FilterNone},
		{"type char on empty", "", "a", "a", FilterNone},
		{"multi-char key ignored", "hello", "ctrl+a", "hello", FilterNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotAction := HandleFilterKey(tt.text, tt.key)
			if gotText != tt.wantText {
				t.Errorf("HandleFilterKey(%q, %q) text = %q, want %q", tt.text, tt.key, gotText, tt.wantText)
			}
			if gotAction != tt.wantAction {
				t.Errorf("HandleFilterKey(%q, %q) action = %d, want %d", tt.text, tt.key, gotAction, tt.wantAction)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"-", "-"},
		{"0", "0"},
		{"999", "999"},
		{"1000", "1,000"},
		{"1234567", "1,234,567"},
		{"abc", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatNumber(tt.input)
			if got != tt.want {
				t.Errorf("FormatNumber(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAutoColumnWidths(t *testing.T) {
	headers := []string{"name", "value"}
	rows := [][]string{
		{"short", "a"},
		{"very long name here", "b"},
	}

	widths := AutoColumnWidths(headers, rows, 100)

	if len(widths) != 2 {
		t.Fatalf("expected 2 widths, got %d", len(widths))
	}
	if widths[0] < len("very long name here") {
		t.Errorf("first column width %d should be >= %d", widths[0], len("very long name here"))
	}
}

func TestAutoColumnWidthsShrinks(t *testing.T) {
	headers := []string{"col1", "col2"}
	rows := [][]string{
		{strings.Repeat("x", 100), strings.Repeat("y", 100)},
	}

	widths := AutoColumnWidths(headers, rows, 50)
	total := 0
	for _, w := range widths {
		total += w
	}
	if total > 50 {
		t.Errorf("total column widths %d should fit within 50", total)
	}
}

func TestFitColumns(t *testing.T) {
	rows := [][]string{
		{"hello world this is long", "short"},
	}
	widths := []int{5, 10}

	fitted := FitColumns(rows, widths)
	if len(fitted) != 1 {
		t.Fatalf("expected 1 row, got %d", len(fitted))
	}
	if fitted[0][1] != "short" {
		t.Errorf("short cell should not be modified, got %q", fitted[0][1])
	}
}

func TestJoinPanesHorizontal(t *testing.T) {
	left := "a\nb\nc"
	right := "1\n2"

	result := JoinPanesHorizontal(0, left, right)
	lines := strings.Split(result, "\n")

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	result2 := JoinPanesHorizontal(2, left, right)
	lines2 := strings.Split(result2, "\n")
	if len(lines2) != 2 {
		t.Errorf("expected 2 lines with maxLines=2, got %d", len(lines2))
	}
}

func TestJoinPanesHorizontalThreePanes(t *testing.T) {
	a := "a1\na2"
	b := "b1\nb2"
	c := "c1\nc2"

	result := JoinPanesHorizontal(0, a, b, c)
	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestRenderBar(t *testing.T) {
	bar := RenderBar(50, 10)
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Errorf("RenderBar should be wrapped in brackets, got %q", bar)
	}

	bar0 := RenderBar(0, 10)
	if !strings.Contains(bar0, "[") {
		t.Error("RenderBar(0, 10) should still produce a bar")
	}

	bar100 := RenderBar(100, 5)
	if !strings.Contains(bar100, "[") {
		t.Error("RenderBar(100, 5) should still produce a bar")
	}

	barNeg := RenderBar(-10, 5)
	if !strings.Contains(barNeg, "[") {
		t.Error("RenderBar(-10, 5) should clamp to 0")
	}

	barOver := RenderBar(150, 5)
	if !strings.Contains(barOver, "[") {
		t.Error("RenderBar(150, 5) should clamp to 100")
	}
}

func TestOverlayModal(t *testing.T) {
	bg := strings.Repeat("x", 20) + "\n" + strings.Repeat("x", 20)
	modal := "Modal"

	result := OverlayModal(bg, modal, 20, 5)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}
