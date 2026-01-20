package ui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	TabOverview = iota
	TabWorkbench
)

type Model struct {
	activeTab int
	width     int
	height    int
	quitting  bool
}

func New() Model {
	return Model{
		activeTab: TabOverview,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	header := HeaderStyle.Width(m.width).Render("stoptail · not connected")

	// Tabs
	var tabs string
	if m.activeTab == TabOverview {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			ActiveTabStyle.Render("Overview"),
			InactiveTabStyle.Render("Workbench"),
		)
	} else {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			InactiveTabStyle.Render("Overview"),
			ActiveTabStyle.Render("Workbench"),
		)
	}

	// Content placeholder
	contentHeight := m.height - 4 // header + tabs + status
	content := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render("Press Tab to switch views")

	// Status bar
	status := StatusBarStyle.Width(m.width).Render("q: quit  Tab: switch view")

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, status)
}
