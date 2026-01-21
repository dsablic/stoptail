package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/spinner"
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

	completionMaxVisible = 8
)

var methods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}

type WorkbenchModel struct {
	client        *es.Client
	methodIdx     int
	path          textinput.Model
	body          textarea.Model
	response      viewport.Model
	responseText  string
	statusCode    int
	duration      string
	focus         WorkbenchFocus
	width         int
	height        int
	executing     bool
	err           error
	history       *storage.History
	historyIdx    int
	spinner       spinner.Model
	searchActive  bool
	searchInput   textinput.Model
	searchMatches []int
	searchIdx     int
	completion    CompletionState
	fieldCache    map[string][]CompletionItem
	lastIndex     string
}

type executeResultMsg struct {
	result es.RequestResult
}

type mappingResultMsg struct {
	index  string
	fields []string
}

func NewWorkbench() WorkbenchModel {
	path := textinput.New()
	path.Placeholder = "/_search"
	path.CharLimit = 200
	path.Width = 40
	path.Cursor.Style = lipgloss.NewStyle().Background(ColorBlue).Foreground(ColorOnAccent)
	path.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	path.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorGray)

	body := textarea.New()
	body.Placeholder = `{"query": {"match_all": {}}}`
	body.CharLimit = 50000
	body.ShowLineNumbers = false
	body.Cursor.Style = lipgloss.NewStyle().Background(ColorBlue).Foreground(ColorOnAccent)
	body.FocusedStyle.Text = lipgloss.NewStyle().Foreground(ColorWhite)
	body.FocusedStyle.CursorLine = lipgloss.NewStyle()
	body.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorGray)
	body.BlurredStyle.Text = lipgloss.NewStyle().Foreground(ColorGray)
	body.BlurredStyle.CursorLine = lipgloss.NewStyle()
	body.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorGray)

	vp := viewport.New(40, 10)

	history, _ := storage.LoadHistory()

	methodIdx := 0
	if last := history.Last(); last != nil {
		path.SetValue(last.Path)
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(last.Body), "", "  "); err == nil {
			body.SetValue(pretty.String())
		} else {
			body.SetValue(last.Body)
		}
		for i, m := range methods {
			if m == last.Method {
				methodIdx = i
				break
			}
		}
	} else {
		path.SetValue("/_search")
		body.SetValue("{}")
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)

	search := textinput.New()
	search.Placeholder = "Search..."
	search.CharLimit = 100
	search.Width = 30

	return WorkbenchModel{
		methodIdx:   methodIdx,
		path:        path,
		body:        body,
		response:    vp,
		focus:       FocusPath,
		history:     history,
		historyIdx:  -1,
		spinner:     s,
		searchInput: search,
		fieldCache:  make(map[string][]CompletionItem),
	}
}

func (m *WorkbenchModel) SetClient(client *es.Client) {
	m.client = client
}

func (m WorkbenchModel) HasActiveInput() bool {
	return m.focus == FocusPath || m.focus == FocusBody
}

