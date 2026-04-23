package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	client       *es.Client
	indices      []es.IndexInfo
	indexNav     ListNav
	filterActive bool
	filterText   string

	documents   []es.DocumentHit
	docNav      ListNav
	loading     bool
	hasMore     bool
	searchAfter []any
	total       int64

	detailSourceLines []string
	detailLines       []string
	detailNav         ListNav
	detailHeight      int
	activePane    BrowserPane
	clipboard     Clipboard

	width  int
	height int
}

type browserSearchMsg struct {
	result *es.SearchResult
	err    error
	append bool
}

func NewBrowser() BrowserModel {
	return BrowserModel{
		activePane: BrowserPaneIndices,
		indexNav:   NewCursorNav(),
		docNav:     NewCursorNav(),
		detailNav:  NewScrollNav(),
		clipboard:  NewClipboard(),
		hasMore:    true,
	}
}

func (m *BrowserModel) SetClient(client *es.Client) {
	m.client = client
}

func (m *BrowserModel) SetIndices(indices []es.IndexInfo) {
	m.indices = indices
	if m.indexNav.Selected >= len(indices) {
		m.indexNav.Selected = max(0, len(indices)-1)
	}
}

func (m *BrowserModel) SetSize(width, height int) {
	oldWidth := m.width
	m.width = width
	m.height = height
	m.detailHeight = height - 7
	if width != oldWidth && len(m.detailSourceLines) > 0 {
		m.rewrapDetailPane()
	}
}

func (m *BrowserModel) SelectIndexByName(name string) bool {
	for i, idx := range m.indices {
		if idx.Name == name {
			m.indexNav.Selected = i
			m.activePane = BrowserPaneDocs
			return true
		}
	}
	return false
}

func (m *BrowserModel) LoadDocumentsSync(ctx context.Context) error {
	index := m.selectedIndexName()
	if index == "" || m.client == nil {
		return nil
	}
	result, err := m.client.SearchDocuments(ctx, index, nil, browserPageSize)
	if err != nil {
		return err
	}
	m.documents = result.Hits
	m.total = result.Total
	m.hasMore = len(m.documents) < int(m.total)
	if len(result.Hits) > 0 {
		m.searchAfter = result.Hits[len(result.Hits)-1].Sort
	}
	m.updateDetailPane()
	return nil
}

func (m BrowserModel) filteredIndices() []es.IndexInfo {
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

func (m BrowserModel) selectedIndexName() string {
	filtered := m.filteredIndices()
	if m.indexNav.Selected >= 0 && m.indexNav.Selected < len(filtered) {
		return filtered[m.indexNav.Selected].Name
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
			m.docNav.Reset()
		}
		m.total = msg.result.Total
		m.hasMore = len(m.documents) < int(m.total)
		if len(msg.result.Hits) > 0 {
			m.searchAfter = msg.result.Hits[len(msg.result.Hits)-1].Sort
		}
		m.updateDetailPane()
		return m, nil

	case tea.KeyPressMsg:
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
				return m, m.startFetchDocuments(false)
			}
		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			nav, total, visible := m.activeNavParams()
			nav.HandleKey(msg.String(), total, visible)
			if m.activePane == BrowserPaneDocs {
				m.updateDetailPane()
				if m.shouldLoadMore() {
					return m, m.startFetchDocuments(true)
				}
			}
		case "ctrl+y":
			if m.activePane == BrowserPaneDetail && len(m.documents) > 0 {
				return m, m.clipboard.Copy(m.selectedDocSource())
			}
		}

	case tea.MouseWheelMsg:
		down := msg.Button != tea.MouseWheelUp
		visible := m.maxVisibleDocs()
		pane := m.paneAtX(msg.X)
		switch pane {
		case BrowserPaneIndices:
			m.indexNav.HandleWheel(down, len(m.filteredIndices()), visible)
		case BrowserPaneDocs:
			m.docNav.HandleWheel(down, len(m.documents), visible)
			if m.shouldLoadMore() {
				return m, m.startFetchDocuments(true)
			}
		case BrowserPaneDetail:
			m.detailNav.HandleWheel(down, len(m.detailLines), m.detailHeight)
		}
	}

	return m, nil
}

func (m BrowserModel) handleFilterInput(msg tea.KeyPressMsg) (BrowserModel, tea.Cmd) {
	text, action := HandleFilterKey(m.filterText, msg.String())
	m.filterText = text
	if action == FilterClose || action == FilterConfirm {
		m.filterActive = false
		m.indexNav.Reset()
		if action == FilterConfirm {
			return m, m.startFetchDocuments(false)
		}
	} else {
		m.indexNav.Reset()
	}
	return m, nil
}

func (m *BrowserModel) activeNavParams() (*ListNav, int, int) {
	visible := m.maxVisibleDocs()
	switch m.activePane {
	case BrowserPaneIndices:
		return &m.indexNav, len(m.filteredIndices()), visible
	case BrowserPaneDocs:
		return &m.docNav, len(m.documents), visible
	default:
		return &m.detailNav, len(m.detailLines), m.detailHeight
	}
}

func (m BrowserModel) shouldLoadMore() bool {
	if !m.hasMore || m.loading {
		return false
	}
	return m.docNav.Scroll+m.maxVisibleDocs() >= len(m.documents)-5
}

