package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/labtiva/stoptail/internal/es"
)

type NodesView int

const (
	ViewMemory NodesView = iota
	ViewDisk
	ViewFielddata
)

type NodesModel struct {
	state      *es.NodesState
	activeView NodesView
	scrollY    int
	width      int
	height     int
	loading    bool
	search     SearchBar
}

func NewNodes() NodesModel {
	return NodesModel{
		activeView: ViewMemory,
		loading:    true,
		search:     NewSearchBar(),
	}
}

func (m *NodesModel) SetState(state *es.NodesState) {
	m.state = state
	m.loading = state == nil
	m.scrollY = 0
}

func (m *NodesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *NodesModel) SetView(view string) {
	switch view {
	case "memory":
		m.activeView = ViewMemory
	case "disk":
		m.activeView = ViewDisk
	case "fielddata":
		m.activeView = ViewFielddata
	}
}

func (m *NodesModel) selectView(view NodesView) {
	m.activeView = view
	m.scrollY = 0
}

func (m NodesModel) getMaxScroll() int {
	total := m.getItemCount()
	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 10
	}
	maxScroll := total - maxVisible
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m NodesModel) Update(msg tea.Msg) (NodesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.search.Active() {
			cmd, action := m.search.HandleKey(msg)
			switch action {
			case SearchActionClose:
				// search deactivated
			case SearchActionNext, SearchActionPrev:
				if match := m.search.CurrentMatch(); match >= 0 {
					m.scrollY = match
				}
			case SearchActionNone:
				(&m).updateNodeSearch()
			}
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+f":
			m.search.Activate()
			return m, nil
		case "1":
			m.selectView(ViewMemory)
		case "2":
			m.selectView(ViewDisk)
		case "3":
			m.selectView(ViewFielddata)
		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			if m.scrollY < m.getMaxScroll() {
				m.scrollY++
			}
		case "pgup":
			pageSize := m.height - 8
			if pageSize < 1 {
				pageSize = 10
			}
			m.scrollY -= pageSize
			if m.scrollY < 0 {
				m.scrollY = 0
			}
		case "pgdown":
			pageSize := m.height - 8
			if pageSize < 1 {
				pageSize = 10
			}
			m.scrollY += pageSize
			if m.scrollY > m.getMaxScroll() {
				m.scrollY = m.getMaxScroll()
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if msg.Y == 0 {
				tabs := []struct {
					label string
					view  NodesView
				}{
					{"[1:Memory]", ViewMemory},
					{"[2:Disk]", ViewDisk},
					{"[3:Fielddata]", ViewFielddata},
				}

				pos := 0
				for _, tab := range tabs {
					tabWidth := lipgloss.Width(InactiveTabStyle.Render(tab.label))
					if msg.X >= pos && msg.X < pos+tabWidth {
						m.selectView(tab.view)
						break
					}
					pos += tabWidth
				}
			}
		}

		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			scrollAmount := 3
			if msg.Button == tea.MouseButtonWheelUp {
				m.scrollY = max(0, m.scrollY-scrollAmount)
			} else {
				m.scrollY = min(m.getMaxScroll(), m.scrollY+scrollAmount)
			}
		}
	}
	return m, nil
}

func (m NodesModel) View() string {
	if m.loading || m.state == nil {
		return "Loading..."
	}

	var b strings.Builder

	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	switch m.activeView {
	case ViewMemory:
		b.WriteString(m.renderMemoryTable())
	case ViewDisk:
		b.WriteString(m.renderDiskTable())
	case ViewFielddata:
		b.WriteString(m.renderFielddataTable())
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderLegend())

	content := b.String()
	if m.search.Active() {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.search.View(m.width-4))
	}

	return content
}

