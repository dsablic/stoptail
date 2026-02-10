package ui

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type Selection struct {
	StartLine, StartCol   int
	EndLine, EndCol       int
	AnchorLine, AnchorCol int
	Active                bool
}

type Editor struct {
	textarea        textarea.Model
	width           int
	height          int
	client          *es.Client
	index           string
	validationState ValidationState
	validationError string
	selection       Selection
	undoStack       []editorState
	redoStack       []editorState
}

type editorState struct {
	content    string
	cursorPos  int
}

func NewEditor() Editor {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = 50000
	ta.Prompt = ""
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

func (e Editor) renderWithSelection(content string) string {
	if !e.selection.Active {
		return content
	}

	lines := strings.Split(content, "\n")
	selStyle := lipgloss.NewStyle().Reverse(true)

	startLine, startCol := e.selection.StartLine, e.selection.StartCol
	endLine, endCol := e.selection.EndLine, e.selection.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	var result []string
	for i, line := range lines {
		if i < startLine || i > endLine {
			result = append(result, line)
			continue
		}

		runes := []rune(line)
		selStart := 0
		selEnd := len(runes)

		if i == startLine {
			selStart = startCol
		}
		if i == endLine {
			selEnd = endCol
		}

		if selStart > len(runes) {
			selStart = len(runes)
		}
		if selEnd > len(runes) {
			selEnd = len(runes)
		}

		var lineResult string
		if selStart > 0 {
			lineResult += string(runes[:selStart])
		}
		if selEnd > selStart {
			lineResult += selStyle.Render(string(runes[selStart:selEnd]))
		}
		if selEnd < len(runes) {
			lineResult += string(runes[selEnd:])
		}

		result = append(result, lineResult)
	}

	return strings.Join(result, "\n")
}

func (e Editor) renderWithCursor(content string) string {
	lines := strings.Split(content, "\n")
	cursorLine := e.textarea.Line()
	cursorCol := e.logicalCol()

	if cursorLine < 0 || cursorLine >= len(lines) {
		return content
	}

	runes := []rune(lines[cursorLine])
	if cursorCol > len(runes) {
		cursorCol = len(runes)
	}

	cursorStyle := lipgloss.NewStyle().Reverse(true)
	if cursorCol >= len(runes) {
		lines[cursorLine] = string(runes) + cursorStyle.Render(" ")
	} else {
		lines[cursorLine] = string(runes[:cursorCol]) + cursorStyle.Render(string(runes[cursorCol])) + string(runes[cursorCol+1:])
	}
	return strings.Join(lines, "\n")
}

func (e Editor) View() string {
	content := e.textarea.Value()
	if e.selection.Active {
		return e.renderWithSelection(content)
	}
	if e.textarea.Focused() {
		return e.renderWithCursor(content)
	}
	return content
}

func (e Editor) GetSelectedText() string {
	if !e.selection.Active {
		return ""
	}

	content := e.textarea.Value()
	lines := strings.Split(content, "\n")

	startLine, startCol := e.selection.StartLine, e.selection.StartCol
	endLine, endCol := e.selection.EndLine, e.selection.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	if startLine == endLine {
		if startLine >= len(lines) {
			return ""
		}
		runes := []rune(lines[startLine])
		if startCol > len(runes) {
			startCol = len(runes)
		}
		if endCol > len(runes) {
			endCol = len(runes)
		}
		return string(runes[startCol:endCol])
	}

	var result []string
	for i := startLine; i <= endLine && i < len(lines); i++ {
		runes := []rune(lines[i])
		if i == startLine {
			if startCol < len(runes) {
				result = append(result, string(runes[startCol:]))
			}
		} else if i == endLine {
			if endCol > len(runes) {
				endCol = len(runes)
			}
			result = append(result, string(runes[:endCol]))
		} else {
			result = append(result, lines[i])
		}
	}

	return strings.Join(result, "\n")
}

func (e *Editor) Focus() {
	e.textarea.Focus()
}

func (e *Editor) Blur() {
	e.textarea.Blur()
}

func (e Editor) Focused() bool {
	return e.textarea.Focused()
}

func (e *Editor) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		key := keyMsg.String()
		switch key {
		case "shift+left", "shift+right", "shift+up", "shift+down", "shift+home", "shift+end":
			return e.handleShiftArrow(keyMsg)
		case "left", "right", "up", "down", "home", "end":
			e.selection.Active = false
		case "backspace", "delete":
			if e.selection.Active {
				e.SaveState()
				e.DeleteSelection()
				return nil
			}
			e.SaveState()
		case "enter":
			if e.selection.Active {
				e.SaveState()
				e.DeleteSelection()
			} else {
				e.SaveState()
			}
		default:
			if len(key) == 1 {
				if e.selection.Active {
					e.SaveState()
					e.DeleteSelection()
				} else {
					e.SaveState()
				}
			}
		}
	}

	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return cmd
}

