package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type MappingsPane int

const (
	PaneIndices MappingsPane = iota
	PaneMappings
)

type MappingsModel struct {
	indices       []es.IndexInfo
	selectedIndex int
	scrollY       int
	width         int
	height        int
	activePane    MappingsPane
	filterActive  bool
	filterText    string
	treeView      bool
	search        SearchBar

	mappings      *es.IndexMappings
	analyzers     []es.AnalyzerInfo
	mappingScroll int
	loadingIndex  string
}

type fetchMappingsMsg struct {
	indexName string
}

func NewMappings() MappingsModel {
	return MappingsModel{
		activePane: PaneIndices,
		treeView:   false,
		search:     NewSearchBar(),
	}
}

func (m *MappingsModel) SetIndices(indices []es.IndexInfo) {
	m.indices = indices
	if m.selectedIndex >= len(indices) {
		m.selectedIndex = max(0, len(indices)-1)
	}
}

func (m *MappingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MappingsModel) SetMappings(mappings *es.IndexMappings, analyzers []es.AnalyzerInfo) {
	m.mappings = mappings
	m.analyzers = analyzers
	m.loadingIndex = ""
	m.mappingScroll = 0
}

func (m MappingsModel) SelectedIndexName() string {
	filtered := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(filtered) {
		return filtered[m.selectedIndex].Name
	}
	return ""
}

func (m MappingsModel) IsLoading() bool {
	return m.loadingIndex != ""
}

func (m *MappingsModel) SetLoading(indexName string) {
	m.loadingIndex = indexName
}

func (m MappingsModel) Update(msg tea.Msg) (MappingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.search.Active() {
			switch msg.String() {
			case "esc":
				m.search.Deactivate()
				return m, nil
			case "enter":
				if match := m.search.NextMatch(); match >= 0 {
					m.mappingScroll = match
				}
				return m, nil
			case "shift+enter":
				if match := m.search.PrevMatch(); match >= 0 {
					m.mappingScroll = match
				}
				return m, nil
			default:
				cmd := m.search.Update(msg)
				(&m).updateMappingSearch()
				return m, cmd
			}
		}

		if m.filterActive {
			return m.handleFilterInput(msg)
		}

		switch msg.String() {
		case "ctrl+f":
			if !m.filterActive {
				m.search.Activate()
				return m, nil
			}
		case "/":
			if m.activePane == PaneIndices {
				m.filterActive = true
				m.filterText = ""
			}
		case "t":
			if m.activePane == PaneMappings {
				m.treeView = !m.treeView
			}
		case "left", "h":
			m.activePane = PaneIndices
		case "right", "l":
			if m.activePane == PaneIndices {
				m.activePane = PaneMappings
				return m, m.fetchMappingsCmd()
			}
		case "enter":
			if m.activePane == PaneIndices {
				m.activePane = PaneMappings
				return m, m.fetchMappingsCmd()
			}
		case "up", "k":
			if m.activePane == PaneIndices {
				if m.selectedIndex > 0 {
					m.selectedIndex--
					if m.selectedIndex < m.scrollY {
						m.scrollY = m.selectedIndex
					}
				}
			} else {
				if m.mappingScroll > 0 {
					m.mappingScroll--
				}
			}
		case "down", "j":
			if m.activePane == PaneIndices {
				filtered := m.filteredIndices()
				if m.selectedIndex < len(filtered)-1 {
					m.selectedIndex++
					maxVisible := m.maxVisibleIndices()
					if m.selectedIndex >= m.scrollY+maxVisible {
						m.scrollY = m.selectedIndex - maxVisible + 1
					}
				}
			} else {
				maxScroll := m.maxMappingScroll()
				if m.mappingScroll < maxScroll {
					m.mappingScroll++
				}
			}
		}
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			scrollAmount := 3
			if m.activePane == PaneIndices {
				maxScroll := m.maxIndicesScroll()
				if msg.Button == tea.MouseButtonWheelUp {
					m.scrollY = max(0, m.scrollY-scrollAmount)
				} else {
					m.scrollY = min(maxScroll, m.scrollY+scrollAmount)
				}
			} else {
				maxScroll := m.maxMappingScroll()
				if msg.Button == tea.MouseButtonWheelUp {
					m.mappingScroll = max(0, m.mappingScroll-scrollAmount)
				} else {
					m.mappingScroll = min(maxScroll, m.mappingScroll+scrollAmount)
				}
			}
		}
	}
	return m, nil
}

