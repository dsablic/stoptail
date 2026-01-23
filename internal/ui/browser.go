package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type BrowserPane int

const (
	BrowserPaneIndices BrowserPane = iota
	BrowserPaneDocs
	BrowserPaneDetail

	browserPageSize = 100
)

type BrowserModel struct {
	client        *es.Client
	indices       []es.IndexInfo
	selectedIndex int
	indexScroll   int
	filterActive  bool
	filterText    string

	documents   []es.DocumentHit
	selectedDoc int
	docScroll   int
	loading     bool
	hasMore     bool
	searchAfter []any
	total       int64

	detail     viewport.Model
	activePane BrowserPane
	clipboard  Clipboard

	width  int
	height int
}

type browserSearchMsg struct {
	result *es.SearchResult
	err    error
	append bool
}

func NewBrowser() BrowserModel {
	vp := viewport.New(40, 10)
	return BrowserModel{
		activePane: BrowserPaneIndices,
		detail:     vp,
		clipboard:  NewClipboard(),
		hasMore:    true,
	}
}

func (m *BrowserModel) SetClient(client *es.Client) {
	m.client = client
}

func (m *BrowserModel) SetIndices(indices []es.IndexInfo) {
	m.indices = indices
	if m.selectedIndex >= len(indices) {
		m.selectedIndex = max(0, len(indices)-1)
	}
}

func (m *BrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	detailWidth := (width - 4) / 3
	detailHeight := height - 7
	m.detail.Width = detailWidth
	m.detail.Height = detailHeight
}

func (m BrowserModel) filteredIndices() []es.IndexInfo {
	if m.filterText == "" {
		return m.indices
	}
	var filtered []es.IndexInfo
	lower := strings.ToLower(m.filterText)
	for _, idx := range m.indices {
		if strings.Contains(strings.ToLower(idx.Name), lower) {
			filtered = append(filtered, idx)
		}
	}
	return filtered
}

func (m BrowserModel) selectedIndexName() string {
	filtered := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(filtered) {
		return filtered[m.selectedIndex].Name
	}
	return ""
}

func (m BrowserModel) HasActiveInput() bool {
	return m.filterActive
}

func (m BrowserModel) ClipboardMessage() string {
	return m.clipboard.Message()
}

func (m BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case browserSearchMsg:
		m.loading = false
		if msg.err != nil {
			return m, nil
		}
		if msg.append {
			m.documents = append(m.documents, msg.result.Hits...)
		} else {
			m.documents = msg.result.Hits
			m.selectedDoc = 0
			m.docScroll = 0
		}
		m.total = msg.result.Total
		m.hasMore = len(m.documents) < int(m.total)
		if len(msg.result.Hits) > 0 {
			m.searchAfter = msg.result.Hits[len(msg.result.Hits)-1].Sort
		}
		m.updateDetailPane()
		return m, nil

	case tea.KeyMsg:
		m.clipboard.ClearMessage()

		if m.filterActive {
			return m.handleFilterInput(msg)
		}

		switch msg.String() {
		case "/":
			if m.activePane == BrowserPaneIndices {
				m.filterActive = true
				m.filterText = ""
			}
		case "left", "h":
			if m.activePane > BrowserPaneIndices {
				m.activePane--
			}
		case "right", "l":
			if m.activePane < BrowserPaneDetail {
				m.activePane++
			}
		case "enter":
			if m.activePane == BrowserPaneIndices {
				m.activePane = BrowserPaneDocs
				return m, m.fetchDocuments(false)
			}
		case "up", "k":
			m.handleUp()
		case "down", "j":
			cmd := m.handleDown()
			if cmd != nil {
				return m, cmd
			}
		case "ctrl+y":
			if m.activePane == BrowserPaneDetail && len(m.documents) > 0 {
				m.clipboard.Copy(m.selectedDocSource())
			}
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			scrollAmount := 3
			switch m.activePane {
			case BrowserPaneIndices:
				maxScroll := m.maxIndexScroll()
				if msg.Button == tea.MouseButtonWheelUp {
					m.indexScroll = max(0, m.indexScroll-scrollAmount)
				} else {
					m.indexScroll = min(maxScroll, m.indexScroll+scrollAmount)
				}
			case BrowserPaneDocs:
				if msg.Button == tea.MouseButtonWheelUp {
					m.docScroll = max(0, m.docScroll-scrollAmount)
				} else {
					cmd := m.scrollDocsDown(scrollAmount)
					if cmd != nil {
						return m, cmd
					}
				}
			case BrowserPaneDetail:
				if msg.Button == tea.MouseButtonWheelUp {
					m.detail.ScrollUp(scrollAmount)
				} else {
					m.detail.ScrollDown(scrollAmount)
				}
			}
		}
	}

	return m, nil
}

