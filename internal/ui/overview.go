package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type ModalInitMsg struct{}

type IndexCreatedMsg struct{ Err error }
type IndexDeletedMsg struct{ Err error }
type AliasAddedMsg struct{ Err error }
type AliasRemovedMsg struct{ Err error }
type AllocationExplainMsg struct {
	Result *es.AllocationExplain
	Err    error
}
type RecoveryMsg struct {
	Index  string
	Shard  string
	Result *es.RecoveryInfo
}

type OverviewModel struct {
	cluster            *es.ClusterState
	client             *es.Client
	filter             textinput.Model
	filterActive       bool
	aliasFilters       map[string]bool
	shardStateFilter   string
	showSystem         bool
	scrollX            int
	scrollY            int
	selectedIndex      int
	selectedNode       int
	width              int
	height             int
	modal              *Modal
	spinner            spinner.Model
	operationMsg       string
	allocationExplain  *es.AllocationExplain
	allocationLoading  bool
	shardPicker        *ShardPicker
	shardInfo          *es.ShardInfo
	recoveryInfo       *es.RecoveryInfo
}

func NewOverview() OverviewModel {
	ti := textinput.New()
	ti.Placeholder = "Filter indices..."
	ti.CharLimit = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)

	return OverviewModel{
		filter:       ti,
		aliasFilters: make(map[string]bool),
		spinner:      s,
	}
}

func (m *OverviewModel) SetCluster(cluster *es.ClusterState) {
	m.cluster = cluster
}

func (m *OverviewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *OverviewModel) SetClient(client *es.Client) {
	m.client = client
}

