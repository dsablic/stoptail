package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	sitter "github.com/smacker/go-tree-sitter"
)

type Editor struct {
	textarea    textarea.Model
	width       int
	height      int
	gutterWidth int
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