func (m NodesModel) renderTabs() string {
	tabs := []struct {
		key   string
		label string
		view  NodesView
	}{
		{"1", "Memory", ViewMemory},
		{"2", "Disk", ViewDisk},
		{"3", "Fielddata", ViewFielddata},
	}

	var parts []string
	for _, tab := range tabs {
		label := fmt.Sprintf("[%s:%s]", tab.key, tab.label)
		if m.activeView == tab.view {
			parts = append(parts, ActiveTabStyle.Render(label))
		} else {
			parts = append(parts, InactiveTabStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m NodesModel) renderMemoryTable() string {
	if len(m.state.Nodes) == 0 {
		return "No nodes found"
	}

	var rows [][]string
	var pctValues []string
	visibleNodes := m.visibleItems(len(m.state.Nodes))
	for _, node := range m.state.Nodes[visibleNodes.start:visibleNodes.end] {
		heapPct := m.parsePercent(node.HeapPercent)
		pctValues = append(pctValues, node.HeapPercent)

		rows = append(rows, []string{
			node.Name,
			fmt.Sprintf("%s %s", node.HeapPercent, RenderBar(heapPct, 10)),
			node.HeapCurrent,
			node.FielddataSize,
			node.QueryCacheSize,
			node.SegmentsCount,
		})
	}

	t := table.New().
		Headers("node", "heap%", "heap", "fielddata", "query_cache", "segments").
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle()
			if col >= 2 {
				base = base.Align(lipgloss.Right)
			} else if col == 1 {
				base = base.Align(lipgloss.Center)
			}
			if row == table.HeaderRow {
				return base.Bold(true).Foreground(ColorWhite)
			}
			if col == 1 && row >= 0 && row < len(pctValues) {
				return base.Inherit(m.percentStyle(pctValues[row]))
			}
			return base
		})

	return t.Render()
}

func (m NodesModel) renderDiskTable() string {
	if len(m.state.Nodes) == 0 {
		return "No nodes found"
	}

	var rows [][]string
	var pctValues []string
	visibleNodes := m.visibleItems(len(m.state.Nodes))
	for _, node := range m.state.Nodes[visibleNodes.start:visibleNodes.end] {
		diskPct := m.parsePercent(node.DiskPercent)
		pctValues = append(pctValues, node.DiskPercent)

		rows = append(rows, []string{
			node.Name,
			node.Version,
			fmt.Sprintf("%s %s", node.DiskPercent, RenderBar(diskPct, 10)),
			node.DiskAvail,
			node.DiskTotal,
			node.DiskUsed,
			node.Shards,
		})
	}

	t := table.New().
		Headers("node", "version", "disk%", "disk.avail", "disk.total", "disk.used", "shards").
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle()
			if col >= 3 {
				base = base.Align(lipgloss.Right)
			} else if col == 2 {
				base = base.Align(lipgloss.Center)
			}
			if row == table.HeaderRow {
				return base.Bold(true).Foreground(ColorWhite)
			}
			if col == 2 && row >= 0 && row < len(pctValues) {
				return base.Inherit(m.percentStyle(pctValues[row]))
			}
			return base
		})

	return t.Render()
}

type fielddataByIndexField struct {
	Index string
	Field string
	Size  int64
}

func (m NodesModel) aggregateFielddataByIndexField() []fielddataByIndexField {
	type key struct {
		index string
		field string
	}

	aggregated := make(map[key]int64)
	for _, fd := range m.state.Fielddata {
		k := key{index: fd.Index, field: fd.Field}
		aggregated[k] += fd.Size
	}

	var result []fielddataByIndexField
	for k, size := range aggregated {
		result = append(result, fielddataByIndexField{
			Index: k.index,
			Field: k.field,
			Size:  size,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Size > result[j].Size
	})

	return result
}

func (m NodesModel) getTotalHeap() int64 {
	var totalHeap int64
	for _, node := range m.state.Nodes {
		heapMax := node.HeapMax
		if heapMax == "" {
			continue
		}
		heapBytes, err := parseSize(heapMax)
		if err == nil {
			totalHeap += heapBytes
		}
	}
	return totalHeap
}

func (m NodesModel) renderFielddataTable() string {
	if len(m.state.Fielddata) == 0 {
		return "No fielddata found"
	}

	aggregated := m.aggregateFielddataByIndexField()
	if len(aggregated) == 0 {
		return "No fielddata found"
	}

	totalHeap := m.getTotalHeap()
	var totalFielddata int64
	for _, fd := range aggregated {
		totalFielddata += fd.Size
	}

	var totalPercentage float64
	if totalHeap > 0 {
		totalPercentage = float64(totalFielddata) / float64(totalHeap) * 100
	}

	var rows [][]string
	var pctValues []string
	visibleItems := m.visibleItems(len(aggregated))
	for _, fd := range aggregated[visibleItems.start:visibleItems.end] {
		field := fd.Field
		if field == "" {
			field = "(all)"
		}

		var heapPercent float64
		if totalHeap > 0 {
			heapPercent = float64(fd.Size) / float64(totalHeap) * 100
		}

		percentStr := fmt.Sprintf("%.1f", heapPercent)
		pctValues = append(pctValues, percentStr)

		rows = append(rows, []string{
			fd.Index,
			field,
			formatBytes(fd.Size),
			fmt.Sprintf("%s %s", percentStr, RenderBar(heapPercent, 10)),
		})
	}

	t := table.New().
		Headers("index", "field", "size", "heap%").
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle()
			if col >= 2 && col != 3 {
				base = base.Align(lipgloss.Right)
			} else if col == 3 {
				base = base.Align(lipgloss.Center)
			}
			if row == table.HeaderRow {
				return base.Bold(true).Foreground(ColorWhite)
			}
			if col == 3 && row >= 0 && row < len(pctValues) {
				return base.Inherit(m.percentStyle(pctValues[row]))
			}
			return base
		})

	totalPercentStr := fmt.Sprintf("%.1f", totalPercentage)
	boldStyle := lipgloss.NewStyle().Bold(true)
	totalLine := fmt.Sprintf("%s  %s  %s",
		boldStyle.Render("TOTAL"),
		boldStyle.Render(formatBytes(totalFielddata)),
		boldStyle.Inherit(m.percentStyle(totalPercentStr)).Render(
			fmt.Sprintf("%s %s", totalPercentStr, RenderBar(totalPercentage, 10)),
		),
	)

	return t.Render() + "\n" + totalLine
}

