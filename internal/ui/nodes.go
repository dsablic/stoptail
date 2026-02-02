package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/labtiva/stoptail/internal/es"
)

type NodesView int

const (
	ViewMemory NodesView = iota
	ViewDisk
	ViewThreadPools
	ViewHotThreads
	ViewFielddata
	ViewShardHealth
	ViewTemplates
	ViewClusterSettings
	ViewDeprecations
)

type NodesModel struct {
	state            *es.NodesState
	clusterSettings  *es.ClusterSettings
	threadPools      []es.ThreadPoolInfo
	hotThreads       string
	templates        []es.IndexTemplate
	deprecations     *es.DeprecationInfo
	shardHealth      []es.ShardHealth
	activeView       NodesView
	scrollY          int
	selectedSetting  int
	settingDetail    *clusterSetting
	selectedTemplate int
	templateDetail   *es.IndexTemplate
	width            int
	height           int
	loading          bool
	filter           textinput.Model
	filterActive     bool
}

func NewNodes() NodesModel {
	f := textinput.New()
	f.Prompt = ""
	f.CharLimit = 100
	return NodesModel{
		activeView: ViewMemory,
		loading:    true,
		filter:     f,
	}
}

func (m NodesModel) matchesFilter(text string) bool {
	if m.filter.Value() == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(m.filter.Value()))
}

func (m NodesModel) getFilteredSettings() []clusterSetting {
	allSettings := m.getClusterSettingsList()
	if m.filter.Value() == "" {
		return allSettings
	}
	var filtered []clusterSetting
	for _, s := range allSettings {
		if m.matchesFilter(s.Key + " " + s.Value + " " + s.Source) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (m NodesModel) getFilteredTemplates() []es.IndexTemplate {
	if m.templates == nil {
		return nil
	}
	if m.filter.Value() == "" {
		return m.templates
	}
	var filtered []es.IndexTemplate
	for _, t := range m.templates {
		searchText := t.Name + " " + strings.Join(t.IndexPatterns, " ")
		if m.matchesFilter(searchText) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (m NodesModel) getRowCounts() (filtered, total int) {
	switch m.activeView {
	case ViewMemory, ViewDisk:
		total = len(m.state.Nodes)
		for _, node := range m.state.Nodes {
			if m.matchesFilter(node.Name + " " + node.Version) {
				filtered++
			}
		}
	case ViewFielddata:
		fdData := m.aggregateFielddataByIndexField()
		total = len(fdData)
		for _, fd := range fdData {
			if m.matchesFilter(fd.Index + " " + fd.Field) {
				filtered++
			}
		}
	case ViewClusterSettings:
		allSettings := m.getClusterSettingsList()
		total = len(allSettings)
		filtered = len(m.getFilteredSettings())
	case ViewThreadPools:
		total = len(m.threadPools)
		for _, p := range m.threadPools {
			if m.matchesFilter(p.NodeName + " " + p.Name + " " + p.PoolType) {
				filtered++
			}
		}
	case ViewHotThreads:
		total, filtered = m.countHotThreads()
	case ViewTemplates:
		if m.templates != nil {
			total = len(m.templates)
			filtered = len(m.getFilteredTemplates())
		}
	case ViewDeprecations:
		if m.deprecations != nil {
			total = len(m.deprecations.Deprecations)
			filtered = len(m.getFilteredDeprecations())
		}
	case ViewShardHealth:
		total = len(m.shardHealth)
		filtered = len(m.getFilteredShardHealth())
	}
	return filtered, total
}

func (m NodesModel) countHotThreads() (total, filtered int) {
	if m.hotThreads == "" {
		return 0, 0
	}
	var currentNode string
	lines := strings.Split(m.hotThreads, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "::: {") {
			end := strings.Index(line[5:], "}")
			if end > 0 {
				currentNode = line[5 : 5+end]
			}
		} else if currentNode != "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Hot threads at") || trimmed == "" {
				continue
			}
			if len(trimmed) > 0 && (trimmed[0] >= '0' && trimmed[0] <= '9') {
				if ht := parseHotThread(currentNode, trimmed); ht != nil {
					total++
					if m.matchesFilter(ht.node + " " + ht.total + " " + ht.cpu) {
						filtered++
					}
				}
			}
		}
	}
	return total, filtered
}


func (m *NodesModel) SetState(state *es.NodesState) {
	m.state = state
	m.loading = state == nil
	m.scrollY = 0
}

func (m *NodesModel) SetClusterSettings(settings *es.ClusterSettings) {
	m.clusterSettings = settings
}

func (m *NodesModel) SetThreadPools(pools []es.ThreadPoolInfo) {
	m.threadPools = pools
}

func (m *NodesModel) SetHotThreads(threads string) {
	m.hotThreads = threads
}

func (m *NodesModel) SetTemplates(templates []es.IndexTemplate) {
	m.templates = templates
}

func (m *NodesModel) SetDeprecations(deprecations *es.DeprecationInfo) {
	m.deprecations = deprecations
}

func (m *NodesModel) SetShardHealth(indices []es.IndexInfo) {
	var health []es.ShardHealth
	for _, idx := range indices {
		health = append(health, es.AnalyzeShardHealth(idx))
	}
	sort.Slice(health, func(i, j int) bool {
		if health[i].Status != health[j].Status {
			return health[i].Status > health[j].Status
		}
		return health[i].IndexName < health[j].IndexName
	})
	m.shardHealth = health
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
	case "settings":
		m.activeView = ViewClusterSettings
	case "threadpools":
		m.activeView = ViewThreadPools
	case "hotthreads":
		m.activeView = ViewHotThreads
	case "templates":
		m.activeView = ViewTemplates
	case "deprecations":
		m.activeView = ViewDeprecations
	case "shardhealth":
		m.activeView = ViewShardHealth
	}
}

