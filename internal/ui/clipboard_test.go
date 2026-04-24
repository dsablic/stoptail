package ui

import "testing"

func TestNewClipboard(t *testing.T) {
	c := NewClipboard()
	if c.Message() != "" {
		t.Error("new clipboard should have empty message")
	}
}

func TestClipboardCopyPlainText(t *testing.T) {
	c := NewClipboard()
	cmd := c.Copy("hello world")
	if cmd == nil {
		t.Error("Copy should return a tea.Cmd for non-empty text")
	}
	if c.Message() != "Copied!" {
		t.Errorf("message = %q, want %q", c.Message(), "Copied!")
	}
}

func TestClipboardCopyEmpty(t *testing.T) {
	c := NewClipboard()
	cmd := c.Copy("")
	if cmd != nil {
		t.Error("Copy of empty string should return nil cmd")
	}
	if c.Message() != "Nothing to copy" {
		t.Errorf("message = %q, want %q", c.Message(), "Nothing to copy")
	}
}

func TestClipboardCopyStripsANSI(t *testing.T) {
	c := NewClipboard()
	cmd := c.Copy("\x1b[31mred text\x1b[0m")
	if cmd == nil {
		t.Error("Copy should return a cmd after stripping ANSI")
	}
	if c.Message() != "Copied!" {
		t.Errorf("message = %q, want %q", c.Message(), "Copied!")
	}
}

func TestClipboardCopyOnlyANSI(t *testing.T) {
	c := NewClipboard()
	cmd := c.Copy("\x1b[31m\x1b[0m")
	if cmd != nil {
		t.Error("Copy of ANSI-only string should return nil cmd")
	}
	if c.Message() != "Nothing to copy" {
		t.Errorf("message = %q, want %q", c.Message(), "Nothing to copy")
	}
}

func TestClipboardClearMessage(t *testing.T) {
	c := NewClipboard()
	c.Copy("test")
	c.ClearMessage()
	if c.Message() != "" {
		t.Errorf("message should be empty after ClearMessage, got %q", c.Message())
	}
}

func TestClipboardPaste(t *testing.T) {
	c := NewClipboard()
	cmd := c.Paste()
	if cmd == nil {
		t.Error("Paste should return a tea.Cmd")
	}
}
