package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
	"github.com/labtiva/stoptail/internal/storage"
)

type WorkbenchFocus int

const (
	FocusMethod WorkbenchFocus = iota
	FocusPath
	FocusBody
	FocusResponse
)

var methods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}

type WorkbenchModel struct {
	client       *es.Client
	methodIdx    int
	path         textinput.Model
	body         textarea.Model
	response     viewport.Model
	responseText string
	statusCode   int
	duration     string
	focus        WorkbenchFocus
	width        int
	height       int
	executing    bool
	err          error
	history      *storage.History
	historyIdx   int
}

type executeResultMsg struct {
	result es.RequestResult
}

func NewWorkbench() WorkbenchModel {
	path := textinput.New()
	path.Placeholder = "/_search"
	path.CharLimit = 200
	path.Width = 40

	body := textarea.New()
	body.Placeholder = `{
  "query": {
    "match_all": {}
  }
}`
	body.CharLimit = 50000
	body.ShowLineNumbers = false

	vp := viewport.New(40, 10)

	history, _ := storage.LoadHistory()

	return WorkbenchModel{
		methodIdx:  0,
		path:       path,
		body:       body,
		response:   vp,
		focus:      FocusPath,
		history:    history,
		historyIdx: -1,
	}
}

func (m *WorkbenchModel) SetClient(client *es.Client) {
	m.client = client
}

func (m *WorkbenchModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Split panes
	paneWidth := (width - 3) / 2 // -3 for divider and padding
	bodyHeight := height - 6      // -6 for method/path row and status

	m.path.Width = paneWidth - 10 // -10 for method selector
	m.body.SetWidth(paneWidth)
	m.body.SetHeight(bodyHeight)
	m.response.Width = paneWidth
	m.response.Height = bodyHeight
}

func (m *WorkbenchModel) Prefill(index string) {
	m.methodIdx = 0 // GET
	m.path.SetValue("/" + index + "/_search")
	m.body.SetValue("{}")
}

func (m *WorkbenchModel) Focus() {
	m.path.Focus()
	m.focus = FocusPath
}

func (m *WorkbenchModel) Blur() {
	m.path.Blur()
	m.body.Blur()
}

func (m WorkbenchModel) isValidJSON() bool {
	val := m.body.Value()
	if val == "" {
		return true
	}
	return json.Valid([]byte(val))
}

func (m WorkbenchModel) Update(msg tea.Msg) (WorkbenchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case executeResultMsg:
		m.executing = false
		if msg.result.Error != nil {
			m.err = msg.result.Error
			m.responseText = fmt.Sprintf("Error: %v", msg.result.Error)
		} else {
			m.err = nil
			m.statusCode = msg.result.StatusCode
			m.duration = msg.result.Duration.String()
			m.responseText = highlightJSON(msg.result.Body)
			if msg.result.StatusCode < 400 {
				m.addToHistory()
			}
		}
		m.response.SetContent(m.responseText)
		m.response.GotoTop()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+m":
			m.methodIdx = (m.methodIdx + 1) % len(methods)
			return m, nil
		case "ctrl+enter", "ctrl+e":
			if m.client != nil && !m.executing {
				m.executing = true
				return m, m.execute()
			}
		case "ctrl+l":
			m.body.SetValue("")
			return m, nil
		case "ctrl+p":
			m.prettyPrintBody()
			return m, nil
		case "ctrl+up":
			m.historyPrev()
			return m, nil
		case "ctrl+down":
			m.historyNext()
			return m, nil
		case "tab":
			m.cycleFocus()
			return m, nil
		case "esc":
			m.path.Blur()
			m.body.Blur()
			return m, nil
		}

		// Delegate to focused component
		switch m.focus {
		case FocusPath:
			m.path, cmd = m.path.Update(msg)
			cmds = append(cmds, cmd)
		case FocusBody:
			m.body, cmd = m.body.Update(msg)
			cmds = append(cmds, cmd)
		case FocusResponse:
			m.response, cmd = m.response.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *WorkbenchModel) cycleFocus() {
	m.path.Blur()
	m.body.Blur()

	m.focus = (m.focus + 1) % 4
	switch m.focus {
	case FocusMethod:
		// No component to focus
	case FocusPath:
		m.path.Focus()
	case FocusBody:
		m.body.Focus()
	case FocusResponse:
		// Viewport doesn't need focus call
	}
}

func (m *WorkbenchModel) prettyPrintBody() {
	val := m.body.Value()
	if val == "" {
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(val), "", "  "); err == nil {
		m.body.SetValue(pretty.String())
	}
}