func (m OverviewModel) Update(msg tea.Msg) (OverviewModel, tea.Cmd) {
	var cmd tea.Cmd

	if _, ok := msg.(ModalInitMsg); ok && m.modal != nil {
		return m, m.modal.Init()
	}

	if m.modal != nil {
		cmd := m.modal.Update(msg)
		if m.modal.Cancelled() {
			m.modal = nil
			return m, nil
		}
		if m.modal.Done() {
			return m.handleModalDone()
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.operationMsg != "" {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "esc", "enter":
				m.filterActive = false
				m.filter.Blur()
				return m, nil
			}
			m.filter, cmd = m.filter.Update(msg)
			m.selectedIndex = 0
			m.scrollX = 0
			return m, cmd
		}

		if m.shardPicker != nil && m.shardInfo == nil {
			switch msg.String() {
			case "left", "h":
				m.shardPicker.Left()
				return m, nil
			case "right", "l":
				m.shardPicker.Right()
				return m, nil
			case "up", "k":
				m.shardPicker.Up()
				return m, nil
			case "down", "j":
				m.shardPicker.Down()
				return m, nil
			case "enter":
				sh := m.shardPicker.Selected()
				if sh != nil {
					return m.showShardInfo(sh)
				}
				return m, nil
			case "esc":
				m.shardPicker = nil
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			m.filterActive = true
			m.filter.Focus()
			return m, textinput.Blink
		case "esc":
			if m.shardInfo != nil {
				m.shardInfo = nil
				m.allocationExplain = nil
				m.recoveryInfo = nil
				return m, nil
			}
			if m.allocationExplain != nil {
				m.allocationExplain = nil
				return m, nil
			}
			m.filter.SetValue("")
			m.aliasFilters = make(map[string]bool)
			m.shardStateFilter = ""
			m.selectedIndex = 0
			m.selectedNode = 0
		case "U":
			if m.shardStateFilter == "UNASSIGNED" {
				m.shardStateFilter = ""
			} else {
				m.shardStateFilter = "UNASSIGNED"
			}
			m.selectedIndex = 0
			m.selectedNode = 0
			m.scrollX = 0
			m.scrollY = 0
		case "R":
			if m.shardStateFilter == "RELOCATING" {
				m.shardStateFilter = ""
			} else {
				m.shardStateFilter = "RELOCATING"
			}
			m.selectedIndex = 0
			m.selectedNode = 0
			m.scrollX = 0
			m.scrollY = 0
		case "I":
			if m.shardStateFilter == "INITIALIZING" {
				m.shardStateFilter = ""
			} else {
				m.shardStateFilter = "INITIALIZING"
			}
			m.selectedIndex = 0
			m.selectedNode = 0
			m.scrollX = 0
			m.scrollY = 0
		case ".":
			m.showSystem = !m.showSystem
			m.selectedIndex = 0
			m.selectedNode = 0
			m.scrollX = 0
			m.scrollY = 0
		case "up", "k":
			nodes := m.getNodeList()
			if m.selectedNode > 0 {
				m.selectedNode--
				if m.selectedNode < m.scrollY {
					m.scrollY = m.selectedNode
				}
			}
			if m.selectedNode >= len(nodes) {
				m.selectedNode = len(nodes) - 1
			}
		case "down", "j":
			nodes := m.getNodeList()
			if m.selectedNode < len(nodes)-1 {
				m.selectedNode++
				maxVisible := m.maxVisibleNodes()
				if m.selectedNode >= m.scrollY+maxVisible {
					m.scrollY = m.selectedNode - maxVisible + 1
				}
			}
		case "pgup":
			pageSize := m.maxVisibleNodes()
			m.selectedNode -= pageSize
			if m.selectedNode < 0 {
				m.selectedNode = 0
			}
			if m.selectedNode < m.scrollY {
				m.scrollY = m.selectedNode
			}
		case "pgdown":
			nodes := m.getNodeList()
			pageSize := m.maxVisibleNodes()
			m.selectedNode += pageSize
			if m.selectedNode >= len(nodes) {
				m.selectedNode = len(nodes) - 1
			}
			maxVisible := m.maxVisibleNodes()
			if m.selectedNode >= m.scrollY+maxVisible {
				m.scrollY = m.selectedNode - maxVisible + 1
			}
		case "left", "h":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				if m.selectedIndex < m.scrollX {
					m.scrollX = m.selectedIndex
				}
			}
		case "right", "l":
			indices := m.filteredIndices()
			if m.selectedIndex < len(indices)-1 {
				m.selectedIndex++
				visibleCols := m.visibleColumns()
				if m.selectedIndex >= m.scrollX+visibleCols {
					m.scrollX = m.selectedIndex - visibleCols + 1
				}
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.cluster != nil {
				aliases := m.cluster.UniqueAliases()
				idx := int(msg.String()[0] - '1')
				if idx < len(aliases) {
					alias := aliases[idx]
					m.aliasFilters[alias] = !m.aliasFilters[alias]
					m.selectedIndex = 0
					m.selectedNode = 0
					m.scrollX = 0
					m.scrollY = 0
				}
			}
		case "c":
			m.modal = NewCreateIndexModal()
			return m, func() tea.Msg { return ModalInitMsg{} }
		case "d":
			if m.SelectedIndex() != "" {
				m.modal = NewDeleteIndexModal(m.SelectedIndex())
				return m, func() tea.Msg { return ModalInitMsg{} }
			}
		case "a":
			if m.SelectedIndex() != "" {
				m.modal = NewAddAliasModal(m.SelectedIndex())
				return m, func() tea.Msg { return ModalInitMsg{} }
			}
		case "A":
			if m.SelectedIndex() != "" && m.cluster != nil {
				aliases := m.cluster.GetAliasesForIndex(m.SelectedIndex())
				m.modal = NewRemoveAliasModal(m.SelectedIndex(), aliases)
				return m, func() tea.Msg { return ModalInitMsg{} }
			}
		case "enter":
			if m.shardInfo != nil {
				m.shardInfo = nil
				m.allocationExplain = nil
				m.recoveryInfo = nil
				return m, nil
			}
			if m.allocationExplain != nil {
				m.allocationExplain = nil
				return m, nil
			}
			return m.handleCellEnter()
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			nodeColWidth := 18
			indexColWidth := 25
			headerRows := 6

			if msg.Y >= headerRows && msg.X > nodeColWidth+2 {
				colOffset := (msg.X - nodeColWidth - 2) / indexColWidth
				clickedIndex := m.scrollX + colOffset
				indices := m.filteredIndices()
				if clickedIndex >= 0 && clickedIndex < len(indices) {
					m.selectedIndex = clickedIndex
				}
			}
		}
	case IndexCreatedMsg:
		m.operationMsg = ""
		if msg.Err != nil {
			m.modal = NewErrorModal(msg.Err.Error())
			return m, func() tea.Msg { return ModalInitMsg{} }
		}
		return m, nil
	case IndexDeletedMsg:
		m.operationMsg = ""
		if msg.Err != nil {
			m.modal = NewErrorModal(msg.Err.Error())
			return m, func() tea.Msg { return ModalInitMsg{} }
		}
		m.selectedIndex = 0
		return m, nil
	case AliasAddedMsg:
		m.operationMsg = ""
		if msg.Err != nil {
			m.modal = NewErrorModal(msg.Err.Error())
			return m, func() tea.Msg { return ModalInitMsg{} }
		}
		return m, nil
	case AliasRemovedMsg:
		m.operationMsg = ""
		if msg.Err != nil {
			m.modal = NewErrorModal(msg.Err.Error())
			return m, func() tea.Msg { return ModalInitMsg{} }
		}
		return m, nil
	case AllocationExplainMsg:
		m.allocationLoading = false
		if m.shardInfo == nil {
			return m, nil
		}
		if msg.Err != nil {
			m.modal = NewErrorModal(msg.Err.Error())
			return m, func() tea.Msg { return ModalInitMsg{} }
		}
		m.allocationExplain = msg.Result
		return m, nil
	case RecoveryMsg:
		if m.shardInfo != nil && m.shardInfo.Index == msg.Index && m.shardInfo.Shard == msg.Shard {
			m.recoveryInfo = msg.Result
		}
		return m, nil
	}
	return m, nil
}

func (m OverviewModel) handleModalDone() (OverviewModel, tea.Cmd) {
	modalType := m.modal.Type()

	switch modalType {
	case ModalCreateIndex:
		name := m.modal.IndexName()
		shards := m.modal.Shards()
		replicas := m.modal.Replicas()
		m.modal = nil
		m.operationMsg = "Creating index..."
		return m, tea.Batch(m.spinner.Tick, m.createIndexCmd(name, shards, replicas))

	case ModalDeleteIndex:
		indexName := m.modal.IndexName()
		m.modal = nil
		m.operationMsg = "Deleting index..."
		return m, tea.Batch(m.spinner.Tick, m.deleteIndexCmd(indexName))

	case ModalAddAlias:
		aliasName := m.modal.AliasName()
		indexName := m.modal.IndexName()
		m.modal = nil
		m.operationMsg = "Adding alias..."
		return m, tea.Batch(m.spinner.Tick, m.addAliasCmd(indexName, aliasName))

	case ModalRemoveAlias:
		if !m.modal.HasAliases() {
			m.modal = nil
			return m, nil
		}
		aliasName := m.modal.AliasName()
		indexName := m.modal.IndexName()
		m.modal = nil
		m.operationMsg = "Removing alias..."
		return m, tea.Batch(m.spinner.Tick, m.removeAliasCmd(indexName, aliasName))

	case ModalError:
		m.modal = nil
		return m, nil
	}

	m.modal = nil
	return m, nil
}

func (m OverviewModel) createIndexCmd(name, shards, replicas string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return IndexCreatedMsg{Err: fmt.Errorf("no client")}
		}
		s, _ := strconv.Atoi(shards)
		r, _ := strconv.Atoi(replicas)
		if s < 1 {
			s = 1
		}
		if r < 0 {
			r = 0
		}
		err := m.client.CreateIndex(context.Background(), name, s, r)
		return IndexCreatedMsg{Err: err}
	}
}

