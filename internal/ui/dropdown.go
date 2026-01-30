package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DropdownAction int

const (
	DropdownActionNone DropdownAction = iota
	DropdownActionSelect
	DropdownActionClose
)

type Dropdown struct {
	items       []string
	selectedIdx int
	open        bool
	x           int
	y           int
}

func NewDropdown(items []string) Dropdown {
	return Dropdown{
		items:       items,
		selectedIdx: 0,
	}
}

func (d *Dropdown) SetItems(items []string) {
	d.items = items
	if d.selectedIdx >= len(items) {
		d.selectedIdx = 0
	}
}

func (d *Dropdown) SetPosition(x, y int) {
	d.x = x
	d.y = y
}

func (d Dropdown) Open() bool {
	return d.open
}

func (d *Dropdown) Toggle() {
	d.open = !d.open
}

func (d *Dropdown) Show() {
	d.open = true
}

func (d *Dropdown) Close() {
	d.open = false
}

func (d Dropdown) SelectedIdx() int {
	return d.selectedIdx
}

func (d *Dropdown) SetSelectedIdx(idx int) {
	if idx >= 0 && idx < len(d.items) {
		d.selectedIdx = idx
	}
}

func (d Dropdown) Selected() string {
	if d.selectedIdx >= 0 && d.selectedIdx < len(d.items) {
		return d.items[d.selectedIdx]
	}
	return ""
}

func (d *Dropdown) HandleKey(msg tea.KeyMsg) DropdownAction {
	if !d.open {
		if msg.String() == "enter" || msg.String() == " " {
			d.open = true
			return DropdownActionNone
		}
		return DropdownActionNone
	}

	switch msg.String() {
	case "up":
		d.selectedIdx = (d.selectedIdx - 1 + len(d.items)) % len(d.items)
		return DropdownActionNone
	case "down":
		d.selectedIdx = (d.selectedIdx + 1) % len(d.items)
		return DropdownActionNone
	case "enter", " ":
		d.open = false
		return DropdownActionSelect
	case "esc":
		d.open = false
		return DropdownActionClose
	}
	return DropdownActionNone
}

func (d *Dropdown) HandleClick(clickX, clickY int) DropdownAction {
	if !d.open {
		return DropdownActionNone
	}

	dropdownHeight := len(d.items) + 2
	dropdownWidth := d.width()

	if clickY >= d.y && clickY < d.y+dropdownHeight &&
		clickX >= d.x && clickX < d.x+dropdownWidth {
		itemIdx := clickY - d.y - 1
		if itemIdx >= 0 && itemIdx < len(d.items) {
			d.selectedIdx = itemIdx
			d.open = false
			return DropdownActionSelect
		}
	}

	d.open = false
	return DropdownActionClose
}

func (d Dropdown) width() int {
	maxLen := 0
	for _, item := range d.items {
		if len(item) > maxLen {
			maxLen = len(item)
		}
	}
	return maxLen + 6
}

func (d Dropdown) Render() string {
	maxLen := 0
	for _, item := range d.items {
		if len(item) > maxLen {
			maxLen = len(item)
		}
	}

	var items []string
	for i, item := range d.items {
		prefix := "  "
		if i == d.selectedIdx {
			prefix = "> "
		}
		items = append(items, fmt.Sprintf("%s%-*s ", prefix, maxLen, item))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Render(strings.Join(items, "\n"))
}

func (d Dropdown) Overlay(view string) string {
	if !d.open {
		return view
	}

	bgLines := strings.Split(view, "\n")
	dropdownLines := strings.Split(d.Render(), "\n")

	for i, dropLine := range dropdownLines {
		targetRow := d.y + i
		if targetRow >= len(bgLines) {
			break
		}
		bgLines[targetRow] = strings.Repeat(" ", d.x) + dropLine
	}

	return strings.Join(bgLines, "\n")
}