func (m MappingsModel) handleFilterInput(msg tea.KeyMsg) (MappingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.filterActive = false
		m.selectedIndex = 0
		m.scrollY = 0
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.selectedIndex = 0
			m.scrollY = 0
		}
	default:
		if len(msg.String()) == 1 {
			m.filterText += msg.String()
			m.selectedIndex = 0
			m.scrollY = 0
		}
	}
	return m, nil
}

func (m MappingsModel) fetchMappingsCmd() tea.Cmd {
	indexName := m.SelectedIndexName()
	if indexName == "" {
		return nil
	}
	return func() tea.Msg {
		return fetchMappingsMsg{indexName: indexName}
	}
}

func (m MappingsModel) filteredIndices() []es.IndexInfo {
	if m.filterText == "" {
		return m.indices
	}
	var filtered []es.IndexInfo
	filterLower := strings.ToLower(m.filterText)
	for _, idx := range m.indices {
		if strings.Contains(strings.ToLower(idx.Name), filterLower) {
			filtered = append(filtered, idx)
		}
	}
	return filtered
}

func (m MappingsModel) maxVisibleIndices() int {
	rows := m.height - 4
	if rows < 1 {
		return 10
	}
	return rows
}

func (m MappingsModel) maxIndicesScroll() int {
	filtered := m.filteredIndices()
	maxScroll := len(filtered) - m.maxVisibleIndices()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m MappingsModel) maxMappingScroll() int {
	if m.mappings == nil {
		return 0
	}
	totalLines := m.countMappingLines()
	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 10
	}
	maxScroll := totalLines - maxVisible
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m MappingsModel) countMappingLines() int {
	if m.mappings == nil {
		return 0
	}
	lines := 2
	if m.treeView {
		lines += m.countTreeLines(m.mappings.Fields, 0)
	} else {
		lines += len(m.flattenMappingFields(m.mappings.Fields))
	}
	if len(m.analyzers) > 0 {
		lines += 2 + len(m.analyzers)
	}
	return lines
}

func (m MappingsModel) countTreeLines(fields []es.MappingField, depth int) int {
	count := 0
	for _, f := range fields {
		count++
		if len(f.Children) > 0 {
			count += m.countTreeLines(f.Children, depth+1)
		}
	}
	return count
}

func (m MappingsModel) View() string {
	if len(m.indices) == 0 {
		return lipgloss.NewStyle().
			Foreground(ColorGray).
			Padding(2).
			Render("No indices found.")
	}

	leftWidth := (m.width - 5) / 3
	rightWidth := m.width - leftWidth - 5

	leftPane := m.renderIndexList(leftWidth)
	rightPane := m.renderMappingsPane(rightWidth)

	leftLines := strings.Split(leftPane, "\n")
	rightLines := strings.Split(rightPane, "\n")

	maxLines := max(len(leftLines), len(rightLines))

	var paneLines []string
	for i := range maxLines {
		left := ""
		right := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}
		paneLines = append(paneLines, TrimANSI(left)+" "+TrimANSI(right))
	}

	return strings.Join(paneLines, "\n")
}

