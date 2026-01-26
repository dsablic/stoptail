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
	TabBrowser
	TabMappings
	TabCluster
	TabTasks
)

type Model struct {
	client       *es.Client
	cfg          *config.Config
	cluster      *es.ClusterState
	overview     OverviewModel
	workbench    WorkbenchModel
	browser      BrowserModel
	mappings     MappingsModel
	nodes        NodesModel
	tasks        TasksModel
	spinner      spinner.Model
	activeTab    int
	width        int
	height       int
	connected bool
	loading   bool
	err       error
	showHelp  bool
}

type connectedMsg struct{ state *es.ClusterState }
type nodesStateMsg struct{ state *es.NodesState }
type clusterSettingsMsg struct{ settings *es.ClusterSettings }
type threadPoolsMsg struct{ pools []es.ThreadPoolInfo }
type hotThreadsMsg struct{ threads string }
type templatesMsg struct{ templates []es.IndexTemplate }
type tasksMsg struct{ tasks []es.TaskInfo }
type pendingTasksMsg struct{ tasks []es.PendingTask }
type taskCancelledMsg struct{ err error }
type mappingsMsg struct {
	mappings  *es.IndexMappings
	analyzers []es.AnalyzerInfo
	err       error
}
type settingsMsg struct {
	settings *es.IndexSettings
	err      error
}
type errMsg struct{ err error }

func New(client *es.Client, cfg *config.Config) Model {
	wb := NewWorkbench()
	wb.SetClient(client)

	ov := NewOverview()
	ov.SetClient(client)

	br := NewBrowser()
	br.SetClient(client)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)

	return Model{
		client:    client,
		cfg:       cfg,
		overview:  ov,
		workbench: wb,
		browser:   br,
		mappings:  NewMappings(),
		nodes:     NewNodes(),
		tasks:     NewTasks(),
		spinner:   s,
		activeTab: TabOverview,
		loading:   true,
	}
}

func (m Model) hasActiveInput() bool {
	switch m.activeTab {
	case TabOverview:
		return m.overview.filterActive || m.overview.HasModal()
	case TabWorkbench:
		return m.workbench.HasActiveInput()
	case TabBrowser:
		return m.browser.HasActiveInput()
	case TabMappings:
		return m.mappings.filterActive || m.mappings.search.Active()
	case TabCluster:
		return m.nodes.filterActive || m.nodes.settingDetail != nil
	case TabTasks:
		return m.tasks.confirming != "" || m.tasks.HasModal() || m.tasks.search.Active()
	}
	return false
}

func (m *Model) switchTab(tab int) tea.Cmd {
	m.activeTab = tab
	return nil
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

func (m Model) fetchCluster() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
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

func (m Model) fetchClusterSettings() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		settings, err := m.client.FetchClusterSettings(ctx)
		if err != nil {
			return errMsg{err}
		}
		return clusterSettingsMsg{settings}
	}
}

func (m Model) fetchThreadPools() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pools, err := m.client.FetchThreadPools(ctx)
		if err != nil {
			return errMsg{err}
		}
		return threadPoolsMsg{pools}
	}
}

func (m Model) fetchHotThreads() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		threads, err := m.client.FetchHotThreads(ctx)
		if err != nil {
			return errMsg{err}
		}
		return hotThreadsMsg{threads}
	}
}

func (m Model) fetchTemplates() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		templates, err := m.client.FetchIndexTemplates(ctx)
		if err != nil {
			return errMsg{err}
		}
		return templatesMsg{templates}
	}
}

func (m Model) fetchClusterTab() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchNodes(), m.fetchClusterSettings(), m.fetchThreadPools(), m.fetchHotThreads(), m.fetchTemplates())
}

func (m Model) fetchTasksTab() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchTasks(), m.fetchPendingTasks())
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

func (m Model) fetchPendingTasks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tasks, err := m.client.FetchPendingTasks(ctx)
		if err != nil {
			return errMsg{err}
		}
		return pendingTasksMsg{tasks}
	}
}

func (m Model) fetchMappings(indexName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		mappings, err := m.client.FetchIndexMappings(ctx, indexName)
		if err != nil {
			return mappingsMsg{err: err}
		}
		analyzers, _ := m.client.FetchIndexAnalyzers(ctx, indexName)
		return mappingsMsg{mappings: mappings, analyzers: analyzers}
	}
}

