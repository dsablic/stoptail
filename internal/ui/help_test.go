package ui

import (
	"testing"
	tea "charm.land/bubbletea/v2"
)

func TestHelpToggle(t *testing.T) {
	m := Model{}
	m.width = 120
	m.height = 40

	// Open help with ?
	k := tea.KeyPressMsg{Code: '?', Text: "?"}
	newM, _ := m.Update(k)
	m = newM.(Model)
	if !m.showHelp {
		t.Fatal("expected showHelp to be true after pressing ?")
	}

	// Close with esc
	k2 := tea.KeyPressMsg{Code: tea.KeyEscape}
	newM, _ = m.Update(k2)
	m = newM.(Model)
	if m.showHelp {
		t.Fatal("expected showHelp to be false after pressing esc")
	}

	// Open again with ?
	newM, _ = m.Update(k)
	m = newM.(Model)
	if !m.showHelp {
		t.Fatal("expected showHelp to be true after pressing ? again")
	}

	// Close with ?
	newM, _ = m.Update(k)
	m = newM.(Model)
	if m.showHelp {
		t.Fatal("expected showHelp to be false after pressing ? to close")
	}
}