func (m NodesModel) renderLegend() string {
	greenStyle := lipgloss.NewStyle().Foreground(ColorGreen)
	yellowStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	redStyle := lipgloss.NewStyle().Foreground(ColorRed)
	grayStyle := lipgloss.NewStyle().Foreground(ColorGray)

	switch m.activeView {
	case ViewMemory:
		return grayStyle.Render("heap%: ") +
			greenStyle.Render("<75") +
			grayStyle.Render(" | ") +
			yellowStyle.Render("75-84") +
			grayStyle.Render(" | ") +
			redStyle.Render(">=85")
	case ViewDisk:
		return grayStyle.Render("disk%: ") +
			greenStyle.Render("<75") +
			grayStyle.Render(" | ") +
			yellowStyle.Render("75-84") +
			grayStyle.Render(" | ") +
			redStyle.Render(">=85")
	case ViewFielddata:
		return grayStyle.Render("heap%: ") +
			greenStyle.Render("<75") +
			grayStyle.Render(" | ") +
			yellowStyle.Render("75-84") +
			grayStyle.Render(" | ") +
			redStyle.Render(">=85")
	default:
		return ""
	}
}

type visibleRange struct {
	start int
	end   int
}

func (m NodesModel) visibleItems(total int) visibleRange {
	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 10
	}

	start := m.scrollY
	if start >= total {
		start = total - 1
	}
	if start < 0 {
		start = 0
	}

	end := start + maxVisible
	if end > total {
		end = total
	}

	return visibleRange{start: start, end: end}
}

func (m NodesModel) getItemCount() int {
	if m.state == nil {
		return 0
	}
	switch m.activeView {
	case ViewMemory, ViewDisk:
		return len(m.state.Nodes)
	case ViewFielddata:
		return len(m.aggregateFielddataByIndexField())
	}
	return 0
}

func (m NodesModel) percentStyle(pctStr string) lipgloss.Style {
	pct, err := strconv.ParseFloat(strings.TrimSpace(pctStr), 64)
	if err != nil {
		return lipgloss.NewStyle().Foreground(ColorGray)
	}

	if pct >= 85 {
		return lipgloss.NewStyle().Foreground(ColorRed)
	} else if pct >= 75 {
		return lipgloss.NewStyle().Foreground(ColorYellow)
	}
	return lipgloss.NewStyle().Foreground(ColorGreen)
}

func (m NodesModel) parsePercent(pctStr string) float64 {
	pct, err := strconv.ParseFloat(strings.TrimSpace(pctStr), 64)
	if err != nil {
		return 0
	}
	return pct
}

func (m *NodesModel) updateNodeSearch() {
	if m.state == nil {
		return
	}
	lines := m.getSearchableLines()
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.scrollY = match
	}
}

func (m *NodesModel) getSearchableLines() []string {
	if m.state == nil {
		return nil
	}
	var lines []string
	switch m.activeView {
	case ViewMemory, ViewDisk:
		for _, node := range m.state.Nodes {
			lines = append(lines, node.Name)
		}
	case ViewFielddata:
		aggregated := m.aggregateFielddataByIndexField()
		for _, fd := range aggregated {
			field := fd.Field
			if field == "" {
				field = "(all)"
			}
			lines = append(lines, fd.Index+" "+field)
		}
	}
	return lines
}

func formatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fgb", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fmb", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fkb", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%db", bytes)
	}
}

func parseSize(s string) (int64, error) {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(s, "tb") {
		multiplier = tb
		numStr = strings.TrimSuffix(s, "tb")
	} else if strings.HasSuffix(s, "gb") {
		multiplier = gb
		numStr = strings.TrimSuffix(s, "gb")
	} else if strings.HasSuffix(s, "mb") {
		multiplier = mb
		numStr = strings.TrimSuffix(s, "mb")
	} else if strings.HasSuffix(s, "kb") {
		multiplier = kb
		numStr = strings.TrimSuffix(s, "kb")
	} else if strings.HasSuffix(s, "b") {
		multiplier = 1
		numStr = strings.TrimSuffix(s, "b")
	} else {
		numStr = s
	}

	numStr = strings.TrimSpace(numStr)
	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", s)
	}

	return int64(value * float64(multiplier)), nil
}
