package ui

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var helpGlobal = `## Global

| Key | Action |
|-----|--------|
| Tab / Shift+Tab | Switch tabs |
| q / Ctrl+C | Quit |
| ? | Toggle help |
| r | Refresh |
`

var helpOverview = `## Overview

| Key | Action |
|-----|--------|
| / | Filter indices |
| Esc | Clear filters |
| ←→ | Select index |
| ↑↓ | Select node |
| Enter | Shard info |
| 1-9 | Toggle alias |
| U/R/I | Shard states |
| . | Toggle system indices |
| c | Create index |
| d | Delete index |
| a/A | Add/remove alias |
`

var helpWorkbench = `## Workbench

| Key | Action |
|-----|--------|
| Enter | Edit |
| Tab | Autocomplete / next |
| Ctrl+R | Execute |
| Ctrl+E | Toggle REST/ES|QL |
| Alt+F | Format JSON |
| Ctrl+S | Save bookmark |
| Ctrl+B | Load bookmark |
| Ctrl+F | Search response |
| Enter/n | Next match |
| Shift+Enter/N | Prev match |
| Ctrl+Y | Copy body/response |
| Ctrl+A | Select all (body) |
| Ctrl+C | Copy selection |
| Ctrl+V | Paste |
| Ctrl+Z | Undo |
| Ctrl+Shift+Z | Redo |
| Shift+Arrow | Select text |
| Up/Down | Navigate |
| Esc | Cancel |
`

var helpBrowser = `## Browser

| Key | Action |
|-----|--------|
| / | Filter indices |
| left/right | Switch panes |
| up/down | Scroll / select |
| Enter | Load documents |
| Ctrl+Y | Copy document |
`

var helpMappings = `## Mappings

| Key | Action |
|-----|--------|
| / | Filter |
| Ctrl+F | Search |
| Ctrl+Y | Copy mappings |
| Enter/n | Next match |
| Shift+Enter/N | Prev match |
| left/right | Switch panes |
| Enter | Load mappings |
| t | Toggle tree view |
| s | Toggle settings |
| up/down | Scroll |
`

var helpCluster = `## Cluster

| Key | Action |
|-----|--------|
| 1-9 | Switch view |
| / | Filter |
| Esc | Clear filter |
| Enter | Details (Settings) |
| up/down | Scroll |
`

var helpTasks = `## Tasks

| Key | Action |
|-----|--------|
| Enter | Task details |
| Ctrl+F | Search |
| c | Cancel task |
| y/n | Confirm |
| up/down | Select |
| r | Refresh |
`

func renderHelp(width, height, activeTab int) string {
	styleName := "dark"
	if !lipgloss.HasDarkBackground() {
		styleName = "light"
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(styleName),
		glamour.WithWordWrap(40),
	)

	var tabHelp string
	switch activeTab {
	case TabOverview:
		tabHelp = helpOverview
	case TabWorkbench:
		tabHelp = helpWorkbench
	case TabBrowser:
		tabHelp = helpBrowser
	case TabMappings:
		tabHelp = helpMappings
	case TabCluster:
		tabHelp = helpCluster
	case TabTasks:
		tabHelp = helpTasks
	}

	content, _ := r.Render(helpGlobal + tabHelp + "\n*Press ? to close*")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(0, 1)

	box := boxStyle.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
