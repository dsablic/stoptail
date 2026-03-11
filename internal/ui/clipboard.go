package ui

import (
	"regexp"

	tea "charm.land/bubbletea/v2"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type Clipboard struct {
	message string
}

func NewClipboard() Clipboard {
	return Clipboard{}
}

func (c *Clipboard) Copy(text string) tea.Cmd {
	text = ansiPattern.ReplaceAllString(text, "")
	if text == "" {
		c.message = "Nothing to copy"
		return nil
	}
	c.message = "Copied!"
	return tea.SetClipboard(text)
}

func (c *Clipboard) Paste() tea.Cmd {
	return func() tea.Msg {
		return tea.ReadClipboard()
	}
}

func (c *Clipboard) Message() string {
	return c.message
}

func (c *Clipboard) ClearMessage() {
	c.message = ""
}