func (m OverviewModel) deleteIndexCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return IndexDeletedMsg{Err: fmt.Errorf("no client")}
		}
		err := m.client.DeleteIndex(context.Background(), name)
		return IndexDeletedMsg{Err: err}
	}
}

func (m OverviewModel) addAliasCmd(index, alias string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return AliasAddedMsg{Err: fmt.Errorf("no client")}
		}
		err := m.client.AddAlias(context.Background(), index, alias)
		return AliasAddedMsg{Err: err}
	}
}

func (m OverviewModel) removeAliasCmd(index, alias string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return AliasRemovedMsg{Err: fmt.Errorf("no client")}
		}
		err := m.client.RemoveAlias(context.Background(), index, alias)
		return AliasRemovedMsg{Err: err}
	}
}

func (m OverviewModel) fetchAllocationExplain(index string, shard int, primary bool) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return AllocationExplainMsg{Err: fmt.Errorf("no client")}
		}
		result, err := m.client.FetchAllocationExplain(context.Background(), index, shard, primary)
		return AllocationExplainMsg{Result: result, Err: err}
	}
}

func (m OverviewModel) fetchRecovery(index string, shard string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return RecoveryMsg{Index: index, Shard: shard}
		}
		recoveries, err := m.client.FetchRecovery(context.Background())
		if err != nil {
			return RecoveryMsg{Index: index, Shard: shard}
		}
		for _, r := range recoveries {
			if r.Index == index && r.Shard == shard {
				return RecoveryMsg{Index: index, Shard: shard, Result: &r}
			}
		}
		return RecoveryMsg{Index: index, Shard: shard}
	}
}

