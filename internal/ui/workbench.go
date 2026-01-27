package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
	"github.com/labtiva/stoptail/internal/storage"
)

type WorkbenchFocus int

const (
	FocusNone WorkbenchFocus = iota
	FocusMethod
	FocusPath
	FocusBody
	FocusResponse

	completionMaxVisible = 8
)

type QueryMode int

const (
	ModeREST QueryMode = iota
	ModeESSQL
)

var methods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}

var bracketPairs = map[string]string{
	"{": "}",
	"[": "]",
	`"`: `"`,
}

type WorkbenchModel struct {
	client       *es.Client
	methodIdx    int
	path         textinput.Model
	editor       Editor
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
	spinner      spinner.Model
	search       SearchBar
	completion   CompletionState
	fieldCache   map[string][]CompletionItem
	lastIndex    string
	clipboard    Clipboard
	bookmarkUI   BookmarkUI
	bookmarks    *storage.Bookmarks
	queryMode    QueryMode
	dslContent   string
	dslPath      string
	esqlContent  string
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

	editor := NewEditor()

	vp := viewport.New(40, 10)

	history, _ := storage.LoadHistory()

	methodIdx := 0
	var dslContent, dslPath, esqlContent string

	if last := history.LastByMode(""); last != nil {
		path.SetValue(last.Path)
		dslPath = last.Path
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(last.Body), "", "  "); err == nil {
			dslContent = pretty.String()
		} else {
			dslContent = last.Body
		}
		for i, m := range methods {
			if m == last.Method {
				methodIdx = i
				break
			}
		}
	} else {
		path.SetValue("/_search")
		dslPath = "/_search"
		dslContent = "{}"
	}

	if last := history.LastByMode("esql"); last != nil {
		esqlContent = last.Body
	} else {
		esqlContent = "FROM * | LIMIT 10"
	}

	editor.SetContent(dslContent)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SpinnerClr)

	bookmarks, _ := storage.LoadBookmarks()

	return WorkbenchModel{
		methodIdx:   methodIdx,
		path:        path,
		editor:      editor,
		response:    vp,
		focus:       FocusNone,
		history:     history,
		historyIdx:  -1,
		spinner:     s,
		search:      NewSearchBar(),
		fieldCache:  make(map[string][]CompletionItem),
		clipboard:   NewClipboard(),
		bookmarkUI:  NewBookmarkUI(),
		bookmarks:   bookmarks,
		queryMode:   ModeREST,
		dslContent:  dslContent,
		dslPath:     dslPath,
		esqlContent: esqlContent,
	}
}

func (m *WorkbenchModel) SetClient(client *es.Client) {
	m.client = client
	m.editor.SetClient(client)
}

func (m WorkbenchModel) HasActiveInput() bool {
	return m.focus == FocusPath || m.focus == FocusBody || m.search.Active() || m.bookmarkUI.Active()
}

func (m WorkbenchModel) ClipboardMessage() string {
	return m.clipboard.Message()
}

func (m *WorkbenchModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	paneInnerWidth := (width - 5) / 2
	bodyHeight := height - 6

	m.path.Width = paneInnerWidth - 8
	m.editor.SetSize(paneInnerWidth, bodyHeight-2)
	m.response.Width = paneInnerWidth
	m.response.Height = bodyHeight - 2
}

func (m *WorkbenchModel) Prefill(index string) {
	m.methodIdx = 0 // GET
	m.path.SetValue("/" + index + "/_search")
	m.editor.SetContent("{}")
}

func (m *WorkbenchModel) toggleMode() {
	if m.queryMode == ModeREST {
		m.dslContent = m.editor.Content()
		m.dslPath = m.path.Value()
		m.queryMode = ModeESSQL
		m.editor.SetContent(m.esqlContent)
		m.path.SetValue("/_query")
	} else {
		m.esqlContent = m.editor.Content()
		m.queryMode = ModeREST
		m.editor.SetContent(m.dslContent)
		m.path.SetValue(m.dslPath)
	}
}

func (m *WorkbenchModel) Focus() {
	m.path.Focus()
	m.focus = FocusPath
}

