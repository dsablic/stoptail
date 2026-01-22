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
	StartLine, StartCol int
	EndLine, EndCol     int
	Active              bool
	Dragging            bool
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
	adjustedX := x - e.gutterWidth - 1
	if adjustedX < 0 {
		adjustedX = 0
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
	e.selection.Active = true
	e.selection.Dragging = true
}

func (e *Editor) HandleDrag(x, y int) {
	if !e.selection.Dragging {
		return
	}
	line, col := e.screenToPosition(x, y)
	e.selection.EndLine = line
	e.selection.EndCol = col
}

func (e *Editor) HandleDragEnd() {
	e.selection.Dragging = false
}

func (e *Editor) setCursorPosition(line, col int) {
	lines := strings.Split(e.textarea.Value(), "\n")
	offset := 0
	for i := 0; i < line && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	if line < len(lines) {
		lineLen := len(lines[line])
		if col > lineLen {
			col = lineLen
		}
		offset += col
	}
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
