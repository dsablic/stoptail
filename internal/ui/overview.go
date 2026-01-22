package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type IndexCreatedMsg struct{ Err error }
type IndexDeletedMsg struct{ Err error }
type AliasAddedMsg struct{ Err error }
type AliasRemovedMsg struct{ Err error }

type OverviewModel struct {
	cluster          *es.ClusterState
	client           *es.Client
	filter           textinput.Model
	filterActive     bool
	aliasFilters     map[string]bool
	shardStateFilter string
	scrollX          int
	scrollY          int
	selectedIndex    int
	width            int
	height           int
	modal            *Modal
	modalAction      string
	modalStep        int
	createName       string
	createShards     string
}

func NewOverview() OverviewModel {
	ti := textinput.New()
	ti.Placeholder = "Filter indices..."
	ti.CharLimit = 50

	return OverviewModel{
		filter:       ti,
		aliasFilters: make(map[string]bool),
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

	if m.modal != nil {
		cmd := m.modal.Update(msg)
		if m.modal.Cancelled() || (m.modal.Done() && m.modalAction == "error") {
			m.modal = nil
			m.modalAction = ""
			m.modalStep = 0
			return m, nil
		}
		if m.modal.Done() {
			return m.handleModalDone()
		}
		return m, cmd
	}

	switch msg := msg.(type) {
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

		switch msg.String() {
		case "/":
			m.filterActive = true
			m.filter.Focus()
			return m, textinput.Blink
		case "esc":
			m.filter.SetValue("")
			m.aliasFilters = make(map[string]bool)
			m.shardStateFilter = ""
			m.selectedIndex = 0
		case "U":
			if m.shardStateFilter == "UNASSIGNED" {
				m.shardStateFilter = ""
			} else {
				m.shardStateFilter = "UNASSIGNED"
			}
			m.selectedIndex = 0
			m.scrollX = 0
		case "R":
			if m.shardStateFilter == "RELOCATING" {
				m.shardStateFilter = ""
			} else {
				m.shardStateFilter = "RELOCATING"
			}
			m.selectedIndex = 0
			m.scrollX = 0
		case "I":
			if m.shardStateFilter == "INITIALIZING" {
				m.shardStateFilter = ""
			} else {
				m.shardStateFilter = "INITIALIZING"
			}
			m.selectedIndex = 0
			m.scrollX = 0
		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			maxScrollY := m.maxScrollY()
			if m.scrollY < maxScrollY {
				m.scrollY++
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
					m.scrollX = 0
				}
			}
		case "c":
			m.modal = NewModal("Create Index", "Index name:")
			m.modalAction = "create"
			m.modalStep = 1
			return m, textinput.Blink
		case "d":
			if m.SelectedIndex() != "" {
				m.modal = NewModal("Delete Index", "Type '"+m.SelectedIndex()+"' to confirm:")
				m.modalAction = "delete"
				return m, textinput.Blink
			}
		case "a":
			if m.SelectedIndex() != "" {
				m.modal = NewModal("Add Alias", "Alias name:")
				m.modalAction = "addAlias"
				return m, textinput.Blink
			}
		case "A":
			if m.SelectedIndex() != "" {
				m.modal = NewModal("Remove Alias", "Alias name:")
				m.modalAction = "removeAlias"
				return m, textinput.Blink
			}
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
		if msg.Err != nil {
			m.modal = NewModal("Error", msg.Err.Error())
			m.modalAction = "error"
		}
		return m, nil
	case IndexDeletedMsg:
		if msg.Err != nil {
			m.modal = NewModal("Error", msg.Err.Error())
			m.modalAction = "error"
		}
		m.selectedIndex = 0
		return m, nil
	case AliasAddedMsg:
		if msg.Err != nil {
			m.modal = NewModal("Error", msg.Err.Error())
			m.modalAction = "error"
		}
		return m, nil
	case AliasRemovedMsg:
		if msg.Err != nil {
			m.modal = NewModal("Error", msg.Err.Error())
			m.modalAction = "error"
		}
		return m, nil
	}
	return m, nil
}

func (m OverviewModel) handleModalDone() (OverviewModel, tea.Cmd) {
	value := m.modal.Value()

	switch m.modalAction {
	case "create":
		switch m.modalStep {
		case 1:
			if value == "" {
				m.modal.SetError("Index name required")
				m.modal.SetDone(false)
				return m, nil
			}
			m.createName = value
			m.modal.Reset("Create Index", "Number of shards (default 1):")
			m.modalStep = 2
			return m, textinput.Blink
		case 2:
			m.createShards = value
			if m.createShards == "" {
				m.createShards = "1"
			}
			m.modal.Reset("Create Index", "Number of replicas (default 1):")
			m.modalStep = 3
			return m, textinput.Blink
		case 3:
			replicas := value
			if replicas == "" {
				replicas = "1"
			}
			m.modal = nil
			m.modalAction = ""
			m.modalStep = 0
			return m, m.createIndexCmd(m.createName, m.createShards, replicas)
		}

	case "delete":
		if value != m.SelectedIndex() {
			m.modal.SetError("Name does not match")
			m.modal.SetDone(false)
			return m, nil
		}
		indexName := m.SelectedIndex()
		m.modal = nil
		m.modalAction = ""
		return m, m.deleteIndexCmd(indexName)

	case "addAlias":
		if value == "" {
			m.modal.SetError("Alias name required")
			m.modal.SetDone(false)
			return m, nil
		}
		indexName := m.SelectedIndex()
		m.modal = nil
		m.modalAction = ""
		return m, m.addAliasCmd(indexName, value)

	case "removeAlias":
		if value == "" {
			m.modal.SetError("Alias name required")
			m.modal.SetDone(false)
			return m, nil
		}
		indexName := m.SelectedIndex()
		m.modal = nil
		m.modalAction = ""
		return m, m.removeAliasCmd(indexName, value)
	}

	m.modal = nil
	m.modalAction = ""
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

func (m OverviewModel) filteredIndices() []es.IndexInfo {
	if m.cluster == nil {
		return nil
	}

	var filtered []es.IndexInfo
	filterText := strings.ToLower(m.filter.Value())

	for _, idx := range m.cluster.Indices {
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

func (m OverviewModel) SelectedIndex() string {
	indices := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(indices) {
		return indices[m.selectedIndex].Name
	}
	return ""
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

func (m OverviewModel) maxScrollY() int {
	if m.cluster == nil {
		return 0
	}
	maxRows := m.maxVisibleNodes()
	maxScroll := len(m.cluster.Nodes) - maxRows
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m OverviewModel) maxVisibleNodes() int {
	maxLinesPerNode := 4
	rows := (m.height - 8) / maxLinesPerNode
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m OverviewModel) View() string {
	if m.cluster == nil {
		return "Loading cluster state..."
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

	return b.String()
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
			healthColor := ColorGreen
			switch idx.Health {
			case "yellow":
				healthColor = ColorYellow
			case "red":
				healthColor = ColorRed
			}
			nameStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(healthColor).
				Bold(true)
			if i == m.selectedIndex {
				nameStyle = nameStyle.Reverse(true)
			}
			b.WriteString(nameStyle.Render(truncate(idx.Name, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+visibleCols {
			statsStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorGray)
			stats := idx.StoreSize + " · " + idx.DocsCount
			b.WriteString(statsStyle.Render(truncate(stats, indexColWidth-2)))
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
			b.WriteString(aliasStyle.Render(truncate(aliasText, indexColWidth-2)))
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
	emptyCol := lipgloss.NewStyle().Width(indexColWidth).Render("")

	maxLinesPerNode := 4

	for _, node := range visibleNodes[:maxRows] {
		var shardLines [][]string
		maxLines := 1

		for i, idx := range indices {
			if i >= m.scrollX && i < m.scrollX+visibleCols {
				shards := m.cluster.GetShardsForIndexAndNode(idx.Name, node.Name)
				lines := m.renderShardBoxes(shards, indexColWidth)
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
				b.WriteString(nodeStyle.Render(truncate(node.Name, nodeColWidth-2)))
				b.WriteString("│ ")
			} else {
				b.WriteString(nodeStyle.Render(""))
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

		var shardLines [][]string
		maxLines := 1

		for i, idx := range indices {
			if i >= m.scrollX && i < m.scrollX+visibleCols {
				shards := m.cluster.GetUnassignedShardsForIndex(idx.Name)
				lines := m.renderShardBoxes(shards, indexColWidth)
				shardLines = append(shardLines, lines)
				if len(lines) > maxLines {
					maxLines = len(lines)
				}
			}
		}

		unassignedStyle := lipgloss.NewStyle().Width(nodeColWidth).Foreground(ColorRed)
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			if lineIdx == 0 {
				b.WriteString(unassignedStyle.Render("Unassigned"))
				b.WriteString("│ ")
			} else {
				b.WriteString(nodeStyle.Render(""))
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

func (m OverviewModel) renderShardBoxes(shards []es.ShardInfo, width int) []string {
	if len(shards) == 0 {
		return []string{lipgloss.NewStyle().Width(width).Render("")}
	}

	var lines []string
	var currentLine []string
	currentWidth := 0

	for _, sh := range shards {
		var color lipgloss.Color
		if sh.Primary {
			color = ColorGreen
		} else {
			color = ColorBlue
		}

		switch sh.State {
		case "RELOCATING":
			color = ColorYellow
		case "UNASSIGNED":
			color = ColorRed
		}

		shardText := "[" + sh.Shard + "]"
		shardWidth := len(shardText)

		if currentWidth+shardWidth > width && len(currentLine) > 0 {
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(strings.Join(currentLine, "")))
			currentLine = nil
			currentWidth = 0
		}

		style := lipgloss.NewStyle().Foreground(color)
		currentLine = append(currentLine, style.Render(shardText))
		currentWidth += shardWidth
	}

	if len(currentLine) > 0 {
		lines = append(lines, lipgloss.NewStyle().Width(width).Render(strings.Join(currentLine, "")))
	}

	if len(lines) == 0 {
		lines = []string{lipgloss.NewStyle().Width(width).Render("")}
	}

	return lines
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
