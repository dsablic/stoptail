package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SearchBar struct {
	input      textinput.Model
	matches    []int
	currentIdx int
	active     bool
}

func NewSearchBar() SearchBar {
	input := textinput.New()
	input.Placeholder = "Search..."
	input.CharLimit = 100
	input.Width = 30
	return SearchBar{input: input}
}

func (s *SearchBar) Active() bool {
	return s.active
}

func (s *SearchBar) Activate() {
	s.active = true
	s.input.Focus()
	s.input.SetValue("")
	s.matches = nil
	s.currentIdx = 0
}

func (s *SearchBar) Deactivate() {
	s.active = false
	s.input.Blur()
}

func (s *SearchBar) Query() string {
	return s.input.Value()
}

func (s *SearchBar) Matches() []int {
	return s.matches
}

func (s *SearchBar) CurrentMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	return s.matches[s.currentIdx]
}

func (s *SearchBar) FindMatches(lines []string) {
	query := strings.ToLower(s.input.Value())
	s.matches = nil
	s.currentIdx = 0

	if query == "" {
		return
	}

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			s.matches = append(s.matches, i)
		}
	}
}

func (s *SearchBar) NextMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	s.currentIdx = (s.currentIdx + 1) % len(s.matches)
	return s.matches[s.currentIdx]
}

func (s *SearchBar) PrevMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	s.currentIdx--
	if s.currentIdx < 0 {
		s.currentIdx = len(s.matches) - 1
	}
	return s.matches[s.currentIdx]
}

func (s *SearchBar) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

func (s *SearchBar) View(width int) string {
	status := ""
	if len(s.matches) > 0 {
		status = lipgloss.NewStyle().Foreground(ColorGray).Render(
			fmt.Sprintf(" %d/%d ", s.currentIdx+1, len(s.matches)))
	} else if s.input.Value() != "" {
		status = lipgloss.NewStyle().Foreground(ColorRed).Render(" No matches ")
	}

	navBtns := ""
	if len(s.matches) > 0 {
		navBtns = " [<] [>]"
	}

	return lipgloss.NewStyle().
		Background(ActiveBg).
		Padding(0, 1).
		Width(width).
		Render("/" + s.input.View() + status + navBtns + " [x]")
}