func (m BrowserModel) handleFilterInput(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.filterActive = false
		m.selectedIndex = 0
		m.indexScroll = 0
		if msg.String() == "enter" {
			return m, m.fetchDocuments(false)
		}
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.filterText += msg.String()
			m.selectedIndex = 0
			m.indexScroll = 0
		}
	}
	return m, nil
}

func (m *BrowserModel) handleUp() {
	switch m.activePane {
	case BrowserPaneIndices:
		if m.selectedIndex > 0 {
			m.selectedIndex--
			if m.selectedIndex < m.indexScroll {
				m.indexScroll = m.selectedIndex
			}
		}
	case BrowserPaneDocs:
		if m.selectedDoc > 0 {
			m.selectedDoc--
			if m.selectedDoc < m.docScroll {
				m.docScroll = m.selectedDoc
			}
			m.updateDetailPane()
		}
	case BrowserPaneDetail:
		m.detail.ScrollUp(1)
	}
}

func (m *BrowserModel) handleDown() tea.Cmd {
	switch m.activePane {
	case BrowserPaneIndices:
		filtered := m.filteredIndices()
		if m.selectedIndex < len(filtered)-1 {
			m.selectedIndex++
			maxVisible := m.maxVisibleIndices()
			if m.selectedIndex >= m.indexScroll+maxVisible {
				m.indexScroll = m.selectedIndex - maxVisible + 1
			}
		}
	case BrowserPaneDocs:
		if m.selectedDoc < len(m.documents)-1 {
			m.selectedDoc++
			maxVisible := m.maxVisibleDocs()
			if m.selectedDoc >= m.docScroll+maxVisible {
				m.docScroll = m.selectedDoc - maxVisible + 1
			}
			m.updateDetailPane()

			if m.shouldLoadMore() {
				return m.fetchDocuments(true)
			}
		}
	case BrowserPaneDetail:
		m.detail.ScrollDown(1)
	}
	return nil
}

func (m *BrowserModel) scrollDocsDown(amount int) tea.Cmd {
	maxScroll := m.maxDocScroll()
	m.docScroll = min(maxScroll, m.docScroll+amount)

	if m.shouldLoadMore() {
		return m.fetchDocuments(true)
	}
	return nil
}

func (m BrowserModel) shouldLoadMore() bool {
	if !m.hasMore || m.loading {
		return false
	}
	return m.docScroll+m.maxVisibleDocs() >= len(m.documents)-5
}

func (m *BrowserModel) updateDetailPane() {
	if m.selectedDoc < 0 || m.selectedDoc >= len(m.documents) {
		m.detail.SetContent("")
		return
	}

	doc := m.documents[m.selectedDoc]
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(doc.Source), "", "  "); err == nil {
		m.detail.SetContent(highlightJSON(pretty.String()))
	} else {
		m.detail.SetContent(doc.Source)
	}
	m.detail.GotoTop()
}

func (m BrowserModel) selectedDocSource() string {
	if m.selectedDoc < 0 || m.selectedDoc >= len(m.documents) {
		return ""
	}
	doc := m.documents[m.selectedDoc]
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(doc.Source), "", "  "); err == nil {
		return pretty.String()
	}
	return doc.Source
}

func (m BrowserModel) fetchDocuments(append bool) tea.Cmd {
	index := m.selectedIndexName()
	if index == "" || m.client == nil {
		return nil
	}

	m.loading = true
	var after []any
	if append {
		after = m.searchAfter
	} else {
		m.documents = nil
		m.searchAfter = nil
		m.hasMore = true
	}

	return func() tea.Msg {
		result, err := m.client.SearchDocuments(context.Background(), index, after, browserPageSize)
		return browserSearchMsg{result: result, err: err, append: append}
	}
}

func (m BrowserModel) maxVisibleIndices() int {
	return m.height - 6
}

func (m BrowserModel) maxIndexScroll() int {
	filtered := m.filteredIndices()
	maxScroll := len(filtered) - m.maxVisibleIndices()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m BrowserModel) maxVisibleDocs() int {
	return m.height - 6
}