func (m *BrowserModel) updateDetailPane() {
	if m.docNav.Selected < 0 || m.docNav.Selected >= len(m.documents) {
		m.detailSourceLines = nil
		m.detailLines = nil
		m.detailNav.Scroll = 0
		return
	}

	doc := m.documents[m.docNav.Selected]
	var sourceLines []string
	var obj any
	if err := json.Unmarshal([]byte(doc.Source), &obj); err == nil {
		if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
			sanitized := SanitizeForTerminal(string(pretty))
			highlighted := highlightJSON(sanitized)
			sourceLines = strings.Split(highlighted, "\n")
		}
	}
	if sourceLines == nil {
		sourceLines = []string{SanitizeForTerminal(doc.Source)}
	}

	m.detailSourceLines = sourceLines
	m.rewrapDetailPane()
	m.detailNav.Scroll = 0
}

func (m *BrowserModel) rewrapDetailPane() {
	innerWidth := m.detailInnerWidth()
	joined := strings.Join(m.detailSourceLines, "\n")
	wrapped := ansi.Hardwrap(joined, innerWidth, false)
	m.detailLines = strings.Split(wrapped, "\n")
}

func (m BrowserModel) paneWidths() (left, middle, right int) {
	left = m.width / 4
	middle = m.width / 3
	right = m.width - left - middle - 4
	return
}

func (m BrowserModel) detailInnerWidth() int {
	_, _, right := m.paneWidths()
	return right - 4
}

func (m BrowserModel) selectedDocSource() string {
	if m.docNav.Selected < 0 || m.docNav.Selected >= len(m.documents) {
		return ""
	}
	doc := m.documents[m.docNav.Selected]
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(doc.Source), "", "  "); err == nil {
		return pretty.String()
	}
	return doc.Source
}

func (m *BrowserModel) startFetchDocuments(appendDocs bool) tea.Cmd {
	index := m.selectedIndexName()
	if index == "" || m.client == nil {
		return nil
	}

	m.loading = true
	var after []any
	if appendDocs {
		after = m.searchAfter
	} else {
		m.documents = nil
		m.searchAfter = nil
		m.hasMore = true
	}

	client := m.client
	return func() tea.Msg {
		result, err := client.SearchDocuments(context.Background(), index, after, browserPageSize)
		return browserSearchMsg{result: result, err: err, append: appendDocs}
	}
}

func (m BrowserModel) paneAtX(x int) BrowserPane {
	leftWidth, middleWidth, _ := m.paneWidths()
	if x < leftWidth+1 {
		return BrowserPaneIndices
	}
	if x < leftWidth+middleWidth+2 {
		return BrowserPaneDocs
	}
	return BrowserPaneDetail
}

func (m BrowserModel) maxVisibleIndices() int {
	return m.height - 7
}

func (m BrowserModel) maxVisibleDocs() int {
	return m.height - 7
}

func (m BrowserModel) View() string {
	if m.height == 0 || m.width == 0 {
		return ""
	}

	leftWidth, middleWidth, rightWidth := m.paneWidths()

	leftPane := m.renderIndexList(leftWidth)
	middlePane := m.renderDocList(middleWidth)
	rightPane := m.renderDetailPane(rightWidth)

	return JoinPanesHorizontal(m.height-2, leftPane, middlePane, rightPane)
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
	for i := m.indexNav.Scroll; i < len(filtered) && i < m.indexNav.Scroll+maxVisible; i++ {
		idx := filtered[i]
		name := Truncate(idx.Name, innerWidth-2)

		style := lipgloss.NewStyle()
		prefix := "  "
		if i == m.indexNav.Selected {
			style = style.Background(ActiveBg).Foreground(ColorWhite)
			prefix = "> "
		}

		line := fmt.Sprintf("%s%-*s", prefix, innerWidth-2, name)
		content.WriteString(style.Render(line))
		content.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(m.height - 4).
		Render(strings.TrimRight(content.String(), "\n"))
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
		previewWidth := innerWidth - idWidth - 3

		maxVisible := m.maxVisibleDocs()
		for i := m.docNav.Scroll; i < len(m.documents) && i < m.docNav.Scroll+maxVisible; i++ {
			doc := m.documents[i]
			id := Truncate(doc.ID, idWidth)
			preview := Truncate(sanitizeLine(doc.Source), previewWidth)

			style := lipgloss.NewStyle()
			prefix := "  "
			if i == m.docNav.Selected {
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
		Render(strings.TrimRight(content.String(), "\n"))
}

func (m BrowserModel) renderDetailPane(width int) string {
	borderColor := ColorGray
	if m.activePane == BrowserPaneDetail {
		borderColor = ColorBlue
	}

	var content strings.Builder
	header := "Document"
	if m.docNav.Selected >= 0 && m.docNav.Selected < len(m.documents) {
		header = fmt.Sprintf("Document: %s", Truncate(m.documents[m.docNav.Selected].ID, width-15))
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	content.WriteString("\n")

	boxInnerHeight := m.height - 4
	visibleLines := max(0, boxInnerHeight-3)
	for i := m.detailNav.Scroll; i < len(m.detailLines) && i-m.detailNav.Scroll < visibleLines; i++ {
		content.WriteString(m.detailLines[i])
		content.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(m.height - 4).
		Render(strings.TrimRight(content.String(), "\n"))
}

func sanitizeLine(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 32 && r < 127 {
			b.WriteRune(r)
		} else {
			quoted := strconv.QuoteRuneToASCII(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		}
	}
	return b.String()
}

