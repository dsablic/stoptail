package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
)

const (
	TabOverview = iota
	TabWorkbench
	TabNodes
	TabTasks
)

type Model struct {
	client       *es.Client
	cfg          *config.Config
	cluster      *es.ClusterState
	overview     OverviewModel
	workbench    WorkbenchModel
	nodes        NodesModel
	tasks        TasksModel
	spinner      spinner.Model
	activeTab    int
	width        int
	height       int
	connected    bool
	loading      bool
	err          error
	quitting     bool
	showHelp     bool
}

type connectedMsg struct{ state *es.ClusterState }
type nodesStateMsg struct{ state *es.NodesState }
type tasksMsg struct{ tasks []es.TaskInfo }
type taskCancelledMsg struct{ err error }
type errMsg struct{ err error }

func New(client *es.Client, cfg *config.Config) Model {
	wb := NewWorkbench()
	wb.SetClient(client)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)

	return Model{
		client:    client,
		cfg:       cfg,
		overview:  NewOverview(),
		workbench: wb,
		nodes:     NewNodes(),
		tasks:     NewTasks(),
		spinner:   s,
		activeTab: TabOverview,
		loading:   true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.connect())
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

func (m Model) fetchNodes() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		state, err := m.client.FetchNodesState(ctx)
		if err != nil {
			return errMsg{err}
		}
		return nodesStateMsg{state}
	}
}

func (m Model) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tasks, err := m.client.FetchTasks(ctx)
		if err != nil {
			return errMsg{err}
		}
		return tasksMsg{tasks}
	}
}

