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

type OverviewModel struct {
	cluster          *es.ClusterState
	client           *es.Client
	filter           textinput.Model
	filterActive     bool
	aliasFilters     map[string]bool
	shardStateFilter string
	showSystem       bool
	scrollX          int
	scrollY          int
	selectedIndex    int
	width            int
	height           int
	modal            *Modal
	spinner          spinner.Model
	operationMsg     string
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
		case ".":
			m.showSystem = !m.showSystem
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
		case "pgup":
			pageSize := m.maxVisibleNodes()
			m.scrollY -= pageSize
			if m.scrollY < 0 {
				m.scrollY = 0
			}
		case "pgdown":
			pageSize := m.maxVisibleNodes()
			maxScrollY := m.maxScrollY()
			m.scrollY += pageSize
			if m.scrollY > maxScrollY {
				m.scrollY = maxScrollY
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

func (m OverviewModel) SelectedIndex() string {
	indices := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(indices) {
		return indices[m.selectedIndex].Name
	}
	return ""
}

func (m OverviewModel) HasModal() bool {
	return m.modal != nil
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
	emptyCol := lipgloss.NewStyle().Width(indexColWidth).Render("")

	maxLinesPerNode := 4

	for _, node := range visibleNodes[:maxRows] {
		var shardLines [][]string
		maxLines := 2

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
				b.WriteString(nodeStyle.Render(Truncate(node.Name, nodeColWidth-2)))
				b.WriteString("│ ")
			} else if lineIdx == 1 {
				versionStyle := lipgloss.NewStyle().Width(nodeColWidth).Foreground(ColorGray)
				b.WriteString(versionStyle.Render(node.Version))
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

func (m OverviewModel) renderShardLegend() string {
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
	yellowBadge := lipgloss.NewStyle().
		Background(ColorYellow).
		Foreground(lipgloss.Color("#000000")).
		Padding(0, 1).
		Render("R")
	redBadge := lipgloss.NewStyle().
		Background(ColorRed).
		Foreground(ColorOnAccent).
		Padding(0, 1).
		Render("U")

	grayStyle := lipgloss.NewStyle().Foreground(ColorGray)

	return grayStyle.Render("Shards: ") +
		greenBadge + grayStyle.Render(" Primary") +
		grayStyle.Render(" | ") +
		blueBadge + grayStyle.Render(" Replica") +
		grayStyle.Render(" | ") +
		yellowBadge + grayStyle.Render(" Relocating/Initializing") +
		grayStyle.Render(" | ") +
		redBadge + grayStyle.Render(" Unassigned")
}

func (m OverviewModel) renderShardBoxes(shards []es.ShardInfo, width int) []string {
	if len(shards) == 0 {
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
			bgColor = ColorYellow
			fgColor = lipgloss.Color("#000000")
		case "UNASSIGNED":
			bgColor = ColorRed
		case "INITIALIZING":
			bgColor = ColorYellow
			fgColor = lipgloss.Color("#000000")
		}

		styledShard := lipgloss.NewStyle().
			Foreground(fgColor).
			Background(bgColor).
			Bold(true).
			Width(4).
			Align(lipgloss.Center).
			MarginRight(1).
			Render(sh.Shard)
		shardWidth := lipgloss.Width(styledShard)

		if currentWidth+shardWidth > width && len(currentLine) > 0 {
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(strings.Join(currentLine, "")))
			currentLine = nil
			currentWidth = 0
		}

		currentLine = append(currentLine, styledShard)
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

