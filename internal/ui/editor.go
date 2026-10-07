package ui

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/labtiva/stoptail/internal/es"
)

type ValidationState int

const (
	ValidationIdle ValidationState = iota
	ValidationPending
	ValidationValid
	ValidationInvalid
)

type validateMsg struct {
	result *es.ValidateResult
	err    error
}

type validateTickMsg struct{}

type Editor struct {
	textarea        textarea.Model
	width           int
	height          int
	client          *es.Client
	index           string
	validationState ValidationState
	validationError string
	undoStack       []editorState
	redoStack       []editorState
}

type editorState struct {
	content   string
	cursorPos int
}

func NewEditor() Editor {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = 50000
	ta.Prompt = ""
	ta.KeyMap.LineStart.SetKeys("home")
	ta.KeyMap.SelectAll.SetKeys("ctrl+a")
	ta.KeyMap.CopySelection.SetEnabled(false)
	styles := ta.Styles()
	styles.Focused.Selection = lipgloss.NewStyle().Reverse(true)
	styles.Blurred.Selection = styles.Focused.Selection
	ta.SetStyles(styles)
	return Editor{
		textarea: ta,
	}
}

func (e *Editor) SetContent(content string) {
	e.textarea.SetValue(content)
}

func (e *Editor) Content() string {
	return e.textarea.Value()
}

func (e *Editor) SetClient(client *es.Client) {
	e.client = client
}

func (e *Editor) SetIndex(index string) {
	e.index = index
}

func (e Editor) triggerValidation() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return validateTickMsg{}
	})
}

func (e Editor) executeValidation(ctx context.Context) tea.Cmd {
	if e.client == nil || e.index == "" {
		return nil
	}
	content := e.textarea.Value()
	if content == "" {
		return nil
	}

	var query map[string]interface{}
	if err := json.Unmarshal([]byte(content), &query); err != nil {
		return nil
	}

	queryPart, ok := query["query"]
	if !ok {
		return nil
	}

	queryBytes, _ := json.Marshal(queryPart)
	return func() tea.Msg {
		result, err := e.client.ValidateQuery(ctx, e.index, queryBytes)
		return validateMsg{result: result, err: err}
	}
}

