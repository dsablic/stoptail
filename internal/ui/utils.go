package ui

import (
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