func (m OverviewModel) filteredIndices() []es.IndexInfo {
	if m.cluster == nil {
		return nil
	}

	var filtered []es.IndexInfo
	filterText := strings.ToLower(m.filter.Value())

	for _, idx := range m.cluster.Indices {
		if !m.showSystem && strings.HasPrefix(idx.Name, ".") {
			continue
		}

		if filterText != "" {
			match := false
			if strings.Contains(strings.ToLower(idx.Name), filterText) {
				match = true
			}
			if strings.HasSuffix(filterText, "*") {
				prefix := strings.TrimSuffix(filterText, "*")
				if strings.HasPrefix(strings.ToLower(idx.Name), prefix) {
					match = true
				}
			}
			if !match {
				continue
			}
		}

		if len(m.aliasFilters) > 0 {
			hasActiveAlias := false
			for _, active := range m.aliasFilters {
				if active {
					hasActiveAlias = true
					break
				}
			}
			if hasActiveAlias {
				indexAliases := m.cluster.GetAliasesForIndex(idx.Name)
				match := false
				for _, a := range indexAliases {
					if m.aliasFilters[a] {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
		}

		if m.shardStateFilter != "" {
			if !m.indexHasShardsInState(idx.Name, m.shardStateFilter) {
				continue
			}
		}

		filtered = append(filtered, idx)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	return filtered
}

func (m OverviewModel) indexHasShardsInState(indexName, state string) bool {
	if m.cluster == nil {
		return false
	}
	for _, sh := range m.cluster.Shards {
		if sh.Index == indexName && sh.State == state {
			return true
		}
	}
	return false
}

func (m OverviewModel) getNodeList() []es.NodeInfo {
	if m.cluster == nil {
		return nil
	}
	nodes := m.cluster.Nodes
	hasUnassigned := false
	indices := m.filteredIndices()
	for _, idx := range indices {
		if len(m.cluster.GetUnassignedShardsForIndex(idx.Name)) > 0 {
			hasUnassigned = true
			break
		}
	}
	if hasUnassigned {
		nodes = append(nodes, es.NodeInfo{Name: "Unassigned"})
	}
	return nodes
}

func (m OverviewModel) handleCellEnter() (OverviewModel, tea.Cmd) {
	if m.cluster == nil {
		return m, nil
	}

	indices := m.filteredIndices()
	if m.selectedIndex < 0 || m.selectedIndex >= len(indices) {
		return m, nil
	}
	indexName := indices[m.selectedIndex].Name

	nodes := m.getNodeList()
	if m.selectedNode < 0 || m.selectedNode >= len(nodes) {
		return m, nil
	}
	nodeName := nodes[m.selectedNode].Name

	var shards []es.ShardInfo
	if nodeName == "Unassigned" {
		shards = m.cluster.GetUnassignedShardsForIndex(indexName)
	} else {
		shards = m.cluster.GetShardsForIndexAndNode(indexName, nodeName)
	}

	if len(shards) == 0 {
		return m, nil
	}

	if len(shards) == 1 {
		return m.showShardInfo(&shards[0])
	}

	m.shardPicker = NewShardPicker(shards, m.width, m.height)
	return m, nil
}

func (m OverviewModel) showShardInfo(sh *es.ShardInfo) (OverviewModel, tea.Cmd) {
	m.shardInfo = sh
	m.recoveryInfo = nil
	if sh.State == "UNASSIGNED" || sh.State == "RELOCATING" || sh.State == "INITIALIZING" {
		shardNum, err := strconv.Atoi(sh.Shard)
		if err != nil {
			return m, nil
		}
		m.allocationLoading = true
		cmds := []tea.Cmd{m.fetchAllocationExplain(sh.Index, shardNum, sh.Primary)}
		if sh.State == "RELOCATING" || sh.State == "INITIALIZING" {
			cmds = append(cmds, m.fetchRecovery(sh.Index, sh.Shard))
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m OverviewModel) SelectedNode() string {
	nodes := m.getNodeList()
	if m.selectedNode >= 0 && m.selectedNode < len(nodes) {
		return nodes[m.selectedNode].Name
	}
	return ""
}

func (m OverviewModel) SelectedIndex() string {
	indices := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(indices) {
		return indices[m.selectedIndex].Name
	}
	return ""
}

func (m OverviewModel) HasModal() bool {
	return m.modal != nil || m.allocationExplain != nil || m.shardPicker != nil || m.shardInfo != nil
}

func (m OverviewModel) visibleColumns() int {
	nodeColWidth := 18
	indexColWidth := 25
	separatorWidth := 2
	cols := (m.width - nodeColWidth - separatorWidth) / indexColWidth
	if cols < 1 {
		cols = 1
	}
	return cols
}

func (m OverviewModel) maxVisibleNodes() int {
	maxLinesPerNode := 4
	rows := (m.height - 9) / maxLinesPerNode
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m OverviewModel) View() string {
	if m.cluster == nil {
		return "Loading cluster state..."
	}

	if m.shardInfo != nil {
		return RenderShardInfoModal(m.shardInfo, m.allocationExplain, m.recoveryInfo, m.width, m.height)
	}

	if m.allocationExplain != nil {
		return m.renderAllocationExplainModal()
	}

	var b strings.Builder

	// Filter bar
	filterStyle := lipgloss.NewStyle().Padding(0, 1)
	if m.filterActive {
		b.WriteString(filterStyle.Render("Filter: " + m.filter.View()))
	} else if m.filter.Value() != "" {
		b.WriteString(filterStyle.Render("Filter: " + m.filter.Value() + " (/ to edit, Esc to clear)"))
	} else {
		b.WriteString(filterStyle.Render("/ to filter"))
	}

	// Shard state filter indicator
	if m.shardStateFilter != "" {
		stateStyle := lipgloss.NewStyle().Padding(0, 1).Background(ColorYellow).Foreground(ColorOnAccent)
		b.WriteString(stateStyle.Render("Showing: " + m.shardStateFilter + " (Esc to clear)"))
	}

	// Alias toggles
	aliases := m.cluster.UniqueAliases()
	if len(aliases) > 0 {
		b.WriteString("  Aliases: ")
		for i, alias := range aliases {
			if i >= 9 {
				break
			}
			style := lipgloss.NewStyle().Padding(0, 1)
			if m.aliasFilters[alias] {
				style = style.Background(ColorBlue).Foreground(ColorOnAccent)
			} else {
				style = style.Foreground(ColorGray)
			}
			b.WriteString(style.Render(string(rune('1'+i)) + ":" + alias))
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")

	// Scroll indicators
	indices := m.filteredIndices()
	if len(indices) > 0 {
		visibleCols := m.visibleColumns()
		leftCount := m.scrollX
		rightCount := len(indices) - m.scrollX - visibleCols
		if rightCount < 0 {
			rightCount = 0
		}

		indicatorStyle := lipgloss.NewStyle().Foreground(ColorGray)
		if leftCount > 0 || rightCount > 0 {
			var indicator string
			if leftCount > 0 {
				indicator += indicatorStyle.Render(fmt.Sprintf("<< %d more  ", leftCount))
			}
			if rightCount > 0 {
				if leftCount > 0 {
					indicator += indicatorStyle.Render(fmt.Sprintf(" | %d more >>", rightCount))
				} else {
					indicator += indicatorStyle.Render(fmt.Sprintf("%d more >>", rightCount))
				}
			}
			b.WriteString(indicator)
		}
	}
	b.WriteString("\n")

	// Shard grid
	b.WriteString(m.renderGrid())

	// Shard color legend
	b.WriteString("\n\n")
	b.WriteString(m.renderShardLegend())

	if m.modal != nil {
		return m.modal.View(m.width, m.height)
	}

	if m.operationMsg != "" {
		content := m.spinner.View() + " " + m.operationMsg
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBlue).
			Padding(1, 3)
		box := boxStyle.Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	background := b.String()

	if m.shardPicker != nil {
		return OverlayModal(background, m.shardPicker.View(), m.width, m.height)
	}

	return background
}

func (m OverviewModel) renderGrid() string {
	if m.cluster == nil || len(m.cluster.Nodes) == 0 {
		return "No nodes found"
	}

	indices := m.filteredIndices()
	if len(indices) == 0 {
		return "No indices match filter"
	}

	nodes := m.cluster.Nodes

	nodeColWidth := 18
	indexColWidth := 25
	visibleCols := m.visibleColumns()
	actualVisibleCols := visibleCols
	if actualVisibleCols > len(indices)-m.scrollX {
		actualVisibleCols = len(indices) - m.scrollX
	}
	if actualVisibleCols < 0 {
		actualVisibleCols = 0
	}
	contentWidth := nodeColWidth + 2 + actualVisibleCols*indexColWidth

	var b strings.Builder

	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+visibleCols {
			nameStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(HealthColor(idx.Health)).
				Bold(true)
			if i == m.selectedIndex {
				nameStyle = nameStyle.Reverse(true)
			}
			b.WriteString(nameStyle.Render(Truncate(idx.Name, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+visibleCols {
			sizeStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorGray)
			priSize := idx.PriStoreSize
			if priSize == "" {
				priSize = idx.StoreSize
			}
			sizeText := priSize + "/" + idx.StoreSize
			b.WriteString(sizeStyle.Render(Truncate(sizeText, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+visibleCols {
			docsStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorGray)
			docsText := FormatNumber(idx.DocsCount) + " docs"
			b.WriteString(docsStyle.Render(Truncate(docsText, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+visibleCols {
			aliases := m.cluster.GetAliasesForIndex(idx.Name)
			aliasStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorBlue)
			aliasText := ""
			if len(aliases) > 0 {
				aliasText = "[" + strings.Join(aliases, ",") + "]"
			}
			b.WriteString(aliasStyle.Render(Truncate(aliasText, indexColWidth-2)))
		}
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", contentWidth) + "\n")

	// Node rows
	visibleNodes := nodes
	if m.scrollY < len(nodes) {
		visibleNodes = nodes[m.scrollY:]
	}
	maxRows := m.maxVisibleNodes()
	if maxRows > len(visibleNodes) {
		maxRows = len(visibleNodes)
	}

	nodeStyle := lipgloss.NewStyle().Width(nodeColWidth)
	selectedNodeStyle := lipgloss.NewStyle().Width(nodeColWidth).Background(ColorBlue).Foreground(ColorOnAccent)
	emptyCol := lipgloss.NewStyle().Width(indexColWidth).Render("")

	maxLinesPerNode := 4

	for rowIdx, node := range visibleNodes[:maxRows] {
		actualNodeIdx := m.scrollY + rowIdx
		isSelectedNode := actualNodeIdx == m.selectedNode

		var shardLines [][]string
		maxLines := 2

		for i, idx := range indices {
			if i >= m.scrollX && i < m.scrollX+visibleCols {
				shards := m.cluster.GetShardsForIndexAndNode(idx.Name, node.Name)
				isSelectedCell := isSelectedNode && i == m.selectedIndex
				lines := m.renderShardBoxesWithHighlight(shards, indexColWidth, isSelectedCell)
				shardLines = append(shardLines, lines)
				if len(lines) > maxLines {
					maxLines = len(lines)
				}
			}
		}

		if maxLines > maxLinesPerNode {
			maxLines = maxLinesPerNode
		}

		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			if lineIdx == 0 {
				style := nodeStyle
				if isSelectedNode {
					style = selectedNodeStyle
				}
				b.WriteString(style.Render(Truncate(node.Name, nodeColWidth-2)))
				b.WriteString("│ ")
			} else if lineIdx == 1 {
				versionStyle := lipgloss.NewStyle().Width(nodeColWidth).Foreground(ColorGray)
				if isSelectedNode {
					versionStyle = versionStyle.Background(ColorBlue).Foreground(ColorOnAccent)
				}
				b.WriteString(versionStyle.Render(node.Version))
				b.WriteString("│ ")
			} else {
				style := nodeStyle
				if isSelectedNode {
					style = selectedNodeStyle
				}
				b.WriteString(style.Render(""))
				b.WriteString("│ ")
			}

			for colIdx := range shardLines {
				if lineIdx < len(shardLines[colIdx]) {
					b.WriteString(shardLines[colIdx][lineIdx])
				} else {
					b.WriteString(emptyCol)
				}
			}
			b.WriteString("\n")
		}
	}

	// Unassigned shards row
	hasUnassigned := false
	for _, idx := range indices {
		if len(m.cluster.GetUnassignedShardsForIndex(idx.Name)) > 0 {
			hasUnassigned = true
			break
		}
	}

	if hasUnassigned {
		b.WriteString(strings.Repeat("─", contentWidth) + "\n")

		unassignedNodeIdx := len(nodes)
		isSelectedUnassigned := m.selectedNode == unassignedNodeIdx

		var shardLines [][]string
		maxLines := 1

		for i, idx := range indices {
			if i >= m.scrollX && i < m.scrollX+visibleCols {
				shards := m.cluster.GetUnassignedShardsForIndex(idx.Name)
				isSelectedCell := isSelectedUnassigned && i == m.selectedIndex
				lines := m.renderShardBoxesWithHighlight(shards, indexColWidth, isSelectedCell)
				shardLines = append(shardLines, lines)
				if len(lines) > maxLines {
					maxLines = len(lines)
				}
			}
		}

		unassignedStyle := lipgloss.NewStyle().Width(nodeColWidth).Foreground(ColorRed)
		if isSelectedUnassigned {
			unassignedStyle = unassignedStyle.Background(ColorBlue).Foreground(ColorOnAccent)
		}
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			if lineIdx == 0 {
				b.WriteString(unassignedStyle.Render("Unassigned"))
				b.WriteString("│ ")
			} else {
				style := nodeStyle
				if isSelectedUnassigned {
					style = selectedNodeStyle
				}
				b.WriteString(style.Render(""))
				b.WriteString("│ ")
			}

			for colIdx := range shardLines {
				if lineIdx < len(shardLines[colIdx]) {
					b.WriteString(shardLines[colIdx][lineIdx])
				} else {
					b.WriteString(emptyCol)
				}
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m OverviewModel) renderShardLegend() string {
	grayStyle := lipgloss.NewStyle().Foreground(ColorGray)

	var countText string
	if m.cluster != nil && m.cluster.Health != nil {
		h := m.cluster.Health
		countText = lipgloss.NewStyle().Foreground(HealthColor(h.Status)).Render(fmt.Sprintf("Primary shards: %d", h.ActivePrimaryShards))
	} else {
		countText = grayStyle.Render("Primary shards: -")
	}

	greenBadge := lipgloss.NewStyle().
		Background(ColorGreen).
		Foreground(ColorOnAccent).
		Padding(0, 1).
		Render("P")
	blueBadge := lipgloss.NewStyle().
		Background(ColorBlue).
		Foreground(ColorOnAccent).
		Padding(0, 1).
		Render("R")
	purpleBadge := lipgloss.NewStyle().
		Background(ColorPurple).
		Foreground(ColorOnAccent).
		Padding(0, 1).
		Render("R")
	yellowBadge := lipgloss.NewStyle().
		Background(ColorYellow).
		Foreground(lipgloss.Color("#000000")).
		Padding(0, 1).
		Render("I")
	redBadge := lipgloss.NewStyle().
		Background(ColorRed).
		Foreground(ColorOnAccent).
		Padding(0, 1).
		Render("U")

	return countText +
		grayStyle.Render("  ") +
		greenBadge + grayStyle.Render(" Primary") +
		grayStyle.Render(" | ") +
		blueBadge + grayStyle.Render(" Replica") +
		grayStyle.Render(" | ") +
		purpleBadge + grayStyle.Render(" Relocating") +
		grayStyle.Render(" | ") +
		yellowBadge + grayStyle.Render(" Initializing") +
		grayStyle.Render(" | ") +
		redBadge + grayStyle.Render(" Unassigned")
}

func (m OverviewModel) renderShardBoxesWithHighlight(shards []es.ShardInfo, width int, highlight bool) []string {
	highlightBg := lipgloss.Color("#3a3a5a")

	if len(shards) == 0 {
		if highlight {
			return []string{lipgloss.NewStyle().Width(width).Background(highlightBg).Render("")}
		}
		return []string{lipgloss.NewStyle().Width(width).Render("")}
	}

	var lines []string
	var currentLine []string
	currentWidth := 0

	for _, sh := range shards {
		var bgColor, fgColor lipgloss.Color
		if sh.Primary {
			bgColor = ColorGreen
		} else {
			bgColor = ColorBlue
		}
		fgColor = ColorOnAccent

		switch sh.State {
		case "RELOCATING":
			bgColor = ColorPurple
		case "INITIALIZING":
			bgColor = ColorYellow
			fgColor = lipgloss.Color("#000000")
		case "UNASSIGNED":
			bgColor = ColorRed
		}

		style := lipgloss.NewStyle().
			Foreground(fgColor).
			Background(bgColor).
			Bold(true).
			Width(4).
			Align(lipgloss.Center).
			MarginRight(1)

		if highlight {
			style = style.Reverse(true)
		}

		styledShard := style.Render(sh.Shard)
		shardWidth := lipgloss.Width(styledShard)

		if currentWidth+shardWidth > width && len(currentLine) > 0 {
			lineStyle := lipgloss.NewStyle().Width(width)
			if highlight {
				lineStyle = lineStyle.Background(highlightBg)
			}
			lines = append(lines, lineStyle.Render(strings.Join(currentLine, "")))
			currentLine = nil
			currentWidth = 0
		}

		currentLine = append(currentLine, styledShard)
		currentWidth += shardWidth
	}

	if len(currentLine) > 0 {
		lineStyle := lipgloss.NewStyle().Width(width)
		if highlight {
			lineStyle = lineStyle.Background(highlightBg)
		}
		lines = append(lines, lineStyle.Render(strings.Join(currentLine, "")))
	}

	if len(lines) == 0 {
		if highlight {
			lines = []string{lipgloss.NewStyle().Width(width).Background(highlightBg).Render("")}
		} else {
			lines = []string{lipgloss.NewStyle().Width(width).Render("")}
		}
	}

	return lines
}

func (m OverviewModel) renderAllocationExplainModal() string {
	ae := m.allocationExplain

	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	shardType := "replica"
	if ae.Primary {
		shardType = "primary"
	}

	content := strings.Join([]string{
		labelStyle.Render("Index:    ") + valueStyle.Render(ae.Index),
		labelStyle.Render("Shard:    ") + valueStyle.Render(fmt.Sprintf("%d (%s)", ae.Shard, shardType)),
		labelStyle.Render("State:    ") + valueStyle.Render(ae.CurrentState),
		"",
		labelStyle.Render("Reason:   ") + valueStyle.Render(ae.UnassignedReason),
		labelStyle.Render("Status:   ") + valueStyle.Render(ae.AllocationStatus),
		"",
		labelStyle.Render("Details:"),
		valueStyle.Render(ae.ExplanationDetail),
	}, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(60)

	box := boxStyle.Render(content)
	footer := lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter or Esc to close")

	modal := lipgloss.JoinVertical(lipgloss.Center, box, footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