func (m BrowserModel) maxDocScroll() int {
	maxScroll := len(m.documents) - m.maxVisibleDocs()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m BrowserModel) View() string {
	leftWidth := m.width / 4
	middleWidth := m.width / 3
	rightWidth := m.width - leftWidth - middleWidth - 4

	leftPane := m.renderIndexList(leftWidth)
	middlePane := m.renderDocList(middleWidth)
	rightPane := m.renderDetailPane(rightWidth)

	leftLines := strings.Split(leftPane, "\n")
	middleLines := strings.Split(middlePane, "\n")
	rightLines := strings.Split(rightPane, "\n")

	maxLines := max(len(leftLines), max(len(middleLines), len(rightLines)))

	var lines []string
	for i := 0; i < maxLines; i++ {
		ll := ""
		ml := ""
		rl := ""
		if i < len(leftLines) {
			ll = TrimANSI(leftLines[i])
		}
		if i < len(middleLines) {
			ml = TrimANSI(middleLines[i])
		}
		if i < len(rightLines) {
			rl = TrimANSI(rightLines[i])
		}
		lines = append(lines, ll+" "+ml+" "+rl)
	}

	return strings.Join(lines, "\n")
}

func (m BrowserModel) renderIndexList(width int) string {
	borderColor := ColorGray
	if m.activePane == BrowserPaneIndices {
		borderColor = ColorBlue
	}

	filtered := m.filteredIndices()
	innerWidth := width - 4

	var content strings.Builder
	header := "Indices"
	if m.filterActive {
		header = fmt.Sprintf("/%s_", m.filterText)
	} else if m.filterText != "" {
		header = fmt.Sprintf("Indices [%s]", m.filterText)
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	content.WriteString("\n")

	maxVisible := m.maxVisibleIndices()
	for i := m.indexScroll; i < len(filtered) && i < m.indexScroll+maxVisible; i++ {
		idx := filtered[i]
		name := Truncate(idx.Name, innerWidth-2)

		style := lipgloss.NewStyle()
		if i == m.selectedIndex {
			style = style.Background(ActiveBg).Foreground(ColorWhite)
			name = "> " + name
		} else {
			name = "  " + name
		}

		content.WriteString(style.Render(fmt.Sprintf("%-*s", innerWidth, name)))
		content.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(m.height - 4).
		Render(content.String())
}

func (m BrowserModel) renderDocList(width int) string {
	borderColor := ColorGray
	if m.activePane == BrowserPaneDocs {
		borderColor = ColorBlue
	}

	innerWidth := width - 4

	var content strings.Builder
	header := "Documents"
	if m.total > 0 {
		header = fmt.Sprintf("Documents (%d/%d)", len(m.documents), m.total)
	}
	if m.loading {
		header += " ..."
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	content.WriteString("\n")

	if len(m.documents) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render("Select an index"))
		content.WriteString("\n")
	} else {
		idWidth := innerWidth / 3
		previewWidth := innerWidth - idWidth - 2

		maxVisible := m.maxVisibleDocs()
		for i := m.docScroll; i < len(m.documents) && i < m.docScroll+maxVisible; i++ {
			doc := m.documents[i]
			id := Truncate(doc.ID, idWidth)
			preview := Truncate(doc.Source, previewWidth)

			style := lipgloss.NewStyle()
			prefix := "  "
			if i == m.selectedDoc {
				style = style.Background(ActiveBg).Foreground(ColorWhite)
				prefix = "> "
			}

			line := fmt.Sprintf("%s%-*s %s", prefix, idWidth, id,
				lipgloss.NewStyle().Foreground(ColorGray).Render(preview))
			content.WriteString(style.Render(fmt.Sprintf("%-*s", innerWidth, line)))
			content.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(m.height - 4).
		Render(content.String())
}

func (m BrowserModel) renderDetailPane(width int) string {
	borderColor := ColorGray
	if m.activePane == BrowserPaneDetail {
		borderColor = ColorBlue
	}

	var content strings.Builder
	header := "Document"
	if m.selectedDoc >= 0 && m.selectedDoc < len(m.documents) {
		header = fmt.Sprintf("Document: %s", Truncate(m.documents[m.selectedDoc].ID, width-15))
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	content.WriteString("\n")
	content.WriteString(m.detail.View())

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(m.height - 4).
		Render(content.String())
}
