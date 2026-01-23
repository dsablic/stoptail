package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
	sitter "github.com/smacker/go-tree-sitter"
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
	Dragging              bool
}

type Editor struct {
	textarea        textarea.Model
	width           int
	height          int
	gutterWidth     int
	client          *es.Client
	index           string
	validationState ValidationState
	validationError string
	selection       Selection
	cursorLine      int
	cursorCol       int
	cursorSet       bool
}

func NewEditor() Editor {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = 50000
	return Editor{
		textarea:    ta,
		gutterWidth: 4,
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

func (e Editor) renderGutter(width, height int) string {
	content := e.textarea.Value()
	lineCount := 1 + strings.Count(content, "\n")
	if content == "" {
		lineCount = 1
	}

	gutterStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(width).
		Align(lipgloss.Right)

	var lines []string
	for i := 1; i <= lineCount && i <= height; i++ {
		lines = append(lines, gutterStyle.Render(fmt.Sprintf("%d", i)))
	}
	return strings.Join(lines, "\n")
}

func (e Editor) parse() *sitter.Tree {
	content := e.textarea.Value()
	if content == "" {
		return nil
	}
	parser := sitter.NewParser()
	parser.SetLanguage(jsonLanguage)
	return parser.Parse(nil, []byte(content))
}

func (e Editor) highlightContent() string {
	content := e.textarea.Value()
	if content == "" {
		return ""
	}

	tree := e.parse()
	if tree == nil {
		return content
	}

	return e.applyHighlighting(content, tree.RootNode())
}

func (e Editor) getASTPath(cursorOffset int) []string {
	tree := e.parse()
	if tree == nil {
		return nil
	}

	content := []byte(e.textarea.Value())

	row, col := e.offsetToRowCol(cursorOffset)

	root := tree.RootNode()
	node := root.NamedDescendantForPointRange(
		sitter.Point{Row: uint32(row), Column: uint32(col)},
		sitter.Point{Row: uint32(row), Column: uint32(col)},
	)

	var path []string
	cursorNode := node
	for node != nil {
		if node.Type() == "pair" {
			keyNode := node.ChildByFieldName("key")
			if keyNode == nil && node.ChildCount() > 0 {
				keyNode = node.Child(0)
			}
			if keyNode != nil && keyNode.Type() == "string" {
				if !nodeContains(keyNode, cursorNode) {
					start := keyNode.StartByte() + 1
					end := keyNode.EndByte() - 1
					if start < end && int(end) <= len(content) {
						key := string(content[start:end])
						path = append([]string{key}, path...)
					}
				}
			}
		}
		node = node.Parent()
	}

	return path
}

func nodeContains(ancestor, descendant *sitter.Node) bool {
	for n := descendant; n != nil; n = n.Parent() {
		if n.Equal(ancestor) {
			return true
		}
	}
	return false
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

	var row, col int
	if e.cursorSet {
		row = e.cursorLine
		col = e.cursorCol
	} else {
		row = e.Line()
		col = e.LineInfo().CharOffset
	}

	offset := 0
	for i := 0; i < row && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	if row < len(lines) && col > len(lines[row]) {
		col = len(lines[row])
	}
	offset += col
	return offset
}

func (e Editor) offsetToRowCol(offset int) (row, col int) {
	content := e.textarea.Value()
	for i, ch := range content {
		if i >= offset {
			break
		}
		if ch == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return row, col
}

func (e Editor) applyHighlighting(content string, root *sitter.Node) string {
	type highlight struct {
		start, end int
		color      lipgloss.Color
	}
	var highlights []highlight

	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		nodeType := node.Type()
		start := int(node.StartByte())
		end := int(node.EndByte())

		switch nodeType {
		case "string":
			parent := node.Parent()
			if parent != nil && parent.Type() == "pair" {
				firstChild := parent.Child(0)
				if firstChild != nil && firstChild.Equal(node) {
					highlights = append(highlights, highlight{start, end, ColorBlue})
				} else {
					highlights = append(highlights, highlight{start, end, ColorGreen})
				}
			} else {
				highlights = append(highlights, highlight{start, end, ColorGreen})
			}
		case "number":
			highlights = append(highlights, highlight{start, end, ColorYellow})
		case "true", "false":
			highlights = append(highlights, highlight{start, end, lipgloss.Color("#c586c0")})
		case "null":
			highlights = append(highlights, highlight{start, end, lipgloss.Color("#c586c0")})
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(root)

	if len(highlights) == 0 {
		return content
	}

	var result strings.Builder
	lastEnd := 0
	for _, h := range highlights {
		if h.start > lastEnd {
			result.WriteString(content[lastEnd:h.start])
		}
		style := lipgloss.NewStyle().Foreground(h.color)
		result.WriteString(style.Render(content[h.start:h.end]))
		lastEnd = h.end
	}
	if lastEnd < len(content) {
		result.WriteString(content[lastEnd:])
	}
	return result.String()
}

func (e *Editor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.textarea.SetWidth(width - e.gutterWidth - 2)
	e.textarea.SetHeight(height)
}

func (e Editor) screenToPosition(x, y int) (line, col int) {
	content := e.textarea.Value()
	lines := strings.Split(content, "\n")
	gutterWidth := 3
	if len(lines) >= 100 {
		gutterWidth = 4
	}
	separatorWidth := 3
	adjustedX := x - gutterWidth - separatorWidth
	if adjustedX < 0 {
		adjustedX = 0
	}
	if y < 0 {
		y = 0
	}
	return y, adjustedX
}

func (e *Editor) HandleClick(x, y int) {
	line, col := e.screenToPosition(x, y)
	e.setCursorPosition(line, col)
	e.selection.Active = false
}

func (e *Editor) HandleDragStart(x, y int) {
	line, col := e.screenToPosition(x, y)
	e.selection.StartLine = line
	e.selection.StartCol = col
	e.selection.EndLine = line
	e.selection.EndCol = col
	e.selection.AnchorLine = line
	e.selection.AnchorCol = col
	e.selection.Active = false
	e.selection.Dragging = true
}

func (e *Editor) HandleDrag(x, y int) {
	if !e.selection.Dragging {
		return
	}
	line, col := e.screenToPosition(x, y)
	e.selection.EndLine = line
	e.selection.EndCol = col
	if line != e.selection.StartLine || col != e.selection.StartCol {
		e.selection.Active = true
	}
}

func (e *Editor) HandleDragEnd() {
	e.selection.Dragging = false
}

func (e *Editor) setCursorPosition(line, col int) {
	lines := strings.Split(e.textarea.Value(), "\n")
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	if line < 0 {
		return
	}
	lineLen := len(lines[line])
	if col < 0 {
		col = 0
	}
	if col > lineLen {
		col = lineLen
	}
	e.cursorLine = line
	e.cursorCol = col
	e.cursorSet = true

	offset := 0
	for i := 0; i < line && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	offset += col
	e.textarea.SetCursor(offset)
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

func (e Editor) renderPlainWithCursor(content string, cursorLine, cursorCol int) string {
	highlighted := e.highlightContent()
	if highlighted == "" {
		highlighted = content
	}

	lines := strings.Split(highlighted, "\n")
	if cursorLine < 0 || cursorLine >= len(lines) {
		return highlighted
	}

	contentLines := strings.Split(content, "\n")
	if cursorLine >= len(contentLines) {
		return highlighted
	}

	plainLine := contentLines[cursorLine]
	plainRunes := []rune(plainLine)
	if cursorCol > len(plainRunes) {
		cursorCol = len(plainRunes)
	}

	highlightedLine := lines[cursorLine]
	lines[cursorLine] = insertCursorInHighlightedLine(highlightedLine, plainRunes, cursorCol)
	return strings.Join(lines, "\n")
}

func insertCursorInHighlightedLine(highlighted string, plainRunes []rune, cursorCol int) string {
	cursorStyle := lipgloss.NewStyle().Reverse(true)

	if cursorCol >= len(plainRunes) {
		return highlighted + cursorStyle.Render(" ")
	}

	charAtCursor := string(plainRunes[cursorCol])
	startPos, endPos := findCharBounds(highlighted, cursorCol)

	before := highlighted[:startPos]
	after := highlighted[endPos:]

	activeColor := extractActiveColor(before)
	var cursorChar string
	if activeColor != "" {
		cursorChar = "\x1b[7;" + activeColor + "m" + charAtCursor + "\x1b[0m"
	} else {
		cursorChar = cursorStyle.Render(charAtCursor)
	}

	return before + cursorChar + after
}

func findCharBounds(s string, visualCol int) (start, end int) {
	visualPos := 0
	i := 0

	for i < len(s) && visualPos < visualCol {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		_, size := utf8DecodeRune(s[i:])
		visualPos++
		i += size
	}

	for i < len(s) && s[i] == '\x1b' {
		for i < len(s) && s[i] != 'm' {
			i++
		}
		if i < len(s) {
			i++
		}
	}
	start = i

	if i < len(s) {
		_, size := utf8DecodeRune(s[i:])
		end = i + size
	} else {
		end = i
	}

	for end < len(s) && s[end] == '\x1b' {
		for end < len(s) && s[end] != 'm' {
			end++
		}
		if end < len(s) {
			end++
		}
	}

	return start, end
}

func extractActiveColor(s string) string {
	lastColor := ""
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			start := i + 2
			end := start
			for end < len(s) && s[end] != 'm' {
				end++
			}
			if end < len(s) {
				code := s[start:end]
				if code == "0" {
					lastColor = ""
				} else {
					lastColor = code
				}
			}
			i = end + 1
		} else {
			i++
		}
	}
	return lastColor
}

func utf8DecodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	for i := 1; i <= 4 && i <= len(s); i++ {
		r := []rune(s[:i])
		if len(r) == 1 {
			return r[0], i
		}
	}
	return rune(s[0]), 1
}

func (e Editor) View() string {
	content := e.textarea.Value()
	lines := strings.Split(content, "\n")
	lineCount := len(lines)

	gutterWidth := 3
	if lineCount >= 100 {
		gutterWidth = 4
	}

	var cursorLine, cursorCol int
	if e.cursorSet {
		cursorLine = e.cursorLine
		cursorCol = e.cursorCol
	} else {
		cursorLine = e.textarea.Line()
		cursorCol = e.textarea.LineInfo().CharOffset
	}

	var displayContent string
	if e.selection.Active {
		displayContent = e.renderWithSelection(content)
	} else if e.textarea.Focused() {
		displayContent = e.renderPlainWithCursor(content, cursorLine, cursorCol)
	} else {
		displayContent = e.highlightContent()
	}

	displayLines := strings.Split(displayContent, "\n")

	gutterStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(gutterWidth).
		Align(lipgloss.Right)

	separatorStyle := lipgloss.NewStyle().Foreground(ColorGray)

	var resultLines []string
	visibleLines := e.height
	if visibleLines == 0 {
		visibleLines = lineCount
	}
	if visibleLines > lineCount {
		visibleLines = lineCount
	}

	for i := 0; i < visibleLines; i++ {
		lineNum := gutterStyle.Render(fmt.Sprintf("%d", i+1))
		separator := separatorStyle.Render(" \u2502 ")
		lineContent := ""
		if i < len(displayLines) {
			lineContent = displayLines[i]
		}
		resultLines = append(resultLines, lineNum+separator+lineContent)
	}

	return strings.Join(resultLines, "\n")
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
		e.cursorSet = false
		switch keyMsg.String() {
		case "shift+left", "shift+right", "shift+up", "shift+down", "shift+home", "shift+end":
			return e.handleShiftArrow(keyMsg)
		case "left", "right", "up", "down", "home", "end":
			e.selection.Active = false
		}
	}

	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return cmd
}

func (e *Editor) handleShiftArrow(msg tea.KeyMsg) tea.Cmd {
	curLine := e.textarea.Line()
	curCol := e.textarea.LineInfo().CharOffset

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
	newCol := e.textarea.LineInfo().CharOffset

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

func (e *Editor) InsertString(s string) {
	e.textarea.InsertString(s)
}

func (e *Editor) SetCursor(pos int) {
	e.textarea.SetCursor(pos)
}

func (e *Editor) ClearSelection() {
	e.selection.Active = false
	e.selection.Dragging = false
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
	e.selection.Dragging = false
}
