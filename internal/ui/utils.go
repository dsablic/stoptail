package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	for strings.HasSuffix(s, " ") || strings.HasSuffix(s, "\x1b[0m") {
		s = strings.TrimSuffix(s, " ")
		s = strings.TrimSuffix(s, "\x1b[0m")
	}
	return s + "\x1b[0m"
}

func HealthColor(health string) lipgloss.Color {
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
		var color lipgloss.Color
		switch {
		case posPercent >= 85:
			color = ColorRed
		case posPercent >= 70:
			color = ColorYellow
		default:
			color = ColorGreen
		}
		style := lipgloss.NewStyle().Foreground(color)
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

func ParseSize(s string) (int64, error) {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(s, "tb") {
		multiplier = tb
		numStr = strings.TrimSuffix(s, "tb")
	} else if strings.HasSuffix(s, "gb") {
		multiplier = gb
		numStr = strings.TrimSuffix(s, "gb")
	} else if strings.HasSuffix(s, "mb") {
		multiplier = mb
		numStr = strings.TrimSuffix(s, "mb")
	} else if strings.HasSuffix(s, "kb") {
		multiplier = kb
		numStr = strings.TrimSuffix(s, "kb")
	} else if strings.HasSuffix(s, "b") {
		multiplier = 1
		numStr = strings.TrimSuffix(s, "b")
	} else {
		numStr = s
	}

	numStr = strings.TrimSpace(numStr)
	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", s)
	}

	return int64(value * float64(multiplier)), nil
}

func FormatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case b >= tb:
		return fmt.Sprintf("%.1ftb", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1fgb", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fmb", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fkb", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%db", b)
	}
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