func (m MappingsModel) renderIndexList(width int) string {
	borderColor := ColorGray
	if m.activePane == PaneIndices {
		borderColor = ColorBlue
	}

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(m.height - 2)

	var b strings.Builder

	if m.filterActive {
		filterStyle := lipgloss.NewStyle().Foreground(ColorBlue)
		b.WriteString(filterStyle.Render("/"+m.filterText) + "_")
		b.WriteString("\n")
	} else if m.filterText != "" {
		filterStyle := lipgloss.NewStyle().Foreground(ColorGray)
		b.WriteString(filterStyle.Render("/"+m.filterText))
		b.WriteString("\n")
	}

	filtered := m.filteredIndices()
	if len(filtered) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render("No matches"))
		return paneStyle.Render(b.String())
	}

	maxVisible := m.maxVisibleIndices()
	if m.filterActive || m.filterText != "" {
		maxVisible--
	}
	endIdx := min(m.scrollY+maxVisible, len(filtered))

	for i := m.scrollY; i < endIdx; i++ {
		idx := filtered[i]
		isSelected := i == m.selectedIndex

		name := Truncate(idx.Name, width-4)

		rowStyle := lipgloss.NewStyle()
		if isSelected {
			rowStyle = rowStyle.Background(ColorBlue).Foreground(ColorOnAccent)
		}

		healthDot := lipgloss.NewStyle().Foreground(HealthColor(idx.Health)).Render("*")

		if isSelected {
			b.WriteString(rowStyle.Render(healthDot + " " + name))
		} else {
			b.WriteString(healthDot + " " + rowStyle.Render(name))
		}
		b.WriteString("\n")
	}

	return paneStyle.Render(b.String())
}

func (m MappingsModel) renderMappingsPane(width int) string {
	borderColor := ColorGray
	if m.activePane == PaneMappings {
		borderColor = ColorBlue
	}

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(m.height - 2)

	if m.loadingIndex != "" {
		return paneStyle.Render(fmt.Sprintf("Loading mappings for %s...", m.loadingIndex))
	}

	if m.mappings == nil {
		hint := lipgloss.NewStyle().Foreground(ColorGray).Render("Select an index and press Enter or Right arrow")
		return paneStyle.Render(hint)
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	b.WriteString(headerStyle.Render(m.mappings.IndexName))
	b.WriteString(fmt.Sprintf(" (%d fields)", m.mappings.FieldCount))
	b.WriteString("\n")

	viewLabel := "flat"
	if m.treeView {
		viewLabel = "tree"
	}
	toggleHint := lipgloss.NewStyle().Foreground(ColorGray).Render(fmt.Sprintf("[t: toggle %s view]", viewLabel))
	b.WriteString(toggleHint)
	b.WriteString("\n")

	var fieldLines []string

	if len(m.analyzers) > 0 {
		fieldLines = append(fieldLines, m.renderAnalyzers(width-4)...)
		fieldLines = append(fieldLines, "")
	}

	if m.treeView {
		fieldLines = append(fieldLines, m.renderFieldsTree(m.mappings.Fields, 0, width-4)...)
	} else {
		fieldLines = append(fieldLines, m.renderFieldsFlat(width-4)...)
	}

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 10
	}
	endIdx := min(m.mappingScroll+maxVisible, len(fieldLines))
	startIdx := m.mappingScroll
	if startIdx >= len(fieldLines) {
		startIdx = max(0, len(fieldLines)-1)
	}

	for i := startIdx; i < endIdx; i++ {
		b.WriteString(fieldLines[i])
		b.WriteString("\n")
	}

	if m.search.Active() {
		b.WriteString(m.search.View(width - 4))
		b.WriteString("\n")
	}

	return paneStyle.Render(b.String())
}

func (m MappingsModel) renderFieldsFlat(width int) []string {
	if m.mappings == nil {
		return nil
	}

	fields := m.flattenMappingFields(m.mappings.Fields)
	var lines []string

	nameWidth := width / 2
	typeWidth := 12

	for _, f := range fields {
		name := Truncate(f.Name, nameWidth)
		fieldType := f.Type
		if fieldType == "" {
			fieldType = "object"
		}

		typeStyle := lipgloss.NewStyle().Foreground(m.typeColor(fieldType))
		attrs := m.formatFieldAttrs(f)

		line := fmt.Sprintf("%-*s %s", nameWidth, name, typeStyle.Render(fmt.Sprintf("%-*s", typeWidth, fieldType)))
		if attrs != "" {
			attrStyle := lipgloss.NewStyle().Foreground(ColorGray)
			line += " " + attrStyle.Render(attrs)
		}
		lines = append(lines, line)
	}

	return lines
}

