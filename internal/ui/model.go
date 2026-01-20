package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
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
	switch msg := msg.(type) {
	case connectedMsg:
		m.connected = true
		m.cluster = msg.state
		m.err = nil
	case errMsg:
		m.err = msg.err
		m.connected = false
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
		case "r":
			return m, m.connect()
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
	} else {
		info := fmt.Sprintf("Connected!\n\nIndices: %d\nNodes: %d\nShards: %d\nAliases: %d",
			len(m.cluster.Indices),
			len(m.cluster.Nodes),
			len(m.cluster.Shards),
			len(m.cluster.Aliases))
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(info)
	}

	// Status bar
	status = "q: quit  Tab: switch view  r: refresh"
	statusBar := StatusBarStyle.Width(m.width).Render(status)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}
