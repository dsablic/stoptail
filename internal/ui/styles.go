package ui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
)

var (
	ColorGreen  color.Color
	ColorYellow color.Color
	ColorRed    color.Color
	ColorBlue   color.Color
	ColorPurple color.Color
	ColorGray   color.Color
	ColorWhite  color.Color

	HeaderBg      color.Color
	ActiveBg      color.Color
	SpinnerClr    color.Color
	TextBg        color.Color
	ColorOnAccent color.Color

	// Base styles
	HeaderStyle      lipgloss.Style
	TabStyle         lipgloss.Style
	ActiveTabStyle   lipgloss.Style
	InactiveTabStyle lipgloss.Style
	StatusBarStyle   lipgloss.Style
	HelpStyle        lipgloss.Style

	ThemeDark bool
)

func init() {
	SetTheme("auto")
}

func SetTheme(theme string) {
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	switch theme {
	case "dark":
		isDark = true
	case "light":
		isDark = false
	}

	ThemeDark = isDark

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