func (e *Editor) handleShiftArrow(msg tea.KeyMsg) tea.Cmd {
	curLine := e.textarea.Line()
	curCol := e.logicalCol()

	if !e.selection.Active {
		e.selection.AnchorLine = curLine
		e.selection.AnchorCol = curCol
		e.selection.Active = true
	}

	var cmd tea.Cmd
	switch msg.String() {
	case "shift+left":
		e.textarea, cmd = e.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
	case "shift+right":
		e.textarea, cmd = e.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	case "shift+up":
		e.textarea, cmd = e.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
	case "shift+down":
		e.textarea, cmd = e.textarea.Update(tea.KeyMsg{Type: tea.KeyDown})
	case "shift+home":
		e.textarea, cmd = e.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})
	case "shift+end":
		e.textarea, cmd = e.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
	}

	newLine := e.textarea.Line()
	newCol := e.logicalCol()

	e.updateSelectionFromAnchor(newLine, newCol)

	return cmd
}

func (e *Editor) updateSelectionFromAnchor(curLine, curCol int) {
	anchorLine := e.selection.AnchorLine
	anchorCol := e.selection.AnchorCol

	if curLine < anchorLine || (curLine == anchorLine && curCol < anchorCol) {
		e.selection.StartLine = curLine
		e.selection.StartCol = curCol
		e.selection.EndLine = anchorLine
		e.selection.EndCol = anchorCol
	} else {
		e.selection.StartLine = anchorLine
		e.selection.StartCol = anchorCol
		e.selection.EndLine = curLine
		e.selection.EndCol = curCol
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
	e.textarea.SetCursor(pos)
}

func (e Editor) CursorOffset() int {
	return e.getCursorOffset()
}

func (e *Editor) ClearSelection() {
	e.selection.Active = false
}

func (e Editor) GetSelection() Selection {
	return e.selection
}

func (e *Editor) SetSelection(startLine, startCol, endLine, endCol int) {
	e.selection.StartLine = startLine
	e.selection.StartCol = startCol
	e.selection.EndLine = endLine
	e.selection.EndCol = endCol
	e.selection.Active = true
}

func (e *Editor) SelectAll() {
	content := e.textarea.Value()
	if content == "" {
		return
	}
	lines := strings.Split(content, "\n")
	lastLine := len(lines) - 1
	lastCol := len(lines[lastLine])
	e.SetSelection(0, 0, lastLine, lastCol)
}

func (e *Editor) DeleteSelection() {
	if !e.selection.Active {
		return
	}
	content := e.textarea.Value()
	lines := strings.Split(content, "\n")

	startLine, startCol := e.selection.StartLine, e.selection.StartCol
	endLine, endCol := e.selection.EndLine, e.selection.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	var result strings.Builder
	for i, line := range lines {
		if i < startLine {
			result.WriteString(line)
			result.WriteString("\n")
		} else if i == startLine && i == endLine {
			result.WriteString(line[:startCol])
			result.WriteString(line[endCol:])
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		} else if i == startLine {
			result.WriteString(line[:startCol])
		} else if i == endLine {
			result.WriteString(line[endCol:])
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		} else if i > endLine {
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		}
	}

	e.textarea.SetValue(result.String())
	offset := 0
	newLines := strings.Split(result.String(), "\n")
	for i := 0; i < startLine && i < len(newLines); i++ {
		offset += len(newLines[i]) + 1
	}
	offset += startCol
	e.textarea.SetCursor(offset)
	e.selection.Active = false
}

func (e *Editor) SaveState() {
	state := editorState{
		content:   e.textarea.Value(),
		cursorPos: e.getCursorOffset(),
	}
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
	current := editorState{
		content:   e.textarea.Value(),
		cursorPos: e.getCursorOffset(),
	}
	e.redoStack = append(e.redoStack, current)

	state := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.textarea.SetValue(state.content)
	e.textarea.SetCursor(state.cursorPos)
	return true
}

func (e *Editor) Redo() bool {
	if len(e.redoStack) == 0 {
		return false
	}
	current := editorState{
		content:   e.textarea.Value(),
		cursorPos: e.getCursorOffset(),
	}
	e.undoStack = append(e.undoStack, current)

	state := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.textarea.SetValue(state.content)
	e.textarea.SetCursor(state.cursorPos)
	return true
}
