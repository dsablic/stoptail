package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func renderHelp(width, height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorWhite).
		MarginBottom(1).
		Render("stoptail - Elasticsearch TUI")

	sections := []struct {
		header string
		keys   [][]string
	}{
		{
			header: "Global",
			keys: [][]string{
				{"Tab / Shift+Tab", "Switch views"},
				{"q / Ctrl+C", "Quit"},
				{"?", "Toggle help"},
				{"r", "Refresh data"},
			},
		},
		{
			header: "Overview",
			keys: [][]string{
				{"/", "Focus filter"},
				{"Esc", "Clear all filters"},
				{"up/down/left/right", "Navigate grid"},
				{"Enter", "Open index in Workbench"},
				{"1-9", "Toggle alias filters"},
				{"U/R/I", "Show Unassigned/Relocating/Initializing"},
			},
		},
		{
			header: "Workbench",
			keys: [][]string{
				{"Enter", "Activate editor"},
				{"Tab", "Trigger autocomplete / cycle focus"},
				{"Ctrl+R", "Execute request"},
				{"Ctrl+Y", "Copy body/response to clipboard"},
				{"Up/Down", "Navigate completions"},
				{"Esc", "Dismiss completions / deactivate"},
			},
		},
		{
			header: "Nodes",
			keys: [][]string{
				{"1/2/3", "Switch view (Mem/Disk/Fielddata)"},
				{"up/down", "Scroll"},
				{"r", "Refresh"},
			},
		},
		{
			header: "Tasks",
			keys: [][]string{
				{"c", "Cancel selected task"},
				{"y/n", "Confirm/abort cancel"},
				{"up/down", "Select task"},
				{"r", "Refresh"},
			},
		},
	}

	keyStyle := lipgloss.NewStyle().Foreground(ColorBlue).Width(20)
	descStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorYellow).MarginTop(1)

	var content string
	content += title + "\n\n"

	for _, section := range sections {
		content += headerStyle.Render(section.header) + "\n"
		for _, kv := range section.keys {
			content += keyStyle.Render(kv[0]) + descStyle.Render(kv[1]) + "\n"
		}
	}

	content += "\n" + lipgloss.NewStyle().Foreground(ColorGray).Render("Mouse supported - Press ? to close")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(50)

	box := boxStyle.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
