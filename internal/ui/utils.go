package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/labtiva/stoptail/internal/es"
)

type FilterAction int

const (
	FilterNone    FilterAction = iota
	FilterClose                // esc
	FilterConfirm              // enter
)

func HandleFilterKey(text, key string) (string, FilterAction) {
	switch key {
	case "enter":
		return text, FilterConfirm
	case "esc":
		return text, FilterClose
	case "backspace":
		r := []rune(text)
		if len(r) > 0 {
			return string(r[:len(r)-1]), FilterNone
		}
		return text, FilterNone
	default:
		if len(key) == 1 {
			return text + key, FilterNone
		}
		return text, FilterNone
	}
}

func MatchesFilter(text, query string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(query))
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)
	return s
}

func Truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-3]) + "..."
}

func TrimANSI(s string) string {
	for strings.HasSuffix(s, " ") || strings.HasSuffix(s, "\x1b[0m") || strings.HasSuffix(s, "\x1b[m") {
		s = strings.TrimSuffix(s, " ")
		s = strings.TrimSuffix(s, "\x1b[0m")
		s = strings.TrimSuffix(s, "\x1b[m")
	}
	return s + "\x1b[0m"
}

func HealthColor(health string) color.Color {
	switch health {
	case "green":
		return ColorGreen
	case "yellow":
		return ColorYellow
	case "red":
		return ColorRed
	default:
		return ColorGray
	}
}

func RenderBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100 * float64(width))
	empty := width - filled

	var b strings.Builder
	b.WriteString("[")

	for i := 0; i < filled; i++ {
		posPercent := float64(i+1) / float64(width) * 100
		var barColor color.Color
		switch {
		case posPercent >= 85:
			barColor = ColorRed
		case posPercent >= 70:
			barColor = ColorYellow
		default:
			barColor = ColorGreen
		}
		style := lipgloss.NewStyle().Foreground(barColor)
		b.WriteString(style.Render("█"))
	}

	emptyStyle := lipgloss.NewStyle().Foreground(ColorGray)
	b.WriteString(emptyStyle.Render(strings.Repeat("░", empty)))
	b.WriteString("]")

	return b.String()
}

func OverlayModal(background, modal string, width, height int) string {
	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	modalHeight := len(modalLines)
	startY := (height - modalHeight) / 2
	if startY < 0 {
		startY = 0
	}

	dimStyle := lipgloss.NewStyle().Faint(true)
	result := make([]string, len(bgLines))
	for i, line := range bgLines {
		result[i] = dimStyle.Render(line)
	}

	for i, modalLine := range modalLines {
		y := startY + i
		if y >= 0 && y < len(result) {
			lineWidth := lipgloss.Width(modalLine)
			padLeft := (width - lineWidth) / 2
			if padLeft < 0 {
				padLeft = 0
			}
			result[y] = strings.Repeat(" ", padLeft) + modalLine
		}
	}

	if len(result) > height {
		result = result[:height]
	}

	return strings.Join(result, "\n")
}

func FormatNumber(s string) string {
	if s == "" || s == "-" {
		return s
	}

	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}

	if num < 1000 {
		return s
	}

	str := strconv.FormatInt(num, 10)
	var result strings.Builder
	n := len(str)

	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}

	return result.String()
}

func AutoColumnWidths(headers []string, rows [][]string, maxTotal int) []int {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				cellLen := lipgloss.Width(cell)
				if cellLen > widths[i] {
					widths[i] = cellLen
				}
			}
		}
	}

	borderOverhead := len(headers) + 1 + len(headers)*2
	available := maxTotal - borderOverhead
	if available < len(headers)*3 {
		available = len(headers) * 3
	}

	total := 0
	for _, w := range widths {
		total += w
	}

	if total > available {
		excess := total - available
		for excess > 0 {
			maxIdx := -1
			maxWidth := 0
			for i, w := range widths {
				if w > maxWidth {
					maxWidth = w
					maxIdx = i
				}
			}
			if maxIdx < 0 || widths[maxIdx] <= 3 {
				break
			}
			widths[maxIdx]--
			excess--
		}
	}

	return widths
}

func FitColumns(rows [][]string, widths []int) [][]string {
	result := make([][]string, len(rows))
	for i, row := range rows {
		newRow := make([]string, len(row))
		for j, cell := range row {
			if j < len(widths) && widths[j] > 0 {
				cellWidth := lipgloss.Width(cell)
				if cellWidth > widths[j] {
					newRow[j] = lipgloss.NewStyle().MaxWidth(widths[j]).Render(cell)
				} else {
					newRow[j] = cell
				}
			} else {
				newRow[j] = cell
			}
		}
		result[i] = newRow
	}
	return result
}

func FormatBytes(b int64) string {
	return es.FormatBytes(b)
}

func ParseSize(s string) (int64, error) {
	v := es.ParseSize(s)
	if v == 0 && strings.TrimSpace(s) != "" && strings.TrimSpace(s) != "0" {
		return 0, fmt.Errorf("invalid size format: %s", s)
	}
	return v, nil
}

func ParseSizeOrZero(s string) int64 {
	return es.ParseSize(s)
}

func SanitizeForTerminal(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 32:
			continue
		case r == 0x7F:
			continue
		case r >= 0x80 && r <= 0x9F:
			continue
		case r >= 0x200B && r <= 0x200F:
			continue
		case r >= 0x202A && r <= 0x202E:
			continue
		case r >= 0x2060 && r <= 0x206F:
			continue
		case r == 0xFEFF:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func JoinPanesHorizontal(maxLines int, panes ...string) string {
	splitPanes := make([][]string, len(panes))
	computed := 0
	for i, p := range panes {
		splitPanes[i] = strings.Split(p, "\n")
		if len(splitPanes[i]) > computed {
			computed = len(splitPanes[i])
		}
	}
	if maxLines <= 0 {
		maxLines = computed
	}
	var lines []string
	for i := range maxLines {
		var parts []string
		for _, sp := range splitPanes {
			line := ""
			if i < len(sp) {
				line = TrimANSI(sp[i])
			}
			parts = append(parts, line)
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	return strings.Join(lines, "\n")
}

func NewFittedTable(headers []string, rows [][]string, width int, border lipgloss.Border, borderColor color.Color) (*table.Table, [][]string) {
	widths := AutoColumnWidths(headers, rows, width)
	fittedRows := FitColumns(rows, widths)
	t := table.New().
		Border(border).
		BorderStyle(lipgloss.NewStyle().Foreground(borderColor)).
		Headers(headers...).
		Rows(fittedRows...)
	return t, fittedRows
}
