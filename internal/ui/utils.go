package ui

import (
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