func (m *WorkbenchModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Split panes: account for borders (2 chars each pane) and divider (1 char)
	paneInnerWidth := (width - 5) / 2
	bodyHeight := height - 6

	m.path.Width = paneInnerWidth - 8
	m.body.SetWidth(paneInnerWidth)
	m.body.SetHeight(bodyHeight - 2)
	m.response.Width = paneInnerWidth
	m.response.Height = bodyHeight - 2
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
	case spinner.TickMsg:
		if m.executing {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
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

	case mappingResultMsg:
		items := make([]CompletionItem, len(msg.fields))
		for i, f := range msg.fields {
			items[i] = CompletionItem{Text: f, Kind: "field"}
		}
		m.fieldCache[msg.index] = items
		return m, nil

	case tea.KeyMsg:
		if m.searchActive {
			switch msg.String() {
			case "esc", "enter":
				m.searchActive = false
				m.searchInput.Blur()
				return m, nil
			case "ctrl+n", "n":
				if len(m.searchMatches) > 0 {
					m.searchIdx = (m.searchIdx + 1) % len(m.searchMatches)
					m.response.GotoTop()
					m.response.SetYOffset(m.searchMatches[m.searchIdx])
				}
				return m, nil
			case "ctrl+p", "N":
				if len(m.searchMatches) > 0 {
					m.searchIdx--
					if m.searchIdx < 0 {
						m.searchIdx = len(m.searchMatches) - 1
					}
					m.response.GotoTop()
					m.response.SetYOffset(m.searchMatches[m.searchIdx])
				}
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.updateSearchMatches()
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+f":
			if m.focus == FocusResponse {
				m.searchActive = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				m.searchMatches = nil
				m.searchIdx = 0
				return m, textinput.Blink
			}
		case "ctrl+m":
			m.methodIdx = (m.methodIdx + 1) % len(methods)
			return m, nil
		case "ctrl+enter", "ctrl+e":
			if m.client != nil && !m.executing {
				m.executing = true
				return m, tea.Batch(m.spinner.Tick, m.execute())
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
			if m.completion.Active {
				m.acceptCompletion()
				return m, nil
			}
			if m.focus == FocusBody {
				return m, nil
			}
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
			if cmd := m.checkIndexChange(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case FocusBody:
			m.body, cmd = m.body.Update(msg)
			cmds = append(cmds, cmd)

			key := msg.String()
			if !m.completion.Active {
				if key == `"` || key == ":" || key == "shift+;" {
					m.triggerCompletion()
				}
			} else {
				switch key {
				case "up":
					m.completion.MoveUp()
					return m, nil
				case "down":
					m.completion.MoveDown()
					return m, nil
				case "enter", "tab":
					m.acceptCompletion()
					return m, nil
				case "esc":
					m.completion.Close()
					return m, nil
				default:
					if len(key) == 1 || key == "backspace" {
						col := m.body.LineInfo().CharOffset
						if col > m.completion.TriggerCol {
							query := m.getCompletionQuery()
							m.completion.Filter(query)
						} else {
							m.completion.Close()
						}
					}
				}
			}
		case FocusResponse:
			m.response, cmd = m.response.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			paneInnerWidth := (m.width - 5) / 2

			topRowHeight := 3

			if msg.Y < topRowHeight+1 {
				btnStyle := lipgloss.NewStyle().Padding(0, 1)
				methodView := btnStyle.Bold(true).Render(methods[m.methodIdx] + " ▼")
				pathView := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(m.path.View())
				execBtn := btnStyle.Render("▶ Run")
				fmtBtn := btnStyle.Render("{ } Format")
				histPrev := btnStyle.Render("◀")
				histNext := btnStyle.Render("▶")

				pos := 0
				methodEnd := pos + lipgloss.Width(methodView)
				pos = methodEnd + 1
				pathEnd := pos + lipgloss.Width(pathView)
				pos = pathEnd + 2
				execEnd := pos + lipgloss.Width(execBtn)
				pos = execEnd + 1
				fmtEnd := pos + lipgloss.Width(fmtBtn)
				pos = fmtEnd + 2
				histPrevEnd := pos + lipgloss.Width(histPrev)
				pos = histPrevEnd
				histNextEnd := pos + lipgloss.Width(histNext)

				if msg.X < methodEnd {
					m.path.Blur()
					m.body.Blur()
					m.focus = FocusMethod
				} else if msg.X < pathEnd {
					m.body.Blur()
					m.path.Focus()
					m.focus = FocusPath
				} else if msg.X < execEnd {
					if m.client != nil && !m.executing {
						m.executing = true
						return m, tea.Batch(m.spinner.Tick, m.execute())
					}
				} else if msg.X < fmtEnd {
					m.prettyPrintBody()
				} else if msg.X < histPrevEnd {
					m.historyPrev()
				} else if msg.X < histNextEnd {
					m.historyNext()
				}
				return m, nil
			} else if msg.X < paneInnerWidth+1 {
				m.path.Blur()
				m.body.Focus()
				m.focus = FocusBody
			} else {
				m.path.Blur()
				m.body.Blur()
				m.focus = FocusResponse

				if m.searchActive && msg.Y >= m.height-4 {
					relX := msg.X - paneInnerWidth - 3
					searchInputEnd := 35
					if relX > searchInputEnd {
						text := m.searchInput.View()
						btnStart := len(text) + 10
						if len(m.searchMatches) > 0 {
							prevEnd := btnStart + 4
							nextEnd := prevEnd + 5
							closeStart := nextEnd + 1
							if relX >= btnStart && relX < prevEnd {
								m.searchIdx--
								if m.searchIdx < 0 {
									m.searchIdx = len(m.searchMatches) - 1
								}
								m.response.GotoTop()
								m.response.SetYOffset(m.searchMatches[m.searchIdx])
							} else if relX >= prevEnd && relX < nextEnd {
								m.searchIdx = (m.searchIdx + 1) % len(m.searchMatches)
								m.response.GotoTop()
								m.response.SetYOffset(m.searchMatches[m.searchIdx])
							} else if relX >= closeStart {
								m.searchActive = false
								m.searchInput.Blur()
							}
						} else if relX >= btnStart {
							m.searchActive = false
							m.searchInput.Blur()
						}
					}
				}
			}
			return m, nil
		}

		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			paneInnerWidth := (m.width - 5) / 2
			topRowHeight := 3
			scrollAmount := 3

			if msg.Y > topRowHeight {
				if msg.X < paneInnerWidth+2 {
					if msg.Button == tea.MouseButtonWheelUp {
						m.body.SetCursor(max(0, m.body.Line()-scrollAmount))
					} else {
						m.body.SetCursor(m.body.Line() + scrollAmount)
					}
				} else {
					if msg.Button == tea.MouseButtonWheelUp {
						m.response.SetYOffset(max(0, m.response.YOffset-scrollAmount))
					} else {
						m.response.SetYOffset(m.response.YOffset + scrollAmount)
					}
				}
			}
			return m, nil
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

func (m *WorkbenchModel) updateSearchMatches() {
	query := strings.ToLower(m.searchInput.Value())
	m.searchMatches = nil
	m.searchIdx = 0

	if query == "" {
		return
	}

	lines := strings.Split(m.responseText, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}

	if len(m.searchMatches) > 0 {
		m.response.GotoTop()
		m.response.SetYOffset(m.searchMatches[0])
	}
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

func (m WorkbenchModel) fetchMapping(index string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}
		ctx := context.Background()
		fields, err := m.client.FetchMapping(ctx, index)
		if err != nil {
			return nil
		}
		return mappingResultMsg{index: index, fields: fields}
	}
}

func (m *WorkbenchModel) checkIndexChange() tea.Cmd {
	index := m.extractIndexFromPath()
	if index != "" && index != m.lastIndex {
		m.lastIndex = index
		if _, ok := m.fieldCache[index]; !ok {
			return m.fetchMapping(index)
		}
	}
	return nil
}

func (m WorkbenchModel) View() string {
	methodStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)
	if m.focus == FocusMethod {
		methodStyle = methodStyle.Background(ColorBlue).Foreground(ColorOnAccent)
	}
	methodView := methodStyle.Render(methods[m.methodIdx] + " ▼")

	pathBorderColor := ColorGray
	if m.focus == FocusPath {
		pathBorderColor = ColorBlue
	}
	pathStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(pathBorderColor)
	pathView := pathStyle.Render(m.path.View())

	btnStyle := lipgloss.NewStyle().Padding(0, 1).Background(ActiveBg)
	execBtn := btnStyle.Render("▶ Run")
	if m.executing {
		execBtn = btnStyle.Render(m.spinner.View())
	}
	fmtBtn := btnStyle.Render("{ } Format")
	histPrev := btnStyle.Render("◀")
	histNext := btnStyle.Render("▶")

	topRow := lipgloss.JoinHorizontal(lipgloss.Center,
		methodView, " ", pathView, "  ",
		execBtn, " ", fmtBtn, "  ",
		histPrev, histNext)

	// Split panes: account for borders (2 chars each pane) and divider (1 char)
	// Total = 2 * (innerWidth + 2) + 1 = width
	// innerWidth = (width - 5) / 2
	paneInnerWidth := (m.width - 5) / 2

	// Left pane - body
	bodyBorder := lipgloss.RoundedBorder()
	bodyBorderColor := ColorGray
	if m.focus == FocusBody {
		bodyBorderColor = ColorBlue
	}
	bodyContent := m.body.View()
	if m.completion.Active {
		dropdown := m.renderCompletionDropdown()
		bodyContent = lipgloss.JoinVertical(lipgloss.Left, bodyContent, dropdown)
	}
	bodyPaneContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Body"),
		bodyContent)
	bodyPane := lipgloss.NewStyle().
		Border(bodyBorder).
		BorderForeground(bodyBorderColor).
		Width(paneInnerWidth).
		Height(m.height - 6).
		Render(bodyPaneContent)

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
		responseHeader = m.spinner.View() + " Executing..."
	}

	responseContent := m.response.View()
	if m.searchActive {
		searchStatus := ""
		if len(m.searchMatches) > 0 {
			searchStatus = fmt.Sprintf(" %d/%d ", m.searchIdx+1, len(m.searchMatches))
		} else if m.searchInput.Value() != "" {
			searchStatus = " No matches "
		}
		navBtns := ""
		if len(m.searchMatches) > 0 {
			navBtns = " [◀] [▶]"
		}
		searchBar := lipgloss.NewStyle().
			Background(ActiveBg).
			Padding(0, 1).
			Width(paneInnerWidth - 4).
			Render("/" + m.searchInput.View() + searchStatus + navBtns + " [×]")
		responseContent = lipgloss.JoinVertical(lipgloss.Left, responseContent, searchBar)
	}

	responsePaneContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(responseHeader),
		responseContent)

	responsePane := lipgloss.NewStyle().
		Border(responseBorder).
		BorderForeground(responseBorderColor).
		Width(paneInnerWidth).
		Height(m.height - 6).
		Render(responsePaneContent)

	trimANSI := func(s string) string {
		for strings.HasSuffix(s, " ") || strings.HasSuffix(s, "\x1b[0m") {
			s = strings.TrimSuffix(s, " ")
			s = strings.TrimSuffix(s, "\x1b[0m")
		}
		return s + "\x1b[0m"
	}
	bodyLines := strings.Split(bodyPane, "\n")
	responseLines := strings.Split(responsePane, "\n")
	maxLines := len(bodyLines)
	if len(responseLines) > maxLines {
		maxLines = len(responseLines)
	}
	var paneLines []string
	for i := 0; i < maxLines; i++ {
		bl := ""
		rl := ""
		if i < len(bodyLines) {
			bl = trimANSI(bodyLines[i])
		}
		if i < len(responseLines) {
			rl = trimANSI(responseLines[i])
		}
		paneLines = append(paneLines, bl+" "+rl)
	}
	panes := strings.Join(paneLines, "\n")

	// Status bar
	validIndicator := lipgloss.NewStyle().Foreground(ColorGreen).Render("✓ Valid JSON")
	if !m.isValidJSON() {
		validIndicator = lipgloss.NewStyle().Foreground(ColorRed).Render("✗ Invalid JSON")
	}

	helpText := "Ctrl+E: Execute  Ctrl+M: Method  Ctrl+P: Pretty  Ctrl+F: Search"
	if m.searchActive {
		helpText = "n/N: Next/Prev match  Enter/Esc: Close search"
	} else if m.focus == FocusResponse {
		helpText = "Ctrl+F: Search  ↑↓: Scroll"
	}

	padding := m.width - 50
	if padding < 0 {
		padding = 0
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Center,
		validIndicator,
		strings.Repeat(" ", padding),
		HelpStyle.Render(helpText))

	output := lipgloss.JoinVertical(lipgloss.Left, topRow, "", panes, statusBar)

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func (m *WorkbenchModel) triggerCompletion() {
	text := m.body.Value()
	lines := strings.Split(text, "\n")
	row := m.body.Line()
	col := m.body.LineInfo().CharOffset

	textUpToCursor := ""
	for i := 0; i < row && i < len(lines); i++ {
		textUpToCursor += lines[i] + "\n"
	}
	if row < len(lines) {
		if col <= len(lines[row]) {
			textUpToCursor += lines[row][:col]
		} else {
			textUpToCursor += lines[row]
		}
	}

	ctx := ParseJSONContext(textUpToCursor)

	var items []CompletionItem
	keywords := GetKeywordsForContext(ctx.Path)
	items = append(items, keywords...)

	if fields, ok := m.fieldCache[m.lastIndex]; ok {
		items = append(items, fields...)
	}

	if len(items) == 0 {
		return
	}

	m.completion.Active = true
	m.completion.Items = items
	m.completion.Filtered = items
	m.completion.SelectedIdx = 0
	m.completion.TriggerCol = col
	m.completion.Query = ""
}

func (m *WorkbenchModel) getCompletionQuery() string {
	lines := strings.Split(m.body.Value(), "\n")
	row := m.body.Line()
	col := m.body.LineInfo().CharOffset

	if row >= len(lines) {
		return ""
	}

	line := lines[row]
	if col > len(line) {
		col = len(line)
	}

	start := m.completion.TriggerCol
	if start > col {
		return ""
	}

	return line[start:col]
}

func (m *WorkbenchModel) acceptCompletion() {
	selected := m.completion.Selected()
	if selected == nil {
		m.completion.Close()
		return
	}

	query := m.getCompletionQuery()
	if len(query) > len(selected.Text) {
		m.completion.Close()
		return
	}
	insertion := selected.Text[len(query):]
	suffix := `": `

	lines := strings.Split(m.body.Value(), "\n")
	row := m.body.Line()
	col := m.body.LineInfo().CharOffset

	if row < len(lines) {
		line := lines[row]
		if col <= len(line) {
			lines[row] = line[:col] + insertion + suffix + line[col:]
		}
	}

	m.body.SetValue(strings.Join(lines, "\n"))
	newCol := col + len(insertion) + len(suffix)
	m.body.SetCursor(newCol)

	m.completion.Close()
}

func (m *WorkbenchModel) extractIndexFromPath() string {
	path := m.path.Value()
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, "_") {
			return part
		}
	}
	return ""
}

func (m WorkbenchModel) renderCompletionDropdown() string {
	if !m.completion.Active || len(m.completion.Filtered) == 0 {
		return ""
	}

	items := m.completion.Filtered
	if len(items) > completionMaxVisible {
		items = items[:completionMaxVisible]
	}

	var lines []string
	for i, item := range items {
		text := fmt.Sprintf(" %s ", item.Text)
		if item.Kind != "" {
			text = fmt.Sprintf(" %s (%s) ", item.Text, item.Kind)
		}

		style := lipgloss.NewStyle().Background(ActiveBg)
		if i == m.completion.SelectedIdx {
			style = style.Background(ColorBlue).Foreground(ColorOnAccent)
		}
		lines = append(lines, style.Render(text))
	}

	dropdown := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(strings.Join(lines, "\n"))

	return dropdown
}

func highlightJSON(input string) string {
	var buf bytes.Buffer
	err := quick.Highlight(&buf, input, "json", "terminal256", "monokai")
	if err != nil {
		return input
	}
	return buf.String()
}