func (m *WorkbenchModel) addToHistory() {
	entry := storage.HistoryEntry{
		Method: methods[m.methodIdx],
		Path:   m.path.Value(),
		Body:   m.body.Value(),
	}

	if m.history.Add(entry) {
		_ = storage.SaveHistory(m.history)
	}
	m.historyIdx = -1
}

func (m *WorkbenchModel) historyPrev() {
	if m.history == nil || len(m.history.Entries) == 0 {
		return
	}

	if m.historyIdx == -1 {
		m.historyIdx = len(m.history.Entries) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}

	m.loadHistoryEntry()
}

func (m *WorkbenchModel) historyNext() {
	if m.history == nil || len(m.history.Entries) == 0 || m.historyIdx == -1 {
		return
	}

	if m.historyIdx < len(m.history.Entries)-1 {
		m.historyIdx++
		m.loadHistoryEntry()
	} else {
		m.historyIdx = -1
	}
}

func (m *WorkbenchModel) loadHistoryEntry() {
	if m.history == nil || m.historyIdx < 0 || m.historyIdx >= len(m.history.Entries) {
		return
	}

	entry := m.history.Entries[m.historyIdx]
	for i, method := range methods {
		if method == entry.Method {
			m.methodIdx = i
			break
		}
	}
	m.path.SetValue(entry.Path)
	m.body.SetValue(entry.Body)
}

func (m WorkbenchModel) execute() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		method := methods[m.methodIdx]
		path := m.path.Value()
		body := m.body.Value()
		result := m.client.Request(ctx, method, path, body)
		return executeResultMsg{result}
	}
}

func (m WorkbenchModel) View() string {
	// Method + Path row
	methodStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)
	if m.focus == FocusMethod {
		methodStyle = methodStyle.Background(ColorBlue).Foreground(ColorWhite)
	}
	methodView := methodStyle.Render(methods[m.methodIdx] + " ▼")

	pathStyle := lipgloss.NewStyle()
	if m.focus == FocusPath {
		pathStyle = pathStyle.Border(lipgloss.RoundedBorder()).BorderForeground(ColorBlue)
	}
	pathView := pathStyle.Render(m.path.View())

	topRow := lipgloss.JoinHorizontal(lipgloss.Center, methodView, " ", pathView)

	// Split panes
	paneWidth := (m.width - 3) / 2

	// Left pane - body
	bodyBorder := lipgloss.RoundedBorder()
	bodyBorderColor := ColorGray
	if m.focus == FocusBody {
		bodyBorderColor = ColorBlue
	}
	bodyPane := lipgloss.NewStyle().
		Border(bodyBorder).
		BorderForeground(bodyBorderColor).
		Width(paneWidth).
		Height(m.height - 6).
		Render(m.body.View())

	// Right pane - response
	responseBorder := lipgloss.RoundedBorder()
	responseBorderColor := ColorGray
	if m.focus == FocusResponse {
		responseBorderColor = ColorBlue
	}

	responseHeader := "Response"
	if m.statusCode > 0 {
		statusColor := ColorGreen
		if m.statusCode >= 400 {
			statusColor = ColorRed
		}
		responseHeader = fmt.Sprintf("Response  %s %s",
			lipgloss.NewStyle().Foreground(statusColor).Render(fmt.Sprintf("%d", m.statusCode)),
			lipgloss.NewStyle().Foreground(ColorGray).Render(m.duration))
	}
	if m.executing {
		responseHeader = "Executing..."
	}

	responsePane := lipgloss.NewStyle().
		Border(responseBorder).
		BorderForeground(responseBorderColor).
		Width(paneWidth).
		Height(m.height - 6).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render(responseHeader),
			m.response.View()))

	panes := lipgloss.JoinHorizontal(lipgloss.Top, bodyPane, " ", responsePane)

	// Status bar
	validIndicator := lipgloss.NewStyle().Foreground(ColorGreen).Render("✓ Valid JSON")
	if !m.isValidJSON() {
		validIndicator = lipgloss.NewStyle().Foreground(ColorRed).Render("✗ Invalid JSON")
	}
	padding := m.width - 50
	if padding < 0 {
		padding = 0
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Center,
		validIndicator,
		strings.Repeat(" ", padding),
		HelpStyle.Render("Ctrl+E: Execute  Ctrl+M: Method  Ctrl+P: Pretty"))

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "", panes, statusBar)
}

func highlightJSON(input string) string {
	var buf bytes.Buffer
	err := quick.Highlight(&buf, input, "json", "terminal256", "monokai")
	if err != nil {
		return input
	}
	return buf.String()
}
