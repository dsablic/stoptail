package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type OverviewModel struct {
	cluster       *es.ClusterState
	filter        textinput.Model
	filterActive  bool
	aliasFilters  map[string]bool
	scrollX       int
	scrollY       int
	selectedIndex int
	width         int
	height        int
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

func (m OverviewModel) Update(msg tea.Msg) (OverviewModel, tea.Cmd) {
	var cmd tea.Cmd

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
		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			m.scrollY++
		case "left", "h":
			if m.scrollX > 0 {
				m.scrollX--
			}
		case "right", "l":
			m.scrollX++
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.cluster != nil {
				aliases := m.cluster.UniqueAliases()
				idx := int(msg.String()[0] - '1')
				if idx < len(aliases) {
					alias := aliases[idx]
					m.aliasFilters[alias] = !m.aliasFilters[alias]
				}
			}
		}
	}
	return m, nil
}

func (m OverviewModel) filteredIndices() []es.IndexInfo {
	if m.cluster == nil {
		return nil
	}

	var filtered []es.IndexInfo
	filterText := strings.ToLower(m.filter.Value())

	for _, idx := range m.cluster.Indices {
		// Text filter
		if filterText != "" {
			match := false
			if strings.Contains(strings.ToLower(idx.Name), filterText) {
				match = true
			}
			// Wildcard support
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

		// Alias filter
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

		filtered = append(filtered, idx)
	}
	return filtered
}

func (m OverviewModel) SelectedIndex() string {
	indices := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(indices) {
		return indices[m.selectedIndex].Name
	}
	return ""
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
				style = style.Background(ColorBlue).Foreground(ColorWhite)
			} else {
				style = style.Foreground(ColorGray)
			}
			b.WriteString(style.Render(string(rune('1'+i)) + ":" + alias))
			b.WriteString(" ")
		}
	}
	b.WriteString("\n\n")

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

	// Calculate column widths
	nodeColWidth := 15
	indexColWidth := 20

	var b strings.Builder

	// Header row - index names
	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
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
			b.WriteString(nameStyle.Render(truncate(idx.Name, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	// Header row - stats
	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
			statsStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorGray)
			stats := idx.StoreSize + " · " + idx.DocsCount
			b.WriteString(statsStyle.Render(truncate(stats, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	// Header row - aliases
	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
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
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	// Node rows
	visibleNodes := nodes
	if m.scrollY < len(nodes) {
		visibleNodes = nodes[m.scrollY:]
	}
	maxRows := (m.height - 8) / 2
	if maxRows > len(visibleNodes) {
		maxRows = len(visibleNodes)
	}

	for _, node := range visibleNodes[:maxRows] {
		var shardLines [][]string
		maxLines := 1

		for i, idx := range indices {
			if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
				shards := m.cluster.GetShardsForIndexAndNode(idx.Name, node.Name)
				lines := m.renderShardBoxes(shards, indexColWidth)
				shardLines = append(shardLines, lines)
				if len(lines) > maxLines {
					maxLines = len(lines)
				}
			}
		}

		nodeStyle := lipgloss.NewStyle().Width(nodeColWidth)
		emptyCol := lipgloss.NewStyle().Width(indexColWidth).Render("")

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

	return b.String()
}

func (m OverviewModel) renderShardBoxes(shards []es.ShardInfo, width int) []string {
	if len(shards) == 0 {
		return []string{lipgloss.NewStyle().Width(width).Render("")}
	}

	shardsPerLine := width / 3
	if shardsPerLine < 1 {
		shardsPerLine = 1
	}

	var lines []string
	var currentLine []string

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

		style := lipgloss.NewStyle().Foreground(color)
		currentLine = append(currentLine, style.Render("["+sh.Shard+"]"))

		if len(currentLine) >= shardsPerLine {
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(strings.Join(currentLine, "")))
			currentLine = nil
		}
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
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