func (m Model) fetchSettings(indexName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		settings, err := m.client.FetchIndexSettings(ctx, indexName)
		if err != nil {
			return settingsMsg{err: err}
		}
		return settingsMsg{settings: settings}
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
	case clusterSettingsMsg:
		m.nodes.SetClusterSettings(msg.settings)
	case threadPoolsMsg:
		m.nodes.SetThreadPools(msg.pools)
	case hotThreadsMsg:
		m.nodes.SetHotThreads(msg.threads)
	case templatesMsg:
		m.nodes.SetTemplates(msg.templates)
	case tasksMsg:
		m.loading = false
		m.tasks.SetTasks(msg.tasks)
	case pendingTasksMsg:
		m.tasks.SetPendingTasks(msg.tasks)
	case mappingsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.mappings.SetMappings(msg.mappings, msg.analyzers)
		}
	case fetchMappingsMsg:
		m.mappings.SetLoading(msg.indexName)
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchMappings(msg.indexName))
	case fetchSettingsMsg:
		m.mappings.settingsLoading = true
		return m, m.fetchSettings(msg.indexName)
	case settingsMsg:
		m.mappings.settingsLoading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.mappings.SetSettings(msg.settings)
		}
		return m, nil
	case taskCancelledMsg:
		m.tasks.ClearConfirming()
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.loading = true
			return m, m.fetchTasksTab()
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
		// Global keys - skip when any input is active (typing in search, filter, editor, etc.)
		if !m.hasActiveInput() {
			switch msg.String() {
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			case "q":
				return m, tea.Quit
			case "r":
				m.loading = true
				switch m.activeTab {
				case TabOverview, TabMappings:
					return m, tea.Batch(m.spinner.Tick, m.connect())
				case TabCluster:
					return m, m.fetchClusterTab()
				case TabTasks:
					return m, m.fetchTasksTab()
				}
			case "tab":
				if m.showHelp {
					break
				}
				switch m.activeTab {
				case TabOverview:
					m.workbench.Blur()
					return m, m.switchTab(TabWorkbench)
				case TabWorkbench:
					m.workbench.Blur()
					if m.cluster != nil {
						m.browser.SetIndices(m.cluster.Indices)
					}
					return m, m.switchTab(TabBrowser)
				case TabBrowser:
					if m.cluster != nil {
						m.mappings.SetIndices(m.cluster.Indices)
					}
					return m, m.switchTab(TabMappings)
				case TabMappings:
					m.loading = true
					return m, tea.Batch(m.switchTab(TabCluster), m.fetchClusterTab())
				case TabCluster:
					m.loading = true
					return m, tea.Batch(m.switchTab(TabTasks), m.fetchTasksTab())
				case TabTasks:
					return m, m.switchTab(TabOverview)
				}
			case "shift+tab":
				if m.showHelp {
					break
				}
				switch m.activeTab {
				case TabWorkbench:
					m.workbench.Blur()
					return m, m.switchTab(TabOverview)
				case TabBrowser:
					return m, m.switchTab(TabWorkbench)
				case TabMappings:
					if m.cluster != nil {
						m.browser.SetIndices(m.cluster.Indices)
					}
					return m, m.switchTab(TabBrowser)
				case TabOverview:
					m.loading = true
					return m, tea.Batch(m.switchTab(TabTasks), m.fetchTasksTab())
				case TabTasks:
					m.loading = true
					return m, tea.Batch(m.switchTab(TabCluster), m.fetchClusterTab())
				case TabCluster:
					if m.cluster != nil {
						m.mappings.SetIndices(m.cluster.Indices)
					}
					return m, m.switchTab(TabMappings)
				}
			}
		}

		// Keys that work even with active input
		switch msg.String() {
		case "ctrl+c":
			if m.activeTab == TabWorkbench && m.workbench.focus == FocusBody && m.workbench.editor.selection.Active {
				break
			}
			return m, tea.Quit
		}
	case IndexCreatedMsg, IndexDeletedMsg, AliasAddedMsg, AliasRemovedMsg:
		m.overview, cmd = m.overview.Update(msg)
		if hasNoError(msg) {
			return m, tea.Batch(cmd, m.fetchCluster())
		}
		return m, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overview.SetSize(msg.Width, msg.Height-4)
		m.workbench.SetSize(msg.Width, msg.Height-4)
		m.browser.SetSize(msg.Width, msg.Height-4)
		m.mappings.SetSize(msg.Width, msg.Height-4)
		m.nodes.SetSize(msg.Width, msg.Height-4)
		m.tasks.SetSize(msg.Width, msg.Height-4)
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if msg.Y == 1 {
				overviewWidth := lipgloss.Width(InactiveTabStyle.Render("Overview"))
				workbenchWidth := lipgloss.Width(InactiveTabStyle.Render("Workbench"))
				browserWidth := lipgloss.Width(InactiveTabStyle.Render("Browser"))
				mappingsWidth := lipgloss.Width(InactiveTabStyle.Render("Mappings"))
				nodesWidth := lipgloss.Width(InactiveTabStyle.Render("Cluster"))

				if msg.X < overviewWidth {
					m.activeTab = TabOverview
					m.workbench.Blur()
				} else if msg.X < overviewWidth+workbenchWidth {
					m.activeTab = TabWorkbench
				} else if msg.X < overviewWidth+workbenchWidth+browserWidth {
					m.activeTab = TabBrowser
					m.workbench.Blur()
					if m.cluster != nil {
						m.browser.SetIndices(m.cluster.Indices)
					}
				} else if msg.X < overviewWidth+workbenchWidth+browserWidth+mappingsWidth {
					m.activeTab = TabMappings
					m.workbench.Blur()
					if m.cluster != nil {
						m.mappings.SetIndices(m.cluster.Indices)
					}
				} else if msg.X < overviewWidth+workbenchWidth+browserWidth+mappingsWidth+nodesWidth {
					m.activeTab = TabCluster
					m.workbench.Blur()
					m.loading = true
					return m, m.fetchClusterTab()
				} else {
					m.activeTab = TabTasks
					m.workbench.Blur()
					m.loading = true
					return m, m.fetchTasksTab()
				}
				return m, nil
			}
		}
	}

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
		case TabBrowser:
			m.browser, cmd = m.browser.Update(delegateMsg)
		case TabMappings:
			m.mappings, cmd = m.mappings.Update(delegateMsg)
		case TabCluster:
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
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.showHelp {
		return renderHelp(m.width, m.height, m.activeTab)
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

	overviewTab := InactiveTabStyle.Render("Overview")
	workbenchTab := InactiveTabStyle.Render("Workbench")
	browserTab := InactiveTabStyle.Render("Browser")
	mappingsTab := InactiveTabStyle.Render("Mappings")
	nodesTab := InactiveTabStyle.Render("Cluster")
	tasksTab := InactiveTabStyle.Render("Tasks")
	switch m.activeTab {
	case TabOverview:
		overviewTab = ActiveTabStyle.Render("Overview")
	case TabWorkbench:
		workbenchTab = ActiveTabStyle.Render("Workbench")
	case TabBrowser:
		browserTab = ActiveTabStyle.Render("Browser")
	case TabMappings:
		mappingsTab = ActiveTabStyle.Render("Mappings")
	case TabCluster:
		nodesTab = ActiveTabStyle.Render("Cluster")
	case TabTasks:
		tasksTab = ActiveTabStyle.Render("Tasks")
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, overviewTab, workbenchTab, browserTab, mappingsTab, nodesTab, tasksTab)

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
		case TabBrowser:
			content = m.browser.View()
		case TabMappings:
			content = m.mappings.View()
		case TabCluster:
			content = m.nodes.View()
		case TabTasks:
			content = m.tasks.View()
		}
	}

	var statusText string
	switch m.activeTab {
	case TabOverview:
		statusText = "q: quit  Tab: workbench  Shift+Tab: tasks  r: refresh  /: filter  ←→: index  ↑↓: node  Enter: shard info  .: system"
		if m.overview.showSystem {
			statusText += " [on]"
		}
	case TabWorkbench:
		statusText = "q: quit  Tab: browser  Shift+Tab: overview  Ctrl+R: execute  Ctrl+Y: copy  Ctrl+F: search  Esc: deactivate"
	case TabBrowser:
		statusText = "q: quit  Tab: mappings  Shift+Tab: workbench  /: filter  ←→: panes  ↑↓: scroll  Ctrl+Y: copy"
	case TabMappings:
		statusText = "q: quit  Tab: cluster  Shift+Tab: browser  r: refresh  /: filter  ←→: panes  ↑↓: scroll  t: tree  s: settings  Ctrl+Y: copy  Ctrl+F: search"
	case TabCluster:
		statusText = "q: quit  Tab: tasks  Shift+Tab: mappings  r: refresh  1-7: views  /: filter  ↑↓: scroll"
	case TabTasks:
		statusText = "q: quit  Tab: overview  Shift+Tab: cluster  r: refresh  Enter: details  c: cancel  ↑↓: select"
	}

	var clipboardMsg string
	switch m.activeTab {
	case TabWorkbench:
		clipboardMsg = m.workbench.ClipboardMessage()
	case TabBrowser:
		clipboardMsg = m.browser.ClipboardMessage()
	case TabMappings:
		clipboardMsg = m.mappings.ClipboardMessage()
	}
	if clipboardMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(ColorGreen)
		if clipboardMsg != "Copied!" {
			msgStyle = msgStyle.Foreground(ColorYellow)
		}
		statusText = msgStyle.Render(clipboardMsg) + "  " + statusText
	}

	statusBar := StatusBarStyle.Width(m.width).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}

func hasNoError(msg tea.Msg) bool {
	switch m := msg.(type) {
	case IndexCreatedMsg:
		return m.Err == nil
	case IndexDeletedMsg:
		return m.Err == nil
	case AliasAddedMsg:
		return m.Err == nil
	case AliasRemovedMsg:
		return m.Err == nil
	}
	return false
}
