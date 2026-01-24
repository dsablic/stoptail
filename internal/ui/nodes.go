package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	colWidths := []int{20, 8, 12, 12, 12, 14, 10}
	headers := []string{"node", "heap%", "", "heap", "fielddata", "query_cache", "segments"}

	var b strings.Builder
	b.WriteString(m.renderTableHeader(headers, colWidths))

	visibleNodes := m.visibleItems(len(m.state.Nodes))
	for _, node := range m.state.Nodes[visibleNodes.start:visibleNodes.end] {
		heapPctStyle := m.percentStyle(node.HeapPercent)
		heapPct := m.parsePercent(node.HeapPercent)

		row := []string{
			m.leftAlign(node.Name, colWidths[0]),
			heapPctStyle.Render(m.rightAlign(node.HeapPercent, colWidths[1])),
			RenderBar(heapPct, 10),
			m.rightAlign(node.HeapCurrent, colWidths[3]),
			m.rightAlign(node.FielddataSize, colWidths[4]),
			m.rightAlign(node.QueryCacheSize, colWidths[5]),
			m.rightAlign(node.SegmentsCount, colWidths[6]),
		}
		b.WriteString(strings.Join(row, " "))
		b.WriteString("\n")
	}

	return b.String()
}

func (m NodesModel) renderDiskTable() string {
	if len(m.state.Nodes) == 0 {
		return "No nodes found"
	}

	colWidths := []int{20, 10, 8, 12, 12, 12, 12, 8}
	headers := []string{"node", "version", "disk%", "", "disk.avail", "disk.total", "disk.used", "shards"}

	var b strings.Builder
	b.WriteString(m.renderTableHeader(headers, colWidths, 2))

	visibleNodes := m.visibleItems(len(m.state.Nodes))
	for _, node := range m.state.Nodes[visibleNodes.start:visibleNodes.end] {
		diskPctStyle := m.percentStyle(node.DiskPercent)
		diskPct := m.parsePercent(node.DiskPercent)
		row := []string{
			m.leftAlign(node.Name, colWidths[0]),
			m.leftAlign(node.Version, colWidths[1]),
			diskPctStyle.Render(m.rightAlign(node.DiskPercent, colWidths[2])),
			RenderBar(diskPct, 10),
			m.rightAlign(node.DiskAvail, colWidths[4]),
			m.rightAlign(node.DiskTotal, colWidths[5]),
			m.rightAlign(node.DiskUsed, colWidths[6]),
			m.rightAlign(node.Shards, colWidths[7]),
		}
		b.WriteString(strings.Join(row, " "))
		b.WriteString("\n")
	}

	return b.String()
}

func (m NodesModel) renderFielddataTable() string {
	if len(m.state.Fielddata) == 0 {
		return "No fielddata found"
	}

	colWidths := []int{18, 25, 25, 12}
	headers := []string{"node", "index", "field", "size"}

	var b strings.Builder
	b.WriteString(m.renderTableHeader(headers, colWidths, 3))

	visibleItems := m.visibleItems(len(m.state.Fielddata))
	for _, fd := range m.state.Fielddata[visibleItems.start:visibleItems.end] {
		field := fd.Field
		if field == "" {
			field = "(all)"
		}
		row := []string{
			m.leftAlign(fd.Node, colWidths[0]),
			m.leftAlign(fd.Index, colWidths[1]),
			m.leftAlign(field, colWidths[2]),
			m.rightAlign(formatBytes(fd.Size), colWidths[3]),
		}
		b.WriteString(strings.Join(row, " "))
		b.WriteString("\n")
	}

	return b.String()
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
		return grayStyle.Render("fielddata size per node/index/field")
	default:
		return ""
	}
}

func (m NodesModel) renderTableHeader(headers []string, widths []int, leftAlignCols ...int) string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	numLeftAlign := 1
	if len(leftAlignCols) > 0 {
		numLeftAlign = leftAlignCols[0]
	}
	var headerParts []string
	for i, h := range headers {
		if i < numLeftAlign {
			headerParts = append(headerParts, headerStyle.Render(m.leftAlign(h, widths[i])))
		} else {
			headerParts = append(headerParts, headerStyle.Render(m.rightAlign(h, widths[i])))
		}
	}
	header := strings.Join(headerParts, " ")

	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}
	totalWidth += len(widths) - 1

	return header + "\n" + strings.Repeat("-", totalWidth) + "\n"
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
		return len(m.state.Fielddata)
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

func (m NodesModel) leftAlign(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

func (m NodesModel) rightAlign(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return strings.Repeat(" ", width-len(r)) + s
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
	for _, node := range m.state.Nodes {
		lines = append(lines, node.Name)
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
