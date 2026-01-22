package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Modal struct {
	title     string
	prompt    string
	input     textinput.Model
	err       string
	done      bool
	cancelled bool
}

func NewModal(title, prompt string) *Modal {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 30

	return &Modal{
		title:  title,
		prompt: prompt,
		input:  ti,
	}
}

func (m *Modal) SetError(err string) {
	m.err = err
}

func (m *Modal) ClearError() {
	m.err = ""
}

func (m *Modal) Value() string {
	return m.input.Value()
}

func (m *Modal) SetValue(v string) {
	m.input.SetValue(v)
}

func (m *Modal) Done() bool {
	return m.done
}

func (m *Modal) Cancelled() bool {
	return m.cancelled
}

func (m *Modal) Reset(title, prompt string) {
	m.title = title
	m.prompt = prompt
	m.input.SetValue("")
	m.err = ""
	m.done = false
	m.cancelled = false
	m.input.Focus()
}

func (m *Modal) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.done = true
			return nil
		case "esc":
			m.cancelled = true
			return nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *Modal) View(width, height int) string {
	boxWidth := 50

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		MarginBottom(1)

	promptStyle := lipgloss.NewStyle().
		MarginBottom(1)

	errorStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		MarginTop(1)

	var content string
	content += titleStyle.Render(m.title) + "\n"
	content += promptStyle.Render(m.prompt) + "\n"
	content += m.input.View() + "\n"

	if m.err != "" {
		content += errorStyle.Render(m.err) + "\n"
	}

	content += helpStyle.Render("Enter: confirm | Esc: cancel")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