func (m *WorkbenchModel) Blur() {
	m.path.Blur()
	m.editor.Blur()
	m.focus = FocusNone
}

func (m *WorkbenchModel) SetBody(body string) {
	m.editor.SetContent(body)
}

func (m WorkbenchModel) jsonError() (line, col int, msg string) {
	val := m.editor.Content()
	if val == "" {
		return 0, 0, ""
	}

	var js json.RawMessage
	err := json.Unmarshal([]byte(val), &js)
	if err == nil {
		return 0, 0, ""
	}

	if syntaxErr, ok := err.(*json.SyntaxError); ok {
		offset := int(syntaxErr.Offset)
		line, col = offsetToLineCol(val, offset)
		return line, col, syntaxErr.Error()
	}

	return 1, 1, err.Error()
}

func offsetToLineCol(text string, offset int) (line, col int) {
	line = 1
	col = 1
	for i, ch := range text {
		if i >= offset {
			break
		}
		if ch == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
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
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, []byte(msg.result.Body), "", "  "); err == nil {
				m.responseText = highlightJSON(pretty.String())
			} else {
				m.responseText = highlightJSON(msg.result.Body)
			}
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

	case validateTickMsg:
		return m, m.editor.executeValidation(context.Background())

	case validateMsg:
		if msg.err != nil {
			m.editor.validationState = ValidationIdle
		} else if msg.result.Valid {
			m.editor.validationState = ValidationValid
			m.editor.validationError = ""
		} else {
			m.editor.validationState = ValidationInvalid
			m.editor.validationError = msg.result.Error
		}
		return m, nil

	case tea.KeyMsg:
		m.clipboard.ClearMessage()
		if m.bookmarkUI.Active() {
			action, bookmark := m.bookmarkUI.HandleKey(msg)
			switch action {
			case BookmarkActionSave:
				if bookmark != nil {
					bookmark.Method = methods[m.methodIdx]
					bookmark.Path = m.path.Value()
					bookmark.Body = m.editor.Content()
					m.bookmarks.Add(*bookmark)
					_ = storage.SaveBookmarks(m.bookmarks)
				}
			case BookmarkActionLoad:
				if bookmark != nil {
					for i, method := range methods {
						if method == bookmark.Method {
							m.methodIdx = i
							break
						}
					}
					m.path.SetValue(bookmark.Path)
					m.editor.SetContent(bookmark.Body)
				}
			}
			return m, nil
		}
		if m.search.Active() {
			cmd, action := m.search.HandleKey(msg)
			switch action {
			case SearchActionClose:
				m.focus = FocusResponse
			case SearchActionNext, SearchActionPrev:
				if match := m.search.CurrentMatch(); match >= 0 {
					m.response.GotoTop()
					m.response.SetYOffset(match)
				}
			case SearchActionNone:
				m.updateSearchMatches()
			}
			return m, cmd
		}

		switch msg.String() {
		case "enter":
			if m.focus == FocusNone {
				m.path.Focus()
				m.focus = FocusPath
				return m, textinput.Blink
			}
		case "ctrl+f":
			if m.focus == FocusResponse {
				m.search.Activate()
				return m, textinput.Blink
			}
		case "ctrl+r":
			if cmd := m.startExecution(); cmd != nil {
				return m, cmd
			}
		case "ctrl+s":
			return m, m.bookmarkUI.OpenSave()
		case "ctrl+b":
			m.bookmarkUI.OpenLoad()
			return m, nil
		case "ctrl+y":
			var text string
			switch m.focus {
			case FocusBody:
				text = m.editor.Content()
			case FocusResponse:
				text = m.responseText
			}
			m.clipboard.Copy(text)
			return m, nil
		case "ctrl+e":
			m.toggleMode()
			return m, nil
		case "ctrl+c":
			if m.focus == FocusBody && m.editor.selection.Active {
				text := m.editor.GetSelectedText()
				if text != "" {
					m.clipboard.Copy(text)
					m.editor.selection.Active = false
				}
				return m, nil
			}
		case "n", "ctrl+n":
			if m.focus == FocusResponse && m.search.MatchCount() > 0 && !m.search.Active() {
				if match := m.search.NextMatch(); match >= 0 {
					m.response.GotoTop()
					m.response.SetYOffset(match)
				}
				return m, nil
			}
		case "N", "ctrl+p":
			if m.focus == FocusResponse && m.search.MatchCount() > 0 && !m.search.Active() {
				if match := m.search.PrevMatch(); match >= 0 {
					m.response.GotoTop()
					m.response.SetYOffset(match)
				}
				return m, nil
			}
		case "tab":
			if m.focus == FocusNone {
				break
			}
			if m.focus == FocusBody {
				if m.queryMode == ModeREST {
					if m.completion.Active {
						m.completion.MoveDown()
						return m, nil
					}
					if m.editor.IsKeyCompletionPosition() {
						m.triggerCompletion()
						return m, nil
					}
				}
				m.editor.InsertString("  ")
				return m, nil
			}
			m.cycleFocus()
			return m, nil
		case "shift+tab":
			if m.focus == FocusBody && m.completion.Active {
				m.completion.MoveUp()
				return m, nil
			}
		case "esc":
			m.path.Blur()
			m.editor.Blur()
			m.focus = FocusNone
			m.completion.Close()
			return m, nil
		}

		// Handle completion keys before passing to textarea
		if m.focus == FocusBody && m.completion.Active {
			switch msg.String() {
			case "up":
				m.completion.MoveUp()
				return m, nil
			case "down":
				m.completion.MoveDown()
				return m, nil
			case "enter":
				m.acceptCompletion()
				return m, nil
			}
		}

		if m.focus == FocusBody && m.queryMode == ModeREST {
			if pair, ok := bracketPairs[msg.String()]; ok {
				m.editor.InsertString(msg.String() + pair)
				m.editor.Update(tea.KeyMsg{Type: tea.KeyLeft})
				if msg.String() == "\"" && m.shouldAutoComplete() {
					m.triggerCompletion()
				}
				return m, nil
			}
		}

		switch m.focus {
		case FocusPath:
			m.path, cmd = m.path.Update(msg)
			cmds = append(cmds, cmd)
			if cmd := m.checkIndexChange(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case FocusBody:
			cmd = m.editor.Update(msg)
			cmds = append(cmds, cmd)

			if m.queryMode == ModeREST {
				key := msg.String()
				if len(key) == 1 || key == "backspace" || key == "enter" || key == "delete" {
					m.editor.validationState = ValidationPending
					cmds = append(cmds, m.editor.triggerValidation())
				}

				if m.completion.Active {
					if len(key) == 1 || key == "backspace" {
						col := m.editor.LineInfo().CharOffset
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
		paneInnerWidth := (m.width - 5) / 2
		topRowHeight := 3
		bodyPaneTop := topRowHeight + 2
		bodyHeaderHeight := 1

		if msg.X < paneInnerWidth+1 && msg.Y >= bodyPaneTop {
			if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
				editorX := msg.X - 1
				editorY := msg.Y - bodyPaneTop - 1 - bodyHeaderHeight
				m.path.Blur()
				m.editor.Focus()
				m.focus = FocusBody
				m.editor.HandleClick(editorX, editorY)
				return m, nil
			}
		}

		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if msg.Y < topRowHeight+1 {
				btnStyle := lipgloss.NewStyle().Padding(0, 1)

				var modeView string
				if m.queryMode == ModeREST {
					modeView = btnStyle.Bold(true).Render("[REST]")
				} else {
					modeView = btnStyle.Bold(true).Render("[ES|QL]")
				}

				pos := 0
				modeEnd := pos + lipgloss.Width(modeView)
				pos = modeEnd + 1

				var methodEnd int
				if m.queryMode == ModeREST {
					methodView := btnStyle.Bold(true).Render(methods[m.methodIdx] + " ▼")
					methodEnd = pos + lipgloss.Width(methodView)
					pos = methodEnd + 1
				} else {
					methodEnd = pos
				}

				pathView := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(m.path.View())
				pathEnd := pos + lipgloss.Width(pathView)
				pos = pathEnd + 2
				execBtn := btnStyle.Render("▶ Run")
				execEnd := pos + lipgloss.Width(execBtn)
				pos = execEnd + 1

				var fmtEnd int
				if m.queryMode == ModeREST {
					fmtBtn := btnStyle.Render("{ } Format")
					fmtEnd = pos + lipgloss.Width(fmtBtn)
					pos = fmtEnd + 2
				} else {
					fmtEnd = pos
					pos += 2
				}

				histPrev := btnStyle.Render("◀")
				histPrevEnd := pos + lipgloss.Width(histPrev)
				pos = histPrevEnd
				histNext := btnStyle.Render("▶")
				histNextEnd := pos + lipgloss.Width(histNext)

				if msg.X < modeEnd {
					m.toggleMode()
				} else if msg.X < methodEnd && m.queryMode == ModeREST {
					m.path.Blur()
					m.editor.Blur()
					m.focus = FocusMethod
				} else if msg.X < pathEnd {
					m.editor.Blur()
					m.path.Focus()
					m.focus = FocusPath
				} else if msg.X < execEnd {
					if cmd := m.startExecution(); cmd != nil {
						return m, cmd
					}
				} else if msg.X < fmtEnd && m.queryMode == ModeREST {
					m.prettyPrintBody()
				} else if msg.X < histPrevEnd {
					m.historyPrev()
				} else if msg.X < histNextEnd {
					m.historyNext()
				}
				return m, nil
			} else if msg.X < paneInnerWidth+1 {
				m.path.Blur()
				m.editor.Focus()
				m.focus = FocusBody
			} else {
				m.path.Blur()
				m.editor.Blur()
				m.focus = FocusResponse

				if m.search.Active() && msg.Y >= m.height-4 {
					relX := msg.X - paneInnerWidth - 3 - 1
					action := m.search.HandleClick(relX)
					if action == SearchActionNext || action == SearchActionPrev {
						if match := m.search.CurrentMatch(); match >= 0 {
							m.response.GotoTop()
							m.response.SetYOffset(match)
						}
					}
				}
			}
			return m, nil
		}

		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			scrollAmount := 3

			if msg.Y > topRowHeight {
				if msg.X < paneInnerWidth+2 {
					if msg.Button == tea.MouseButtonWheelUp {
						m.editor.SetCursor(max(0, m.editor.Line()-scrollAmount))
					} else {
						m.editor.SetCursor(m.editor.Line() + scrollAmount)
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
	m.editor.Blur()

	m.focus = (m.focus + 1) % 4
	switch m.focus {
	case FocusMethod:
		// No component to focus
	case FocusPath:
		m.path.Focus()
	case FocusBody:
		m.editor.Focus()
	case FocusResponse:
		// Viewport doesn't need focus call
	}
}

func (m *WorkbenchModel) prettyPrintBody() {
	val := m.editor.Content()
	if val == "" {
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(val), "", "  "); err == nil {
		m.editor.SetContent(pretty.String())
	}
}

func (m *WorkbenchModel) startExecution() tea.Cmd {
	if m.client == nil || m.executing {
		return nil
	}
	m.prettyPrintBody()
	m.executing = true
	return tea.Batch(m.spinner.Tick, m.execute())
}

func (m *WorkbenchModel) addToHistory() {
	var mode string
	if m.queryMode == ModeESSQL {
		mode = "esql"
	}
	entry := storage.HistoryEntry{
		Method: methods[m.methodIdx],
		Path:   m.path.Value(),
		Body:   m.editor.Content(),
		Mode:   mode,
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

	targetMode := ""
	if m.queryMode == ModeESSQL {
		targetMode = "esql"
	}

	start := m.historyIdx
	if start == -1 {
		start = len(m.history.Entries)
	}

	for i := start - 1; i >= 0; i-- {
		if m.history.Entries[i].Mode == targetMode {
			m.historyIdx = i
			m.loadHistoryEntry()
			return
		}
	}
}

func (m *WorkbenchModel) historyNext() {
	if m.history == nil || len(m.history.Entries) == 0 || m.historyIdx == -1 {
		return
	}

	targetMode := ""
	if m.queryMode == ModeESSQL {
		targetMode = "esql"
	}

	for i := m.historyIdx + 1; i < len(m.history.Entries); i++ {
		if m.history.Entries[i].Mode == targetMode {
			m.historyIdx = i
			m.loadHistoryEntry()
			return
		}
	}
	m.historyIdx = -1
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
	m.editor.SetContent(entry.Body)
}

func (m *WorkbenchModel) updateSearchMatches() {
	lines := strings.Split(m.responseText, "\n")
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.response.GotoTop()
		m.response.SetYOffset(match)
	}
}

func (m WorkbenchModel) execute() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var method, path, body string

		if m.queryMode == ModeESSQL {
			method = "POST"
			path = m.path.Value()
			query := m.editor.Content()
			escaped, _ := json.Marshal(query)
			body = fmt.Sprintf(`{"query":%s}`, string(escaped))
		} else {
			method = methods[m.methodIdx]
			path = m.path.Value()
			body = m.editor.Content()
		}

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
		m.editor.SetIndex(index)
		if _, ok := m.fieldCache[index]; !ok {
			return m.fetchMapping(index)
		}
	}
	return nil
}

func (m WorkbenchModel) View() string {
	modeStyle := lipgloss.NewStyle().Padding(0, 1).Bold(true)
	var modeView string
	if m.queryMode == ModeREST {
		modeView = modeStyle.Render("[REST]")
	} else {
		modeView = modeStyle.Background(ColorBlue).Foreground(ColorOnAccent).Render("[ES|QL]")
	}

	var methodView string
	if m.queryMode == ModeREST {
		methodStyle := lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)
		if m.focus == FocusMethod {
			methodStyle = methodStyle.Background(ColorBlue).Foreground(ColorOnAccent)
		}
		methodView = methodStyle.Render(methods[m.methodIdx] + " ▼")
	}

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

	var fmtBtn string
	if m.queryMode == ModeREST {
		fmtBtn = btnStyle.Render("{ } Format")
	}

	histPrev := btnStyle.Render("◀")
	histNext := btnStyle.Render("▶")

	var topRowParts []string
	topRowParts = append(topRowParts, modeView)
	if methodView != "" {
		topRowParts = append(topRowParts, " ", methodView)
	}
	topRowParts = append(topRowParts, " ", pathView, "  ", execBtn)
	if fmtBtn != "" {
		topRowParts = append(topRowParts, " ", fmtBtn)
	}
	topRowParts = append(topRowParts, "  ", histPrev, histNext)

	topRow := lipgloss.JoinHorizontal(lipgloss.Center, topRowParts...)

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
	var errLine, errCol int
	var errMsg string
	if m.queryMode == ModeREST {
		errLine, errCol, errMsg = m.jsonError()
	}

	bodyContent := m.editor.View()
	if errMsg != "" {
		bodyContent = m.overlayErrorMarker(bodyContent, errLine)
	}
	if m.completion.Active {
		dropdown := m.renderCompletionDropdown()
		bodyContent = m.overlayDropdown(bodyContent, dropdown)
	}

	var bodyHeaderText string
	var bodyValidation string
	if m.queryMode == ModeESSQL {
		bodyHeaderText = "Body (ES|QL)"
		bodyValidation = ""
	} else {
		bodyHeaderText = "Body"
		if errMsg != "" {
			bodyValidation = lipgloss.NewStyle().Foreground(ColorRed).Render(
				fmt.Sprintf("✗ %d:%d", errLine, errCol))
		} else {
			jsonValid := lipgloss.NewStyle().Foreground(ColorGreen).Render("✓")
			switch m.editor.validationState {
			case ValidationPending:
				bodyValidation = jsonValid + " " + lipgloss.NewStyle().Foreground(ColorYellow).Render("⋯")
			case ValidationValid:
				bodyValidation = jsonValid
			case ValidationInvalid:
				esErr := m.editor.validationError
				if len(esErr) > 30 {
					esErr = esErr[:30] + "..."
				}
				bodyValidation = jsonValid + " " + lipgloss.NewStyle().Foreground(ColorRed).Render("✗ "+esErr)
			default:
				bodyValidation = jsonValid
			}
		}
	}
	bodyHeader := lipgloss.NewStyle().Bold(true).Render(bodyHeaderText) + "  " + bodyValidation

	bodyPaneContent := lipgloss.JoinVertical(lipgloss.Left,
		bodyHeader,
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
	if m.search.Active() {
		searchBar := m.search.View(paneInnerWidth - 4)
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
			bl = TrimANSI(bodyLines[i])
		}
		if i < len(responseLines) {
			rl = TrimANSI(responseLines[i])
		}
		paneLines = append(paneLines, bl+" "+rl)
	}
	panes := strings.Join(paneLines, "\n")

	output := lipgloss.JoinVertical(lipgloss.Left, topRow, "", panes)

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}

	if m.bookmarkUI.Active() {
		return m.bookmarkUI.View(m.width, m.height)
	}

	return strings.Join(lines, "\n")
}

func (m *WorkbenchModel) triggerCompletion() {
	text := m.editor.Content()
	lines := strings.Split(text, "\n")
	row := m.editor.Line()
	col := m.editor.LineInfo().CharOffset

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

func (m *WorkbenchModel) shouldAutoComplete() bool {
	content := m.editor.Content()
	lines := strings.Split(content, "\n")
	row := m.editor.Line()
	col := m.editor.LineInfo().CharOffset

	if row >= len(lines) {
		return false
	}

	line := lines[row]
	if col < 1 || col > len(line) {
		return false
	}

	beforeQuote := strings.TrimRight(line[:col-1], " \t")
	if len(beforeQuote) == 0 {
		for i := row - 1; i >= 0; i-- {
			trimmed := strings.TrimRight(lines[i], " \t")
			if len(trimmed) > 0 {
				beforeQuote = trimmed
				break
			}
		}
	}

	if len(beforeQuote) == 0 {
		return false
	}

	lastChar := beforeQuote[len(beforeQuote)-1]
	return lastChar == '{' || lastChar == ','
}

func (m *WorkbenchModel) getCompletionQuery() string {
	lines := strings.Split(m.editor.Content(), "\n")
	row := m.editor.Line()
	col := m.editor.LineInfo().CharOffset

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

	content := m.editor.Content()
	lines := strings.Split(content, "\n")
	row := m.editor.Line()
	col := m.editor.LineInfo().CharOffset

	needsOpenQuote := true
	if row < len(lines) {
		line := lines[row]
		if m.completion.TriggerCol > 0 && m.completion.TriggerCol <= len(line) {
			if line[m.completion.TriggerCol-1] == '"' {
				needsOpenQuote = false
			}
		}
		if col < len(line) && line[col] == '"' {
			m.editor.Update(tea.KeyMsg{Type: tea.KeyDelete})
		}
	}

	prefix := ""
	if needsOpenQuote {
		prefix = `"`
	}
	m.editor.InsertString(prefix + insertion + suffix)
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

func (m WorkbenchModel) overlayErrorMarker(bodyView string, errorLine int) string {
	if errorLine <= 0 {
		return bodyView
	}

	lines := strings.Split(bodyView, "\n")
	if errorLine > len(lines) {
		return bodyView
	}

	redIndicator := lipgloss.NewStyle().Foreground(ColorRed).Render("┃")
	lines[errorLine-1] = strings.Replace(lines[errorLine-1], "┃", redIndicator, 1)

	return strings.Join(lines, "\n")
}

func (m WorkbenchModel) overlayDropdown(bodyView, dropdown string) string {
	if dropdown == "" {
		return bodyView
	}

	bodyLines := strings.Split(bodyView, "\n")
	dropdownLines := strings.Split(dropdown, "\n")

	dropdownHeight := len(dropdownLines)
	if dropdownHeight >= len(bodyLines) {
		return dropdown
	}

	result := make([]string, len(bodyLines))
	copy(result, bodyLines[:len(bodyLines)-dropdownHeight])
	copy(result[len(bodyLines)-dropdownHeight:], dropdownLines)

	return strings.Join(result, "\n")
}

func highlightJSON(input string) string {
	var buf bytes.Buffer
	err := quick.Highlight(&buf, input, "json", "terminal256", "monokai")
	if err != nil {
		return input
	}
	return buf.String()
}