func (e Editor) IsKeyCompletionPosition() bool {
	content := e.textarea.Value()
	cursorOffset := e.getCursorOffset()
	if cursorOffset > len(content) {
		cursorOffset = len(content)
	}

	lastNonWhitespace := byte(0)
	for i := cursorOffset - 1; i >= 0; i-- {
		ch := content[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			continue
		}
		lastNonWhitespace = ch
		break
	}

	if lastNonWhitespace == '[' || lastNonWhitespace == ':' ||
		lastNonWhitespace == '"' || lastNonWhitespace == '}' ||
		lastNonWhitespace == ']' || lastNonWhitespace == 0 {
		return false
	}

	if lastNonWhitespace != '{' && lastNonWhitespace != ',' {
		return false
	}

	var bracketStack []byte
	inString := false

	for i := 0; i < cursorOffset; i++ {
		ch := content[i]
		if inString {
			if ch == '"' && (i == 0 || content[i-1] != '\\') {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{', '[':
			bracketStack = append(bracketStack, ch)
		case '}':
			if len(bracketStack) > 0 && bracketStack[len(bracketStack)-1] == '{' {
				bracketStack = bracketStack[:len(bracketStack)-1]
			}
		case ']':
			if len(bracketStack) > 0 && bracketStack[len(bracketStack)-1] == '[' {
				bracketStack = bracketStack[:len(bracketStack)-1]
			}
		}
	}

	if len(bracketStack) == 0 {
		return false
	}
	return bracketStack[len(bracketStack)-1] == '{'
}

func (e Editor) getCursorOffset() int {
	content := e.textarea.Value()
	lines := strings.Split(content, "\n")

	row := e.Line()
	col := e.logicalCol()

	offset := 0
	for i := 0; i < row && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	if row < len(lines) {
		runes := []rune(lines[row])
		if col > len(runes) {
			col = len(runes)
		}
		offset += len(string(runes[:col]))
	}
	return offset
}

func (e *Editor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.textarea.SetWidth(width)
	e.textarea.SetHeight(height)
}

func (e Editor) View() string {
	return e.textarea.View()
}

func (e Editor) GetSelectedText() string {
	return e.textarea.SelectedText()
}

func (e *Editor) Focus() {
	e.textarea.Focus()
}

func (e *Editor) Blur() {
	e.textarea.Blur()
}

func (e *Editor) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if key := keyMsg.String(); key == "shift+home" || key == "shift+end" {
			e.selectToLineEdge(key == "shift+end")
			return nil
		}
	}

	before := e.currentState()
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	if e.textarea.Value() != before.content {
		e.pushState(before)
	}
	return cmd
}

func (e *Editor) selectToLineEdge(toEnd bool) {
	code, steps := tea.KeyLeft, e.textarea.Column()
	if toEnd {
		line := strings.Split(e.textarea.Value(), "\n")[e.textarea.Line()]
		code, steps = tea.KeyRight, len([]rune(line))-e.textarea.Column()
	}
	step := tea.KeyPressMsg{Code: code, Mod: tea.ModShift}
	for range steps {
		e.textarea, _ = e.textarea.Update(step)
	}
}

func (e Editor) Line() int {
	return e.textarea.Line()
}

func (e Editor) LineInfo() textarea.LineInfo {
	return e.textarea.LineInfo()
}

func (e Editor) logicalCol() int {
	li := e.textarea.LineInfo()
	return li.StartColumn + li.ColumnOffset
}

func (e *Editor) InsertString(s string) {
	e.textarea.InsertString(s)
}

func (e *Editor) SetCursor(pos int) {
	e.textarea.SetCursorColumn(pos)
}

func (e Editor) CursorOffset() int {
	return e.getCursorOffset()
}

func (e Editor) HasSelection() bool {
	return e.textarea.HasSelection()
}

func (e *Editor) ClearSelection() {
	e.textarea.ClearSelection()
}

func (e *Editor) SelectAll() {
	e.textarea.SelectAll()
}

func (e *Editor) DeleteSelection() {
	e.textarea.DeleteSelection()
}

func (e *Editor) BeginMouseSelection(x, y int) {
	e.textarea.BeginSelection(x, y)
}

func (e *Editor) ExtendMouseSelection(x, y int) {
	e.textarea.ExtendSelection(x, y)
}

func (e *Editor) EndMouseSelection() {
	e.textarea.EndSelection()
}

func (e Editor) currentState() editorState {
	return editorState{
		content:   e.textarea.Value(),
		cursorPos: e.getCursorOffset(),
	}
}

func (e *Editor) SaveState() {
	e.pushState(e.currentState())
}

func (e *Editor) pushState(state editorState) {
	if len(e.undoStack) > 0 && e.undoStack[len(e.undoStack)-1].content == state.content {
		return
	}
	e.undoStack = append(e.undoStack, state)
	if len(e.undoStack) > 100 {
		e.undoStack = e.undoStack[1:]
	}
	e.redoStack = nil
}

func (e *Editor) Undo() bool {
	if len(e.undoStack) == 0 {
		return false
	}
	current := e.currentState()
	e.redoStack = append(e.redoStack, current)

	state := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.textarea.SetValue(state.content)
	e.textarea.SetCursorColumn(state.cursorPos)
	return true
}

func (e *Editor) Redo() bool {
	if len(e.redoStack) == 0 {
		return false
	}
	current := e.currentState()
	e.undoStack = append(e.undoStack, current)

	state := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.textarea.SetValue(state.content)
	e.textarea.SetCursorColumn(state.cursorPos)
	return true
}