func (m Model) cancelTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.client.CancelTask(ctx, taskID)
		return taskCancelledMsg{err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case connectedMsg:
		m.connected = true
		m.loading = false
		m.cluster = msg.state
		m.overview.SetCluster(msg.state)
		m.err = nil
	case nodesStateMsg:
		m.loading = false
		m.nodes.SetState(msg.state)
	case tasksMsg:
		m.loading = false
		m.tasks.SetTasks(msg.tasks)
	case taskCancelledMsg:
		m.tasks.ClearConfirming()
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
		}
	case taskCancelRequestMsg:
		return m, m.cancelTask(msg.taskID)
	case errMsg:
		m.err = msg.err
		m.loading = false
		m.connected = false
	case executeResultMsg:
		m.workbench, cmd = m.workbench.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			// Only quit if not in a focused input
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.quitting = true
				return m, tea.Quit
			}
			if m.activeTab == TabWorkbench && m.workbench.focus != FocusPath && m.workbench.focus != FocusBody {
				m.quitting = true
				return m, tea.Quit
			}
			if m.activeTab == TabNodes {
				m.quitting = true
				return m, tea.Quit
			}
			if m.activeTab == TabTasks && m.tasks.confirming == "" {
				m.quitting = true
				return m, tea.Quit
			}
		case "tab":
			// Global tab to switch views, unless in focused input
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.activeTab = TabWorkbench
				m.workbench.Blur()
				return m, nil
			}
			if m.activeTab == TabWorkbench && !m.workbench.HasActiveInput() {
				m.activeTab = TabNodes
				m.workbench.Blur()
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
			}
			if m.activeTab == TabNodes {
				m.activeTab = TabTasks
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
			}
			if m.activeTab == TabTasks && m.tasks.confirming == "" {
				m.activeTab = TabOverview
				return m, nil
			}
		case "shift+tab":
			// Switch backward through tabs
			if m.activeTab == TabWorkbench {
				m.activeTab = TabOverview
				m.workbench.Blur()
				return m, nil
			}
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.activeTab = TabTasks
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
			}
			if m.activeTab == TabTasks && m.tasks.confirming == "" {
				m.activeTab = TabNodes
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
			}
			if m.activeTab == TabNodes {
				m.activeTab = TabWorkbench
				m.workbench.Blur()
				return m, nil
			}
		case "r":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.connect())
			}
			if m.activeTab == TabNodes {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
			}
			if m.activeTab == TabTasks {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
			}
		case "enter":
			// From overview, enter on index switches to workbench
			if m.activeTab == TabOverview && !m.overview.filterActive {
				if idx := m.overview.SelectedIndex(); idx != "" {
					m.workbench.Prefill(idx)
					m.activeTab = TabWorkbench
					m.workbench.Focus()
					return m, nil
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overview.SetSize(msg.Width, msg.Height-4)
		m.workbench.SetSize(msg.Width, msg.Height-4)
		m.nodes.SetSize(msg.Width, msg.Height-4)
		m.tasks.SetSize(msg.Width, msg.Height-4)
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if msg.Y == 1 {
				overviewWidth := lipgloss.Width(InactiveTabStyle.Render("Overview"))
				workbenchWidth := lipgloss.Width(InactiveTabStyle.Render("Workbench"))
				nodesWidth := lipgloss.Width(InactiveTabStyle.Render("Nodes"))

				if msg.X < overviewWidth {
					m.activeTab = TabOverview
					m.workbench.Blur()
				} else if msg.X < overviewWidth+workbenchWidth {
					m.activeTab = TabWorkbench
				} else if msg.X < overviewWidth+workbenchWidth+nodesWidth {
					m.activeTab = TabNodes
					m.workbench.Blur()
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, m.fetchNodes())
				} else {
					m.activeTab = TabTasks
					m.workbench.Blur()
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, m.fetchTasks())
				}
				return m, nil
			}
		}
	}

	// Delegate to active tab
	if m.connected {
		delegateMsg := msg
		if mouseMsg, ok := msg.(tea.MouseMsg); ok {
			mouseMsg.Y -= 2
			delegateMsg = mouseMsg
		}
		switch m.activeTab {
		case TabOverview:
			m.overview, cmd = m.overview.Update(delegateMsg)
		case TabWorkbench:
			m.workbench, cmd = m.workbench.Update(delegateMsg)
		case TabNodes:
			m.nodes, cmd = m.nodes.Update(delegateMsg)
		case TabTasks:
			var cmd tea.Cmd
			m.tasks, cmd = m.tasks.Update(delegateMsg)
			if cmd != nil {
				return m, cmd
			}
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return renderHelp(m.width, m.height)
	}

	// Header
	status := "connecting..."
	if m.connected {
		status = "connected"
	}
	if m.err != nil {
		status = "error"
	}
	if m.loading {
		status = m.spinner.View() + " loading"
	}
	headerText := fmt.Sprintf("stoptail · %s [%s]", m.cfg.MaskedURL(), status)
	header := HeaderStyle.Width(m.width).Render(headerText)

	// Tabs
	overviewTab := InactiveTabStyle.Render("Overview")
	workbenchTab := InactiveTabStyle.Render("Workbench")
	nodesTab := InactiveTabStyle.Render("Nodes")
	tasksTab := InactiveTabStyle.Render("Tasks")
	switch m.activeTab {
	case TabOverview:
		overviewTab = ActiveTabStyle.Render("Overview")
	case TabWorkbench:
		workbenchTab = ActiveTabStyle.Render("Workbench")
	case TabNodes:
		nodesTab = ActiveTabStyle.Render("Nodes")
	case TabTasks:
		tasksTab = ActiveTabStyle.Render("Tasks")
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, overviewTab, workbenchTab, nodesTab, tasksTab)

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
			Render(m.spinner.View() + " Connecting...")
	} else {
		switch m.activeTab {
		case TabOverview:
			content = m.overview.View()
		case TabWorkbench:
			content = m.workbench.View()
		case TabNodes:
			content = m.nodes.View()
		case TabTasks:
			content = m.tasks.View()
		}
	}

	// Status bar
	var statusText string
	switch m.activeTab {
	case TabOverview:
		statusText = "q: quit  Tab: workbench  Shift+Tab: tasks  r: refresh  /: filter  ←→: select index  ↑↓: scroll  Enter: open"
	case TabWorkbench:
		statusText = "q: quit  Tab: autocomplete  Shift+Tab: overview  Ctrl+R: execute  Ctrl+Y: copy  Ctrl+F: search  Esc: deactivate"
	case TabNodes:
		statusText = "q: quit  Tab: tasks  Shift+Tab: workbench  r: refresh  1-4: views  ↑↓: scroll"
	case TabTasks:
		statusText = "q: quit  Tab: overview  Shift+Tab: nodes  r: refresh  c: cancel  ↑↓: select"
	}
	statusBar := StatusBarStyle.Width(m.width).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}
