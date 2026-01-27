package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - adapt based on terminal background
	ColorGreen  lipgloss.Color
	ColorYellow lipgloss.Color
	ColorRed    lipgloss.Color
	ColorBlue   lipgloss.Color
	ColorPurple lipgloss.Color
	ColorGray   lipgloss.Color
	ColorWhite  lipgloss.Color

	HeaderBg      lipgloss.Color
	ActiveBg      lipgloss.Color
	SpinnerClr    lipgloss.Color
	TextBg        lipgloss.Color
	ColorOnAccent lipgloss.Color

	// Base styles
	HeaderStyle      lipgloss.Style
	TabStyle         lipgloss.Style
	ActiveTabStyle   lipgloss.Style
	InactiveTabStyle lipgloss.Style
	StatusBarStyle   lipgloss.Style
	HelpStyle        lipgloss.Style
)

func init() {
	SetTheme("auto")
}

func SetTheme(theme string) {
	isDark := lipgloss.HasDarkBackground()

	switch theme {
	case "dark":
		isDark = true
	case "light":
		isDark = false
	}

	if isDark {
		ColorGreen = lipgloss.Color("#22c55e")
		ColorYellow = lipgloss.Color("#eab308")
		ColorRed = lipgloss.Color("#ef4444")
		ColorBlue = lipgloss.Color("#3b82f6")
		ColorPurple = lipgloss.Color("#a855f7")
		ColorGray = lipgloss.Color("#9ca3af")
		ColorWhite = lipgloss.Color("#f9fafb")
		HeaderBg = lipgloss.Color("#1f2937")
		ActiveBg = lipgloss.Color("#374151")
		SpinnerClr = lipgloss.Color("#3b82f6")
		TextBg = lipgloss.Color("#111827")
		ColorOnAccent = lipgloss.Color("#f9fafb")
	} else {
		ColorGreen = lipgloss.Color("#16a34a")
		ColorYellow = lipgloss.Color("#ca8a04")
		ColorRed = lipgloss.Color("#dc2626")
		ColorBlue = lipgloss.Color("#2563eb")
		ColorPurple = lipgloss.Color("#9333ea")
		ColorGray = lipgloss.Color("#6b7280")
		ColorWhite = lipgloss.Color("#111827")
		HeaderBg = lipgloss.Color("#e5e7eb")
		ActiveBg = lipgloss.Color("#d1d5db")
		SpinnerClr = lipgloss.Color("#2563eb")
		TextBg = lipgloss.Color("#f9fafb")
		ColorOnAccent = lipgloss.Color("#f9fafb")
	}

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorWhite).
		Background(HeaderBg).
		Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
		Padding(0, 2)

	ActiveTabStyle = TabStyle.
		Bold(true).
		Foreground(ColorOnAccent).
		Background(ColorBlue)

	InactiveTabStyle = TabStyle.
		Foreground(ColorGray)

	StatusBarStyle = lipgloss.NewStyle().
		Foreground(ColorGray).
		Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
		Foreground(ColorGray)
}