func (m MappingsModel) renderFieldsTree(fields []es.MappingField, depth int, width int) []string {
	var lines []string
	indent := strings.Repeat("  ", depth)

	for _, f := range fields {
		name := f.Name
		if depth > 0 {
			parts := strings.Split(f.Name, ".")
			name = parts[len(parts)-1]
		}

		fieldType := f.Type
		if fieldType == "" {
			fieldType = "object"
		}

		typeStyle := lipgloss.NewStyle().Foreground(m.typeColor(fieldType))
		attrs := m.formatFieldAttrs(f)

		line := fmt.Sprintf("%s%s: %s", indent, name, typeStyle.Render(fieldType))
		if attrs != "" {
			attrStyle := lipgloss.NewStyle().Foreground(ColorGray)
			line += " " + attrStyle.Render(attrs)
		}
		lines = append(lines, line)

		if len(f.Children) > 0 {
			lines = append(lines, m.renderFieldsTree(f.Children, depth+1, width)...)
		}
	}

	return lines
}

func (m MappingsModel) renderAnalyzers(width int) []string {
	var lines []string

	divider := strings.Repeat("-", width)
	lines = append(lines, divider)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	lines = append(lines, headerStyle.Render("Custom Analyzers"))

	kindOrder := map[string]int{"analyzer": 0, "tokenizer": 1, "filter": 2}
	sorted := make([]es.AnalyzerInfo, len(m.analyzers))
	copy(sorted, m.analyzers)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Kind != sorted[j].Kind {
			return kindOrder[sorted[i].Kind] < kindOrder[sorted[j].Kind]
		}
		return sorted[i].Name < sorted[j].Name
	})

	for _, a := range sorted {
		kindStyle := lipgloss.NewStyle().Foreground(ColorGray)
		nameStyle := lipgloss.NewStyle().Foreground(ColorBlue)

		line := fmt.Sprintf("%s %s", kindStyle.Render(fmt.Sprintf("[%s]", a.Kind)), nameStyle.Render(a.Name))

		var settingParts []string
		for k, v := range a.Settings {
			if k != "type" {
				settingParts = append(settingParts, fmt.Sprintf("%s=%s", k, v))
			}
		}
		if len(settingParts) > 0 {
			sort.Strings(settingParts)
			settingsStyle := lipgloss.NewStyle().Foreground(ColorGray)
			line += " " + settingsStyle.Render(strings.Join(settingParts, ", "))
		}

		lines = append(lines, line)
	}

	return lines
}

func (m MappingsModel) flattenMappingFields(fields []es.MappingField) []es.MappingField {
	var result []es.MappingField
	for _, f := range fields {
		result = append(result, f)
		if len(f.Children) > 0 {
			result = append(result, m.flattenMappingFields(f.Children)...)
		}
	}
	return result
}

func (m *MappingsModel) updateMappingSearch() {
	if m.mappings == nil {
		return
	}
	var lines []string
	if len(m.analyzers) > 0 {
		lines = append(lines, m.renderAnalyzers(1000)...)
		lines = append(lines, "")
	}
	if m.treeView {
		lines = append(lines, m.renderFieldsTree(m.mappings.Fields, 0, 1000)...)
	} else {
		lines = append(lines, m.renderFieldsFlat(1000)...)
	}
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.mappingScroll = match
	}
}

func (m MappingsModel) formatFieldAttrs(f es.MappingField) string {
	var attrs []string
	for k, v := range f.Properties {
		attrs = append(attrs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(attrs)
	return strings.Join(attrs, " ")
}

func (m MappingsModel) typeColor(fieldType string) lipgloss.Color {
	switch fieldType {
	case "keyword", "text":
		return ColorGreen
	case "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float":
		return ColorBlue
	case "date", "date_nanos":
		return ColorYellow
	case "boolean":
		return ColorRed
	case "object", "nested":
		return ColorGray
	default:
		return ColorWhite
	}
}