func (m *NodesModel) selectView(view NodesView) {
	m.activeView = view
	m.scrollY = 0
	m.selectedSetting = 0
	m.settingDetail = nil
	m.selectedTemplate = 0
	m.templateDetail = nil
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
		if m.filterActive {
			switch msg.String() {
			case "esc", "enter":
				m.filterActive = false
				m.filter.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				m.scrollY = 0
				m.selectedSetting = 0
				m.selectedTemplate = 0
				return m, cmd
			}
		}
		if m.settingDetail != nil {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.settingDetail = nil
			}
			return m, nil
		}
		if m.templateDetail != nil {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.templateDetail = nil
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			m.filterActive = true
			m.filter.Focus()
			return m, textinput.Blink
		case "esc":
			if m.filter.Value() != "" {
				m.filter.SetValue("")
				m.scrollY = 0
				m.selectedSetting = 0
				m.selectedTemplate = 0
			}
			return m, nil
		case "1":
			m.selectView(ViewMemory)
		case "2":
			m.selectView(ViewDisk)
		case "3":
			m.selectView(ViewThreadPools)
		case "4":
			m.selectView(ViewHotThreads)
		case "5":
			m.selectView(ViewFielddata)
		case "6":
			m.selectView(ViewShardHealth)
		case "7":
			m.selectView(ViewTemplates)
		case "8":
			m.selectView(ViewClusterSettings)
		case "9":
			m.selectView(ViewDeprecations)
		case "enter":
			if m.activeView == ViewClusterSettings {
				filtered := m.getFilteredSettings()
				if m.selectedSetting >= 0 && m.selectedSetting < len(filtered) {
					s := filtered[m.selectedSetting]
					m.settingDetail = &s
				}
			} else if m.activeView == ViewTemplates {
				filtered := m.getFilteredTemplates()
				if m.selectedTemplate >= 0 && m.selectedTemplate < len(filtered) {
					t := filtered[m.selectedTemplate]
					m.templateDetail = &t
				}
			}
			return m, nil
		case "up", "k":
			if m.activeView == ViewClusterSettings {
				if m.selectedSetting > 0 {
					m.selectedSetting--
					maxVisible := m.height - 10
					if maxVisible < 1 {
						maxVisible = 10
					}
					if m.selectedSetting < m.scrollY {
						m.scrollY = m.selectedSetting
					}
				}
			} else if m.activeView == ViewTemplates {
				if m.selectedTemplate > 0 {
					m.selectedTemplate--
					maxVisible := m.height - 10
					if maxVisible < 1 {
						maxVisible = 10
					}
					if m.selectedTemplate < m.scrollY {
						m.scrollY = m.selectedTemplate
					}
				}
			} else if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			if m.activeView == ViewClusterSettings {
				filtered := m.getFilteredSettings()
				if m.selectedSetting < len(filtered)-1 {
					m.selectedSetting++
					maxVisible := m.height - 10
					if maxVisible < 1 {
						maxVisible = 10
					}
					if m.selectedSetting >= m.scrollY+maxVisible {
						m.scrollY = m.selectedSetting - maxVisible + 1
					}
				}
			} else if m.activeView == ViewTemplates {
				filtered := m.getFilteredTemplates()
				if m.selectedTemplate < len(filtered)-1 {
					m.selectedTemplate++
					maxVisible := m.height - 10
					if maxVisible < 1 {
						maxVisible = 10
					}
					if m.selectedTemplate >= m.scrollY+maxVisible {
						m.scrollY = m.selectedTemplate - maxVisible + 1
					}
				}
			} else if m.scrollY < m.getMaxScroll() {
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
					{"[3:Threads]", ViewThreadPools},
					{"[4:Hot]", ViewHotThreads},
					{"[5:Fielddata]", ViewFielddata},
					{"[6:Shards]", ViewShardHealth},
					{"[7:Templates]", ViewTemplates},
					{"[8:Settings]", ViewClusterSettings},
					{"[9:Deprecations]", ViewDeprecations},
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
	if m.activeView == ViewClusterSettings {
		if m.clusterSettings == nil {
			return "Loading cluster settings..."
		}
		if m.settingDetail != nil {
			return m.renderSettingDetailModal()
		}
	} else if m.activeView == ViewTemplates {
		if m.templates == nil {
			return "Loading templates..."
		}
		if m.templateDetail != nil {
			return m.renderTemplateDetailModal()
		}
	} else if m.activeView == ViewThreadPools {
		if m.threadPools == nil {
			return "Loading thread pools..."
		}
	} else if m.loading || m.state == nil {
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
	case ViewClusterSettings:
		b.WriteString(m.renderClusterSettingsTable())
	case ViewThreadPools:
		b.WriteString(m.renderThreadPoolsTable())
	case ViewHotThreads:
		b.WriteString(m.renderHotThreads())
	case ViewTemplates:
		b.WriteString(m.renderTemplates())
	case ViewDeprecations:
		b.WriteString(m.renderDeprecations())
	case ViewShardHealth:
		b.WriteString(m.renderShardHealth())
	}

	if m.activeView == ViewMemory || m.activeView == ViewDisk {
		grayStyle := lipgloss.NewStyle().Foreground(ColorGray)
		masterBadge := lipgloss.NewStyle().Bold(true).Render("*")
		b.WriteString("\n")
		b.WriteString(masterBadge + grayStyle.Render(" Master node"))
	}

	b.WriteString("\n")

	filterStyle := lipgloss.NewStyle().Padding(0, 1)
	countStyle := lipgloss.NewStyle().Foreground(ColorGray)
	filtered, total := m.getRowCounts()
	if m.filterActive {
		b.WriteString(filterStyle.Render("Filter: " + m.filter.View()))
		if m.filter.Value() != "" {
			b.WriteString(countStyle.Render(fmt.Sprintf(" (%d/%d)", filtered, total)))
		}
	} else if m.filter.Value() != "" {
		b.WriteString(filterStyle.Render("Filter: " + m.filter.Value()))
		b.WriteString(countStyle.Render(fmt.Sprintf(" (%d/%d)", filtered, total)))
		b.WriteString(filterStyle.Render(" (Esc to clear)"))
	}

	return b.String()
}

func (m NodesModel) renderTabs() string {
	tabs := []struct {
		key   string
		label string
		view  NodesView
	}{
		{"1", "Memory", ViewMemory},
		{"2", "Disk", ViewDisk},
		{"3", "Threads", ViewThreadPools},
		{"4", "Hot", ViewHotThreads},
		{"5", "Fielddata", ViewFielddata},
		{"6", "Shards", ViewShardHealth},
		{"7", "Templates", ViewTemplates},
		{"8", "Settings", ViewClusterSettings},
		{"9", "Deprecations", ViewDeprecations},
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
	var filtered []es.NodeStats
	for _, node := range m.state.Nodes {
		if m.matchesFilter(node.Name + " " + node.Version) {
			filtered = append(filtered, node)
		}
	}

	if len(filtered) == 0 {
		if m.filter.Value() != "" {
			return "No matching nodes"
		}
		return "No nodes found"
	}

	headers := []string{"node", "version", "heap%", "heap", "fielddata", "query_cache", "segments"}
	var rows [][]string
	var pctValues []string
	visibleNodes := m.visibleItems(len(filtered))
	for _, node := range filtered[visibleNodes.start:visibleNodes.end] {
		heapPct := m.parsePercent(node.HeapPercent)
		pctValues = append(pctValues, node.HeapPercent)

		nodeName := node.Name
		if node.Master == "*" {
			nodeName = lipgloss.NewStyle().Bold(true).Render(nodeName + " *")
		}

		rows = append(rows, []string{
			nodeName,
			node.Version,
			fmt.Sprintf("%s %s", node.HeapPercent, RenderBar(heapPct, 10)),
			node.HeapCurrent,
			node.FielddataSize,
			node.QueryCacheSize,
			node.SegmentsCount,
		})
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Headers(headers...).
		Rows(fittedRows...).
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

func (m NodesModel) renderDiskTable() string {
	var filtered []es.NodeStats
	for _, node := range m.state.Nodes {
		if m.matchesFilter(node.Name + " " + node.Version) {
			filtered = append(filtered, node)
		}
	}

	if len(filtered) == 0 {
		if m.filter.Value() != "" {
			return "No matching nodes"
		}
		return "No nodes found"
	}

	headers := []string{"node", "version", "disk%", "disk.avail", "disk.total", "disk.used", "shards"}
	var rows [][]string
	var pctValues []string
	visibleNodes := m.visibleItems(len(filtered))
	for _, node := range filtered[visibleNodes.start:visibleNodes.end] {
		diskPct := m.parsePercent(node.DiskPercent)
		pctValues = append(pctValues, node.DiskPercent)

		nodeName := node.Name
		if node.Master == "*" {
			nodeName = lipgloss.NewStyle().Bold(true).Render(nodeName + " *")
		}

		rows = append(rows, []string{
			nodeName,
			node.Version,
			fmt.Sprintf("%s %s", node.DiskPercent, RenderBar(diskPct, 10)),
			node.DiskAvail,
			node.DiskTotal,
			node.DiskUsed,
			node.Shards,
		})
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Headers(headers...).
		Rows(fittedRows...).
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
		heapBytes, err := ParseSize(heapMax)
		if err == nil {
			totalHeap += heapBytes
		}
	}
	return totalHeap
}

func (m NodesModel) renderFielddataTable() string {
	aggregated := m.aggregateFielddataByIndexField()

	var filtered []fielddataByIndexField
	for _, fd := range aggregated {
		field := fd.Field
		if field == "" {
			field = "(all)"
		}
		if m.matchesFilter(fd.Index + " " + field) {
			filtered = append(filtered, fd)
		}
	}

	if len(filtered) == 0 {
		if m.filter.Value() != "" {
			return "No matching fielddata"
		}
		return "No fielddata found"
	}

	totalHeap := m.getTotalHeap()
	var totalFielddata int64
	for _, fd := range filtered {
		totalFielddata += fd.Size
	}

	var totalPercentage float64
	if totalHeap > 0 {
		totalPercentage = float64(totalFielddata) / float64(totalHeap) * 100
	}

	headers := []string{"index", "field", "size", "heap%"}
	var rows [][]string
	var pctValues []string
	visibleItems := m.visibleItems(len(filtered))
	for _, fd := range filtered[visibleItems.start:visibleItems.end] {
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

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Headers(headers...).
		Rows(fittedRows...).
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
	switch m.activeView {
	case ViewMemory, ViewDisk:
		if m.state == nil {
			return 0
		}
		return len(m.state.Nodes)
	case ViewFielddata:
		if m.state == nil {
			return 0
		}
		return len(m.aggregateFielddataByIndexField())
	case ViewClusterSettings:
		if m.clusterSettings == nil {
			return 0
		}
		return len(m.getClusterSettingsList())
	case ViewThreadPools:
		return len(m.threadPools)
	case ViewHotThreads:
		total, _ := m.countHotThreads()
		return total
	case ViewTemplates:
		return len(m.templates)
	case ViewDeprecations:
		if m.deprecations == nil {
			return 0
		}
		return len(m.getFilteredDeprecations())
	case ViewShardHealth:
		return len(m.getFilteredShardHealth())
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


type clusterSetting struct {
	Key    string
	Value  string
	Source string
}

func (m NodesModel) getClusterSettingsList() []clusterSetting {
	if m.clusterSettings == nil {
		return nil
	}

	seen := make(map[string]bool)
	var settings []clusterSetting

	for k, v := range m.clusterSettings.Transient {
		settings = append(settings, clusterSetting{Key: k, Value: v, Source: "transient"})
		seen[k] = true
	}

	for k, v := range m.clusterSettings.Persistent {
		if !seen[k] {
			settings = append(settings, clusterSetting{Key: k, Value: v, Source: "persistent"})
			seen[k] = true
		}
	}

	for k, v := range m.clusterSettings.Defaults {
		if !seen[k] {
			settings = append(settings, clusterSetting{Key: k, Value: v, Source: "default"})
		}
	}

	sort.Slice(settings, func(i, j int) bool {
		return settings[i].Key < settings[j].Key
	})

	return settings
}

func (m NodesModel) renderClusterSettingsTable() string {
	filtered := m.getFilteredSettings()

	if len(filtered) == 0 {
		if m.filter.Value() != "" {
			return "No matching settings"
		}
		return "No cluster settings"
	}

	vr := m.visibleItems(len(filtered))

	headers := []string{"Setting", "Value", "Source"}
	var rows [][]string
	var rowIndices []int
	for i := vr.start; i < vr.end && i < len(filtered); i++ {
		s := filtered[i]
		rows = append(rows, []string{s.Key, s.Value, s.Source})
		rowIndices = append(rowIndices, i)
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		Headers(headers...).
		Rows(fittedRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(ColorWhite)
			}
			if row >= 0 && row < len(rowIndices) && rowIndices[row] == m.selectedSetting {
				return style.Background(ColorBlue).Foreground(ColorOnAccent)
			}
			if col == 2 && row >= 0 && row < len(fittedRows) {
				switch fittedRows[row][2] {
				case "transient":
					return style.Foreground(ColorYellow)
				case "persistent":
					return style.Foreground(ColorBlue)
				default:
					return style.Foreground(ColorGray)
				}
			}
			return style.Foreground(ColorWhite)
		})

	return t.Render()
}

func (m NodesModel) renderSettingDetailModal() string {
	s := m.settingDetail
	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	var lines []string
	lines = append(lines, labelStyle.Render("Setting: ")+valueStyle.Render(s.Key))
	lines = append(lines, labelStyle.Render("Source:  ")+valueStyle.Render(s.Source))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Value:"))

	maxValueWidth := m.width - 20
	if maxValueWidth < 40 {
		maxValueWidth = 40
	}
	if maxValueWidth > 100 {
		maxValueWidth = 100
	}

	value := s.Value
	for len(value) > maxValueWidth {
		lines = append(lines, valueStyle.Render(value[:maxValueWidth]))
		value = value[maxValueWidth:]
	}
	if len(value) > 0 {
		lines = append(lines, valueStyle.Render(value))
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		MaxWidth(m.width - 10)

	box := boxStyle.Render(content)
	footer := lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter or Esc to close")

	modal := lipgloss.JoinVertical(lipgloss.Center, box, footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m NodesModel) renderThreadPoolsTable() string {
	var filtered []es.ThreadPoolInfo
	for _, p := range m.threadPools {
		if m.matchesFilter(p.NodeName + " " + p.Name + " " + p.PoolType) {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		if m.filter.Value() != "" {
			return "No matching thread pools"
		}
		return "No thread pools"
	}

	vr := m.visibleItems(len(filtered))

	headers := []string{"Node", "Pool", "Active", "Queue", "Rejected", "Completed", "Size", "Type"}
	var rows [][]string
	for i := vr.start; i < vr.end && i < len(filtered); i++ {
		p := filtered[i]
		rows = append(rows, []string{p.NodeName, p.Name, p.Active, p.Queue, p.Rejected, p.Completed, p.PoolSize, p.PoolType})
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		Headers(headers...).
		Rows(fittedRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(ColorWhite)
			}
			if row >= 0 && row < len(fittedRows) {
				switch col {
				case 2:
					if fittedRows[row][2] != "0" {
						return style.Foreground(ColorGreen)
					}
				case 3:
					if fittedRows[row][3] != "0" {
						return style.Foreground(ColorYellow)
					}
				case 4:
					if fittedRows[row][4] != "0" {
						return style.Foreground(ColorRed)
					}
				}
			}
			return style.Foreground(ColorWhite)
		})

	return t.Render()
}

type hotThread struct {
	node       string
	total      string
	cpu        string
	other      string
	threadType string
}

func parseHotThread(node, line string) *hotThread {
	ht := &hotThread{node: node}

	pctEnd := strings.Index(line, "%")
	if pctEnd == -1 {
		return nil
	}
	ht.total = line[:pctEnd+1]

	bracketStart := strings.Index(line, "[")
	bracketEnd := strings.Index(line, "]")
	if bracketStart > 0 && bracketEnd > bracketStart {
		breakdown := line[bracketStart+1 : bracketEnd]
		parts := strings.Split(breakdown, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "cpu=") {
				ht.cpu = strings.TrimPrefix(p, "cpu=")
			} else if strings.HasPrefix(p, "other=") {
				ht.other = strings.TrimPrefix(p, "other=")
			}
		}
	}

	if idx := strings.Index(line, "thread '"); idx > 0 {
		rest := line[idx+8:]
		if end := strings.Index(rest, "'"); end > 0 {
			threadName := rest[:end]
			ht.threadType = extractThreadType(threadName)
		}
	}

	return ht
}

func extractThreadType(threadName string) string {
	bracketParts := strings.Split(threadName, "][")
	if len(bracketParts) >= 2 {
		part := strings.TrimPrefix(bracketParts[1], "[")
		part = strings.TrimSuffix(part, "]")
		if idx := strings.Index(part, "["); idx > 0 {
			part = part[:idx]
		}
		return part
	}
	if strings.Contains(threadName, "lucene") {
		return "merge"
	}
	if strings.Contains(threadName, "refresh") {
		return "refresh"
	}
	if strings.Contains(threadName, "flush") {
		return "flush"
	}
	return "other"
}

func threadTypeColor(threadType string) lipgloss.Color {
	switch threadType {
	case "search", "get":
		return ColorBlue
	case "write", "bulk", "index":
		return ColorGreen
	case "merge", "refresh", "flush", "force_merge", "warmer":
		return ColorYellow
	case "management", "generic":
		return ColorGray
	default:
		return ColorWhite
	}
}

func (m NodesModel) renderHotThreads() string {
	if m.hotThreads == "" {
		return "No hot threads data. Press 'r' to refresh."
	}

	var threads []hotThread
	var currentNode string
	lines := strings.Split(m.hotThreads, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "::: {") {
			end := strings.Index(line[5:], "}")
			if end > 0 {
				currentNode = line[5 : 5+end]
			} else {
				currentNode = "unknown"
			}
		} else if currentNode != "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Hot threads at") || trimmed == "" {
				continue
			}
			if len(trimmed) > 0 && (trimmed[0] >= '0' && trimmed[0] <= '9') {
				if ht := parseHotThread(currentNode, trimmed); ht != nil {
					threads = append(threads, *ht)
				}
			}
		}
	}

	if len(threads) == 0 {
		idleStyle := lipgloss.NewStyle().Foreground(ColorGray)
		return idleStyle.Render("All nodes idle - no hot threads detected")
	}

	var filtered []hotThread
	for _, t := range threads {
		if m.matchesFilter(t.node + " " + t.total + " " + t.cpu) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) == 0 {
		return "No matching threads"
	}

	vr := m.visibleItems(len(filtered))

	headers := []string{"Node", "Type", "Total", "CPU", "Other"}
	var rows [][]string
	for i := vr.start; i < vr.end && i < len(filtered); i++ {
		t := filtered[i]
		rows = append(rows, []string{t.node, t.threadType, t.total, t.cpu, t.other})
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		Headers(headers...).
		Rows(fittedRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(ColorWhite)
			}
			if col == 1 && row >= 0 && row < len(fittedRows) {
				return style.Foreground(threadTypeColor(fittedRows[row][1]))
			}
			if col == 2 && row >= 0 && row < len(fittedRows) {
				pctStr := strings.TrimSuffix(fittedRows[row][2], "%")
				return style.Inherit(m.percentStyle(pctStr))
			}
			return style.Foreground(ColorWhite)
		})

	return tbl.Render()
}

func (m NodesModel) renderTemplates() string {
	if m.templates == nil {
		return "Loading templates..."
	}

	if len(m.templates) == 0 {
		return "No index templates found"
	}

	filtered := m.getFilteredTemplates()

	if len(filtered) == 0 {
		return "No matching templates"
	}

	vr := m.visibleItems(len(filtered))

	headers := []string{"Name", "Patterns", "Pri", "S/R", "Composed Of", "DS"}
	var rows [][]string
	var rowIndices []int
	for i := vr.start; i < vr.end && i < len(filtered); i++ {
		t := filtered[i]
		patterns := strings.Join(t.IndexPatterns, ", ")
		composed := strings.Join(t.ComposedOf, ", ")
		dataStream := ""
		if t.DataStream {
			dataStream = "Y"
		}
		shards := t.NumberOfShards + "/" + t.NumberOfReplicas
		rows = append(rows, []string{
			t.Name,
			patterns,
			fmt.Sprintf("%d", t.Priority),
			shards,
			composed,
			dataStream,
		})
		rowIndices = append(rowIndices, i)
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		Headers(headers...).
		Rows(fittedRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(ColorWhite)
			}
			if row >= 0 && row < len(rowIndices) && rowIndices[row] == m.selectedTemplate {
				return style.Background(ColorBlue).Foreground(ColorOnAccent)
			}
			if col == 2 && row >= 0 && row < len(fittedRows) {
				return style.Align(lipgloss.Right)
			}
			if col == 5 && row >= 0 && row < len(fittedRows) && fittedRows[row][5] == "Y" {
				return style.Foreground(ColorBlue)
			}
			return style.Foreground(ColorWhite)
		})

	return tbl.Render()
}

func (m NodesModel) renderTemplateDetailModal() string {
	t := m.templateDetail
	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	var lines []string
	lines = append(lines, labelStyle.Render("Template: ")+valueStyle.Render(t.Name))
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Index Patterns:"))
	for _, p := range t.IndexPatterns {
		lines = append(lines, "  "+valueStyle.Render(p))
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Priority: ")+valueStyle.Render(fmt.Sprintf("%d", t.Priority)))
	if t.Version > 0 {
		lines = append(lines, labelStyle.Render("Version:  ")+valueStyle.Render(fmt.Sprintf("%d", t.Version)))
	}
	lines = append(lines, "")

	if t.NumberOfShards != "" || t.NumberOfReplicas != "" {
		lines = append(lines, labelStyle.Render("Settings:"))
		if t.NumberOfShards != "" {
			lines = append(lines, "  "+labelStyle.Render("Shards:   ")+valueStyle.Render(t.NumberOfShards))
		}
		if t.NumberOfReplicas != "" {
			lines = append(lines, "  "+labelStyle.Render("Replicas: ")+valueStyle.Render(t.NumberOfReplicas))
		}
		lines = append(lines, "")
	}

	if len(t.ComposedOf) > 0 {
		lines = append(lines, labelStyle.Render("Composed Of:"))
		for _, c := range t.ComposedOf {
			lines = append(lines, "  "+valueStyle.Render(c))
		}
		lines = append(lines, "")
	}

	if t.DataStream {
		lines = append(lines, labelStyle.Render("Data Stream: ")+lipgloss.NewStyle().Foreground(ColorBlue).Render("yes"))
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		MaxWidth(m.width - 10)

	box := boxStyle.Render(content)
	footer := lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter or Esc to close")

	modal := lipgloss.JoinVertical(lipgloss.Center, box, footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m NodesModel) renderDeprecations() string {
	if m.deprecations == nil {
		return "Loading deprecations..."
	}

	if len(m.deprecations.Deprecations) == 0 {
		return lipgloss.NewStyle().Foreground(ColorGreen).Render("No deprecations found")
	}

	filtered := m.getFilteredDeprecations()
	visible := m.visibleItems(len(filtered))

	headers := []string{"level", "category", "resource", "message"}
	var rows [][]string
	for i := visible.start; i < visible.end; i++ {
		dep := filtered[i]
		resource := dep.Resource
		if resource == "" {
			resource = "-"
		}
		rows = append(rows, []string{dep.Level, dep.Category, resource, dep.Message})
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		Headers(headers...).
		Rows(fittedRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
			}
			if row >= 0 && row < len(filtered) {
				if col == 0 {
					switch filtered[row].Level {
					case "critical":
						return lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
					case "warning":
						return lipgloss.NewStyle().Foreground(ColorYellow)
					}
				}
			}
			return lipgloss.NewStyle()
		})

	return t.Render()
}

func (m NodesModel) getFilteredDeprecations() []es.Deprecation {
	if m.deprecations == nil {
		return nil
	}
	if m.filter.Value() == "" {
		return m.deprecations.Deprecations
	}
	filterVal := strings.ToLower(m.filter.Value())
	var filtered []es.Deprecation
	for _, dep := range m.deprecations.Deprecations {
		if strings.Contains(strings.ToLower(dep.Message), filterVal) ||
			strings.Contains(strings.ToLower(dep.Category), filterVal) ||
			strings.Contains(strings.ToLower(dep.Resource), filterVal) {
			filtered = append(filtered, dep)
		}
	}
	return filtered
}

func (m NodesModel) getFilteredShardHealth() []es.ShardHealth {
	if m.filter.Value() == "" {
		return m.shardHealth
	}
	filterVal := strings.ToLower(m.filter.Value())
	var filtered []es.ShardHealth
	for _, h := range m.shardHealth {
		if strings.Contains(strings.ToLower(h.IndexName), filterVal) ||
			strings.Contains(strings.ToLower(h.StatusText), filterVal) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func (m NodesModel) renderShardHealth() string {
	if len(m.shardHealth) == 0 {
		return "No indices found"
	}

	filtered := m.getFilteredShardHealth()
	visible := m.visibleItems(len(filtered))

	headers := []string{"index", "shards", "size", "avg shard", "docs/shard", "status"}
	var rows [][]string
	for i := visible.start; i < visible.end; i++ {
		h := filtered[i]
		avgShard := "-"
		if h.AvgShardSize > 0 {
			avgShard = formatShardSize(h.AvgShardSize)
		}
		docsPerShard := "-"
		if h.AvgDocsPerShard > 0 {
			docsPerShard = FormatNumber(fmt.Sprintf("%d", h.AvgDocsPerShard))
		}
		rows = append(rows, []string{
			h.IndexName,
			fmt.Sprintf("%d", h.ShardCount),
			formatShardSize(h.TotalSize),
			avgShard,
			docsPerShard,
			h.StatusText,
		})
	}

	widths := AutoColumnWidths(headers, rows, m.width)
	fittedRows := FitColumns(rows, widths)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		Headers(headers...).
		Rows(fittedRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
			}
			if col == 5 && row >= 0 && row < len(filtered) {
				switch filtered[row].Status {
				case es.ShardHealthCritical:
					return lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
				case es.ShardHealthWarning:
					return lipgloss.NewStyle().Foreground(ColorYellow)
				case es.ShardHealthOK:
					return lipgloss.NewStyle().Foreground(ColorGreen)
				}
			}
			return lipgloss.NewStyle()
		})

	return t.Render()
}

func formatShardSize(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case b >= tb:
		return fmt.Sprintf("%.1ftb", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1fgb", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fmb", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fkb", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%db", b)
	}
}
