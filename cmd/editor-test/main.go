package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/labtiva/stoptail/internal/ui"
)

const (
	editorOffsetX = 1
	editorOffsetY = 3
)

type model struct {
	editor ui.Editor
	width  int
	height int
}

func newModel() model {
	e := ui.NewEditor()
	e.SetContent(`{
  "query": {
    "bool": {
      "must": []
    }
  }
}`)
	e.Focus()
	return model{editor: e}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		case "ctrl+a":
			m.editor.SelectAll()
			return m, nil
		case "esc":
			m.editor.ClearSelection()
			return m, nil
		}
		cmd := m.editor.Update(msg)
		return m, cmd

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			m.editor.BeginMouseSelection(msg.X-editorOffsetX, msg.Y-editorOffsetY)
		}
		return m, nil
	case tea.MouseMotionMsg:
		m.editor.ExtendMouseSelection(msg.X-editorOffsetX, msg.Y-editorOffsetY)
		return m, nil
	case tea.MouseReleaseMsg:
		m.editor.EndMouseSelection()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetSize(msg.Width-2, msg.Height-6)
		return m, nil
	}

	return m, nil
}

func (m model) View() tea.View {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7aa2f7")).
		Padding(0, 1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3b4261"))

	title := titleStyle.Render("Query Editor Test")

	cursorLine := m.editor.Line()
	cursorCol := m.editor.LineInfo().CharOffset

	selectedText := m.editor.GetSelectedText()
	var stateInfo string
	if selectedText != "" {
		stateInfo = fmt.Sprintf("Selected: %q", selectedText)
	} else {
		stateInfo = fmt.Sprintf("Cursor: line %d, col %d", cursorLine+1, cursorCol+1)
	}

	info := infoStyle.Render(stateInfo + " | Ctrl+Q: quit | Ctrl+A: select all | Drag: select | Esc: clear selection")

	editorView := m.editor.View()
	editorBox := borderStyle.Render(editorView)

	content := fmt.Sprintf("%s\n%s\n%s\n", title, info, editorBox)
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func runTests() {
	fmt.Println("=== Editor Test Suite ===")
	fmt.Println()
	passed := 0
	failed := 0

	check := func(name string, ok bool, details string) {
		if ok {
			fmt.Printf("[PASS] %s\n", name)
			passed++
		} else {
			fmt.Printf("[FAIL] %s: %s\n", name, details)
			failed++
		}
	}

	e := ui.NewEditor()
	e.SetContent(`{"query": {"match_all": {}}}`)
	e.SetSize(60, 10)

	fmt.Println("--- Basic View ---")
	view1 := e.View()
	fmt.Println(view1)
	fmt.Println()

	hasContent := strings.Contains(view1, "query")
	check("View contains content", hasContent, "content not found")

	shift := func(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: tea.ModShift} }
	newFocused := func(content string) ui.Editor {
		ed := ui.NewEditor()
		ed.SetContent(content)
		ed.SetSize(60, 10)
		ed.Focus()
		ed.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl})
		return ed
	}

	fmt.Println("\n--- Shift+Arrow selection ---")
	e3 := newFocused("hello world")
	for range 3 {
		e3.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	for range 3 {
		e3.Update(shift(tea.KeyRight))
	}
	check("Shift+arrow creates selection", e3.HasSelection(), "selection not active after shift+right")
	check("Shift+right selects correct text", e3.GetSelectedText() == "lo ",
		fmt.Sprintf("got %q, expected \"lo \"", e3.GetSelectedText()))

	view := e3.View()
	fmt.Println(view)
	check("Selection view has reverse video", strings.Contains(view, "\x1b[7m"), "no selection highlight")

	e3.Update(shift(tea.KeyLeft))
	check("Shift+left shrinks selection", e3.GetSelectedText() == "lo",
		fmt.Sprintf("got %q, expected \"lo\"", e3.GetSelectedText()))

	e3.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	check("Regular arrow clears selection", !e3.HasSelection(), "selection still active after non-shift arrow")

	fmt.Println("\n--- Shift+Home/End selection ---")
	e5 := newFocused("hello world\nline two")
	for range 6 {
		e5.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	e5.Update(shift(tea.KeyEnd))
	check("Shift+End selects to line end", e5.GetSelectedText() == "world",
		fmt.Sprintf("got %q", e5.GetSelectedText()))
	e5.Update(shift(tea.KeyHome))
	check("Shift+Home selects to line start", e5.GetSelectedText() == "hello ",
		fmt.Sprintf("got %q", e5.GetSelectedText()))

	fmt.Println("\n--- Multi-line and word selection ---")
	e6 := newFocused("hello world\nline two")
	e6.Update(shift(tea.KeyDown))
	check("Shift+down selects across lines", e6.GetSelectedText() == "hello world\n",
		fmt.Sprintf("got %q", e6.GetSelectedText()))
	e6.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl})
	e6.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl | tea.ModShift})
	check("Ctrl+Shift+right selects a word", e6.GetSelectedText() == "hello",
		fmt.Sprintf("got %q", e6.GetSelectedText()))

	fmt.Println("\n--- Editing with a selection ---")
	e7 := newFocused("hello world")
	e7.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl | tea.ModShift})
	e7.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	check("Typing replaces selection", e7.Content() == "j world", fmt.Sprintf("got %q", e7.Content()))
	e7.Undo()
	check("Undo restores replaced selection", e7.Content() == "hello world", fmt.Sprintf("got %q", e7.Content()))
	e7.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl})
	e7.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl | tea.ModShift})
	e7.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	check("Backspace deletes selection", e7.Content() == " world", fmt.Sprintf("got %q", e7.Content()))

	fmt.Println("\n--- SelectAll and Delete ---")
	e4 := newFocused("test content")
	e4.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	check("Ctrl+A selects entire content", e4.GetSelectedText() == "test content",
		fmt.Sprintf("got %q", e4.GetSelectedText()))
	e4.DeleteSelection()
	check("DeleteSelection clears content", e4.Content() == "", fmt.Sprintf("got %q", e4.Content()))
	check("Selection inactive after delete", !e4.HasSelection(), "selection still active")

	e4.SetContent("again")
	e4.SelectAll()
	e4.ClearSelection()
	check("ClearSelection clears selection", !e4.HasSelection(), "selection still active")

	fmt.Println("\n--- Mouse selection ---")
	e8 := newFocused("hello world\nline two")
	e8.BeginMouseSelection(0, 0)
	e8.ExtendMouseSelection(5, 0)
	e8.EndMouseSelection()
	check("Mouse drag selects text", e8.GetSelectedText() == "hello", fmt.Sprintf("got %q", e8.GetSelectedText()))
	e8.BeginMouseSelection(6, 0)
	e8.ExtendMouseSelection(4, 1)
	e8.EndMouseSelection()
	check("Mouse drag selects across lines", e8.GetSelectedText() == "world\nline",
		fmt.Sprintf("got %q", e8.GetSelectedText()))
	e8.BeginMouseSelection(2, 1)
	e8.EndMouseSelection()
	check("Mouse click clears selection", !e8.HasSelection(), "selection still active after click")
	check("Mouse click moves cursor", e8.Line() == 1 && e8.LineInfo().CharOffset == 2,
		fmt.Sprintf("got line %d col %d", e8.Line(), e8.LineInfo().CharOffset))

	fmt.Println("\n=== Results ===")
	fmt.Printf("Passed: %d, Failed: %d\n", passed, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func main() {
	testMode := flag.Bool("test", false, "Run non-interactive tests")
	flag.Parse()

	if *testMode {
		runTests()
		return
	}

	p := tea.NewProgram(newModel())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
