package ui

import (
	_ "embed"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

//go:embed help.md
var helpMarkdown string

func renderHelp(width, height int) string {
	styleName := "dark"
	if !lipgloss.HasDarkBackground() {
		styleName = "light"
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(styleName),
		glamour.WithWordWrap(46),
	)

	content, _ := r.Render(helpMarkdown)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(0, 1)

	box := boxStyle.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
