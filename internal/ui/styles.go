package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorGreen  = lipgloss.Color("#22c55e")
	ColorYellow = lipgloss.Color("#eab308")
	ColorRed    = lipgloss.Color("#ef4444")
	ColorBlue   = lipgloss.Color("#3b82f6")
	ColorGray   = lipgloss.Color("#6b7280")
	ColorWhite  = lipgloss.Color("#f9fafb")

	// Base styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(lipgloss.Color("#1f2937")).
			Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	ActiveTabStyle = TabStyle.
			Bold(true).
			Foreground(ColorWhite).
			Background(lipgloss.Color("#374151"))

	InactiveTabStyle = TabStyle.
				Foreground(ColorGray)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorGray)
)
