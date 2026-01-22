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
