package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/labtiva/stoptail/internal/es"
)

type MappingsPane int

const (
	PaneIndices MappingsPane = iota
	PaneMappings
)

type MappingsViewMode int

const (
	ViewMappings MappingsViewMode = iota
	ViewSettings
)

type MappingsModel struct {
	indices    []es.IndexInfo
	indexNav   ListNav
	contentNav ListNav
	width      int
	height     int
	activePane MappingsPane
	filterActive  bool
	filterText    string
	treeView      bool
	search        SearchBar
	viewMode      MappingsViewMode

	mappings  *es.IndexMappings
	analyzers []es.AnalyzerInfo
	loadingIndex  string
	clipboard     Clipboard

	settings        *es.IndexSettings
	settingsLoading bool
}

type fetchMappingsMsg struct {
	indexName string
}

type fetchSettingsMsg struct {
	indexName string
}

func NewMappings() MappingsModel {
	return MappingsModel{
		indexNav:   NewCursorNav(),
		contentNav: NewScrollNav(),
		activePane: PaneIndices,
		search:     NewSearchBar(),
		clipboard:  NewClipboard(),
	}
}

func (m *MappingsModel) SetIndices(indices []es.IndexInfo) {
	m.indices = indices
	if m.indexNav.Selected >= len(indices) {
		m.indexNav.Selected = max(0, len(indices)-1)
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
	m.contentNav.Reset()
}

func (m *MappingsModel) SetSettings(settings *es.IndexSettings) {
	m.settings = settings
	m.settingsLoading = false
	m.contentNav.Reset()
}

func (m MappingsModel) SelectedIndexName() string {
	filtered := m.filteredIndices()
	if m.indexNav.Selected >= 0 && m.indexNav.Selected < len(filtered) {
		return filtered[m.indexNav.Selected].Name
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
	case tea.KeyPressMsg:
		m.clipboard.ClearMessage()
		if m.search.Active() {
			cmd, action := m.search.HandleKey(msg)
			switch action {
			case SearchActionClose:
				// search deactivated
			case SearchActionNext, SearchActionPrev:
				if match := m.search.CurrentMatch(); match >= 0 {
					m.contentNav.Scroll = match
				}
			case SearchActionNone:
				(&m).updateMappingSearch()
			}
			return m, cmd
		}

		if m.filterActive {
			return m.handleFilterInput(msg)
		}

		switch msg.String() {
		case "ctrl+f":
			if m.activePane == PaneMappings {
				m.search.Activate()
				return m, nil
			}
		case "ctrl+y":
			return m, m.clipboard.Copy(m.copyableContent())
		case "/":
			if m.activePane == PaneIndices {
				m.filterActive = true
				m.filterText = ""
			}
		case "t":
			if m.activePane == PaneMappings && m.viewMode == ViewMappings {
				m.treeView = !m.treeView
			}
		case "s":
			if m.activePane == PaneMappings {
				if m.viewMode == ViewMappings {
					m.viewMode = ViewSettings
					return m, m.fetchSettingsCmd()
				} else {
					m.viewMode = ViewMappings
				}
			}
		case "left", "h":
			m.activePane = PaneIndices
		case "right", "l":
			if m.activePane == PaneIndices {
				m.activePane = PaneMappings
				if m.viewMode == ViewSettings {
					return m, m.fetchSettingsCmd()
				}
				return m, m.fetchMappingsCmd()
			}
		case "enter":
			if m.activePane == PaneIndices {
				m.activePane = PaneMappings
				if m.viewMode == ViewSettings {
					return m, m.fetchSettingsCmd()
				}
				return m, m.fetchMappingsCmd()
			}
		default:
			if m.activePane == PaneIndices {
				m.indexNav.HandleKey(msg.String(), len(m.filteredIndices()), m.maxVisibleIndices())
			} else {
				m.contentNav.HandleKey(msg.String(), m.countMappingLines(), m.mappingVisibleHeight())
			}
		}
	case tea.MouseWheelMsg:
		if m.activePane == PaneIndices {
			m.indexNav.HandleWheel(msg.Button != tea.MouseWheelUp, len(m.filteredIndices()), m.maxVisibleIndices())
		} else {
			m.contentNav.HandleWheel(msg.Button != tea.MouseWheelUp, m.countMappingLines(), m.mappingVisibleHeight())
		}
	}
	return m, nil
}

func (m MappingsModel) handleFilterInput(msg tea.KeyPressMsg) (MappingsModel, tea.Cmd) {
	text, action := HandleFilterKey(m.filterText, msg.String())
	m.filterText = text
	if action == FilterClose || action == FilterConfirm {
		m.filterActive = false
	}
	m.indexNav.Reset()
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

func (m MappingsModel) fetchSettingsCmd() tea.Cmd {
	indexName := m.SelectedIndexName()
	if indexName == "" {
		return nil
	}
	return func() tea.Msg {
		return fetchSettingsMsg{indexName: indexName}
	}
}

func (m MappingsModel) filteredIndices() []es.IndexInfo {
	if m.filterText == "" {
		return m.indices
	}
	var filtered []es.IndexInfo
	for _, idx := range m.indices {
		if MatchesFilter(idx.Name, m.filterText) {
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

func (m MappingsModel) mappingVisibleHeight() int {
	h := m.height - 6
	if h < 1 {
		return 10
	}
	return h
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
	var rightPane string
	if m.viewMode == ViewSettings {
		rightPane = m.renderSettingsPane(rightWidth)
	} else {
		rightPane = m.renderMappingsPane(rightWidth)
	}

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
		return paneStyle.Render(strings.TrimRight(b.String(), "\n"))
	}

	maxVisible := m.maxVisibleIndices()
	if m.filterActive || m.filterText != "" {
		maxVisible--
	}
	endIdx := min(m.indexNav.Scroll+maxVisible, len(filtered))

	for i := m.indexNav.Scroll; i < endIdx; i++ {
		idx := filtered[i]
		isSelected := i == m.indexNav.Selected

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

	return paneStyle.Render(strings.TrimRight(b.String(), "\n"))
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
	endIdx := min(m.contentNav.Scroll+maxVisible, len(fieldLines))
	startIdx := m.contentNav.Scroll
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

	return paneStyle.Render(strings.TrimRight(b.String(), "\n"))
}

func (m MappingsModel) renderSettingsPane(width int) string {
	borderColor := ColorGray
	if m.activePane == PaneMappings {
		borderColor = ColorBlue
	}

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(m.height - 2)

	if m.settingsLoading {
		return paneStyle.Render("Loading settings...")
	}

	if m.settings == nil {
		hint := lipgloss.NewStyle().Foreground(ColorGray).Render("Press s to load settings")
		return paneStyle.Render(hint)
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	b.WriteString(headerStyle.Render("Settings: " + m.settings.IndexName))
	b.WriteString("\n")

	toggleHint := lipgloss.NewStyle().Foreground(ColorGray).Render("[s: switch to mappings]")
	b.WriteString(toggleHint)
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	settings := []struct {
		label string
		value string
	}{
		{"Shards", m.settings.NumberOfShards},
		{"Replicas", m.settings.NumberOfReplicas},
		{"Refresh Interval", m.settings.RefreshInterval},
		{"Codec", m.settings.Codec},
		{"Created", m.settings.CreationDate},
		{"UUID", m.settings.UUID},
		{"Version", m.settings.Version},
	}

	if m.settings.RoutingAllocation != "" {
		settings = append(settings, struct {
			label string
			value string
		}{"Routing Allocation", m.settings.RoutingAllocation})
	}

	labelWidth := 18
	for _, s := range settings {
		if s.value != "" {
			b.WriteString(labelStyle.Render(fmt.Sprintf("%-*s", labelWidth, s.label+":")))
			b.WriteString(valueStyle.Render(s.value))
			b.WriteString("\n")
		}
	}

	if len(m.settings.AllSettings) > 0 {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("All Settings"))
		b.WriteString("\n")

		var keys []string
		for k := range m.settings.AllSettings {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		maxVisible := max(m.height-15, 5)
		endIdx := min(m.contentNav.Scroll+maxVisible, len(keys))
		startIdx := m.contentNav.Scroll
		if startIdx >= len(keys) {
			startIdx = max(0, len(keys)-1)
		}

		for i := startIdx; i < endIdx; i++ {
			k := keys[i]
			v := m.settings.AllSettings[k]
			innerWidth := width - 2
			keyTrunc := Truncate(k, innerWidth/2)
			valTrunc := Truncate(v, innerWidth/2-2)
			b.WriteString(labelStyle.Render(keyTrunc + ": "))
			b.WriteString(valueStyle.Render(valTrunc))
			b.WriteString("\n")
		}
	}

	return paneStyle.Render(strings.TrimRight(b.String(), "\n"))
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
			remaining := width - lipgloss.Width(line) - 1
			if remaining > 0 {
				attrStyle := lipgloss.NewStyle().Foreground(ColorGray)
				line += " " + attrStyle.Render(Truncate(attrs, remaining))
			}
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
			remaining := width - lipgloss.Width(line) - 1
			if remaining > 0 {
				attrStyle := lipgloss.NewStyle().Foreground(ColorGray)
				line += " " + attrStyle.Render(Truncate(attrs, remaining))
			}
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

	kindOrder := map[string]int{"analyzer": 0, "tokenizer": 1, "filter": 2, "normalizer": 3}
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
			remaining := width - lipgloss.Width(line) - 1
			if remaining > 0 {
				settingsStyle := lipgloss.NewStyle().Foreground(ColorGray)
				line += " " + settingsStyle.Render(Truncate(strings.Join(settingParts, ", "), remaining))
			}
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

func (m MappingsModel) copyableContent() string {
	if m.mappings == nil {
		return ""
	}
	var lines []string
	if m.treeView {
		lines = m.copyableTree(m.mappings.Fields, 0)
	} else {
		for _, f := range m.flattenMappingFields(m.mappings.Fields) {
			lines = append(lines, fmt.Sprintf("%s: %s", f.Name, f.Type))
		}
	}
	return strings.Join(lines, "\n")
}

func (m MappingsModel) copyableTree(fields []es.MappingField, depth int) []string {
	var lines []string
	indent := strings.Repeat("  ", depth)
	for _, f := range fields {
		if f.Type == "object" || f.Type == "nested" {
			lines = append(lines, fmt.Sprintf("%s%s (%s)", indent, f.Name, f.Type))
			lines = append(lines, m.copyableTree(f.Children, depth+1)...)
		} else {
			lines = append(lines, fmt.Sprintf("%s%s: %s", indent, f.Name, f.Type))
		}
	}
	return lines
}

func (m MappingsModel) ClipboardMessage() string {
	return m.clipboard.Message()
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
		m.contentNav.Scroll = match
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

func (m MappingsModel) typeColor(fieldType string) color.Color {
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

