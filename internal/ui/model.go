package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
)

const (
	TabOverview = iota
	TabWorkbench
)

type Model struct {
	client    *es.Client
	cfg       *config.Config
	cluster   *es.ClusterState
	overview  OverviewModel
	activeTab int
	width     int
	height    int
	connected bool
	err       error
	quitting  bool
}

type connectedMsg struct{ state *es.ClusterState }
type errMsg struct{ err error }

func New(client *es.Client, cfg *config.Config) Model {
	return Model{
		client:    client,
		cfg:       cfg,
		overview:  NewOverview(),
		activeTab: TabOverview,
	}
}

func (m Model) Init() tea.Cmd {
	return m.connect()
}

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := m.client.Ping(ctx); err != nil {
			return errMsg{err}
		}
		state, err := m.client.FetchClusterState(ctx)
		if err != nil {
			return errMsg{err}
		}
		return connectedMsg{state}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case connectedMsg:
		m.connected = true
		m.cluster = msg.state
		m.overview.SetCluster(msg.state)
		m.err = nil
	case errMsg:
		m.err = msg.err
		m.connected = false
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.activeTab == TabOverview && m.overview.filterActive {
				// Let overview handle it
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case "tab":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.activeTab = (m.activeTab + 1) % 2
				return m, nil
			}
		case "r":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				return m, m.connect()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overview.SetSize(msg.Width, msg.Height-4)
	}

	// Delegate to active tab
	if m.activeTab == TabOverview && m.connected {
		m.overview, cmd = m.overview.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	status := "connecting..."
	if m.connected {
		status = "connected"
	}
	if m.err != nil {
		status = "error"
	}
	headerText := fmt.Sprintf("stoptail · %s [%s]", m.cfg.MaskedURL(), status)
	header := HeaderStyle.Width(m.width).Render(headerText)

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

	// Content
	contentHeight := m.height - 4
	var content string
	if m.err != nil {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Foreground(ColorRed).
			Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("Connection error:\n%v\n\nPress 'r' to retry", m.err))
	} else if !m.connected {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Connecting...")
	} else if m.activeTab == TabOverview {
		content = m.overview.View()
	} else {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Workbench (coming soon)")
	}

	// Status bar
	statusText := "q: quit  Tab: switch view  r: refresh"
	if m.activeTab == TabOverview {
		statusText = "q: quit  Tab: switch  r: refresh  /: filter  ←→↑↓: scroll"
	}
	statusBar := StatusBarStyle.Width(m.width).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}
