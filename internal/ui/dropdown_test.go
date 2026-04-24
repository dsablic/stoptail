package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewDropdown(t *testing.T) {
	d := NewDropdown([]string{"GET", "POST", "PUT"})
	if d.Open() {
		t.Error("new dropdown should be closed")
	}
	if d.SelectedIdx() != 0 {
		t.Errorf("initial selected = %d, want 0", d.SelectedIdx())
	}
	if d.Selected() != "GET" {
		t.Errorf("Selected() = %q, want %q", d.Selected(), "GET")
	}
}

func TestDropdownSetItems(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"})
	d.SetSelectedIdx(2)
	d.SetItems([]string{"x", "y"})
	if d.SelectedIdx() != 0 {
		t.Errorf("SetItems should reset selectedIdx when out of range, got %d", d.SelectedIdx())
	}
	if d.Selected() != "x" {
		t.Errorf("Selected() = %q, want %q", d.Selected(), "x")
	}
}

func TestDropdownSetItemsKeepsIdx(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"})
	d.SetSelectedIdx(1)
	d.SetItems([]string{"x", "y", "z"})
	if d.SelectedIdx() != 1 {
		t.Errorf("SetItems should keep selectedIdx when in range, got %d", d.SelectedIdx())
	}
}

func TestDropdownToggleShowClose(t *testing.T) {
	d := NewDropdown([]string{"a"})
	d.Toggle()
	if !d.Open() {
		t.Error("Toggle should open")
	}
	d.Toggle()
	if d.Open() {
		t.Error("Toggle should close")
	}
	d.Show()
	if !d.Open() {
		t.Error("Show should open")
	}
	d.Close()
	if d.Open() {
		t.Error("Close should close")
	}
}

func TestDropdownSetSelectedIdx(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"})
	d.SetSelectedIdx(2)
	if d.Selected() != "c" {
		t.Errorf("Selected() = %q, want %q", d.Selected(), "c")
	}
	d.SetSelectedIdx(-1)
	if d.SelectedIdx() != 2 {
		t.Error("SetSelectedIdx should ignore negative index")
	}
	d.SetSelectedIdx(10)
	if d.SelectedIdx() != 2 {
		t.Error("SetSelectedIdx should ignore out-of-range index")
	}
}

func TestDropdownSelectedEmpty(t *testing.T) {
	d := NewDropdown([]string{})
	if d.Selected() != "" {
		t.Errorf("Selected() on empty dropdown = %q, want empty", d.Selected())
	}
}

func keyMsg(key string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 0, Text: key}
}

func TestDropdownHandleKeyOpenClose(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"})

	action := d.HandleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !d.Open() {
		t.Error("enter when closed should open")
	}
	if action != DropdownActionNone {
		t.Errorf("action = %d, want DropdownActionNone", action)
	}

	action = d.HandleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if d.Open() {
		t.Error("esc when open should close")
	}
	if action != DropdownActionClose {
		t.Errorf("action = %d, want DropdownActionClose", action)
	}
}

func TestDropdownHandleKeyNavigation(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"})
	d.Show()

	d.HandleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if d.SelectedIdx() != 1 {
		t.Errorf("down: selectedIdx = %d, want 1", d.SelectedIdx())
	}

	d.HandleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if d.SelectedIdx() != 2 {
		t.Errorf("down: selectedIdx = %d, want 2", d.SelectedIdx())
	}

	d.HandleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if d.SelectedIdx() != 0 {
		t.Errorf("down should wrap: selectedIdx = %d, want 0", d.SelectedIdx())
	}

	d.HandleKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if d.SelectedIdx() != 2 {
		t.Errorf("up should wrap: selectedIdx = %d, want 2", d.SelectedIdx())
	}
}

func TestDropdownHandleKeySelect(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"})
	d.Show()
	d.HandleKey(tea.KeyPressMsg{Code: tea.KeyDown})

	action := d.HandleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d.Open() {
		t.Error("enter should close dropdown")
	}
	if action != DropdownActionSelect {
		t.Errorf("action = %d, want DropdownActionSelect", action)
	}
	if d.Selected() != "b" {
		t.Errorf("Selected() = %q, want %q", d.Selected(), "b")
	}
}

func TestDropdownHandleClickItem(t *testing.T) {
	d := NewDropdown([]string{"GET", "POST", "PUT"})
	d.SetPosition(5, 10)
	d.Show()

	action := d.HandleClick(7, 12)
	if action != DropdownActionSelect {
		t.Errorf("action = %d, want DropdownActionSelect", action)
	}
	if d.Open() {
		t.Error("click on item should close dropdown")
	}
	if d.SelectedIdx() != 1 {
		t.Errorf("selectedIdx = %d, want 1 (POST)", d.SelectedIdx())
	}
}

func TestDropdownHandleClickOutside(t *testing.T) {
	d := NewDropdown([]string{"GET", "POST"})
	d.SetPosition(5, 10)
	d.Show()

	action := d.HandleClick(100, 100)
	if action != DropdownActionClose {
		t.Errorf("click outside = %d, want DropdownActionClose", action)
	}
	if d.Open() {
		t.Error("click outside should close dropdown")
	}
}

func TestDropdownHandleClickWhenClosed(t *testing.T) {
	d := NewDropdown([]string{"GET"})
	action := d.HandleClick(0, 0)
	if action != DropdownActionNone {
		t.Errorf("click when closed = %d, want DropdownActionNone", action)
	}
}

func TestDropdownRender(t *testing.T) {
	d := NewDropdown([]string{"GET", "POST"})
	rendered := d.Render()
	if rendered == "" {
		t.Error("Render should produce non-empty output")
	}
}

func TestDropdownOverlayWhenClosed(t *testing.T) {
	d := NewDropdown([]string{"GET"})
	bg := "line1\nline2\nline3"
	result := d.Overlay(bg)
	if result != bg {
		t.Error("Overlay when closed should return background unchanged")
	}
}

func TestDropdownOverlayWhenOpen(t *testing.T) {
	d := NewDropdown([]string{"GET"})
	d.SetPosition(0, 0)
	d.Show()
	bg := "line1\nline2\nline3\nline4\nline5"
	result := d.Overlay(bg)
	if result == bg {
		t.Error("Overlay when open should modify the background")
	}
}

func TestDropdownSetPosition(t *testing.T) {
	d := NewDropdown([]string{"a"})
	d.SetPosition(10, 20)
	if d.x != 10 || d.y != 20 {
		t.Errorf("position = (%d, %d), want (10, 20)", d.x, d.y)
	}
}
