package ui

import "testing"

func TestCursorNavDown(t *testing.T) {
	n := NewCursorNav()
	n.Down(10, 5)
	if n.Selected != 1 || n.Scroll != 0 {
		t.Errorf("got Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
	for range 5 {
		n.Down(10, 5)
	}
	if n.Selected != 6 || n.Scroll != 2 {
		t.Errorf("after 6 downs: Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavDownAtEnd(t *testing.T) {
	n := NewCursorNav()
	for range 20 {
		n.Down(5, 3)
	}
	if n.Selected != 4 {
		t.Errorf("should clamp at end: Selected=%d", n.Selected)
	}
}

func TestCursorNavUp(t *testing.T) {
	n := NewCursorNav()
	n.Selected = 3
	n.Scroll = 2
	n.Up(10, 5)
	if n.Selected != 2 || n.Scroll != 2 {
		t.Errorf("got Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
	n.Up(10, 5)
	if n.Scroll != 1 {
		t.Errorf("scroll should follow: Scroll=%d", n.Scroll)
	}
}

func TestCursorNavUpAtTop(t *testing.T) {
	n := NewCursorNav()
	n.Up(10, 5)
	if n.Selected != 0 || n.Scroll != 0 {
		t.Errorf("should stay at 0: Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavPageDown(t *testing.T) {
	n := NewCursorNav()
	n.PageDown(20, 5)
	if n.Selected != 5 || n.Scroll != 1 {
		t.Errorf("got Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavPageDownClamp(t *testing.T) {
	n := NewCursorNav()
	n.PageDown(3, 5)
	if n.Selected != 2 {
		t.Errorf("should clamp to last: Selected=%d", n.Selected)
	}
}

func TestCursorNavPageUp(t *testing.T) {
	n := NewCursorNav()
	n.Selected = 8
	n.Scroll = 4
	n.PageUp(20, 5)
	if n.Selected != 3 {
		t.Errorf("got Selected=%d", n.Selected)
	}
}

func TestCursorNavPageUpClamp(t *testing.T) {
	n := NewCursorNav()
	n.Selected = 2
	n.PageUp(20, 5)
	if n.Selected != 0 || n.Scroll != 0 {
		t.Errorf("should clamp: Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavHome(t *testing.T) {
	n := NewCursorNav()
	n.Selected = 15
	n.Scroll = 10
	n.Home()
	if n.Selected != 0 || n.Scroll != 0 {
		t.Errorf("got Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavEnd(t *testing.T) {
	n := NewCursorNav()
	n.End(20, 5)
	if n.Selected != 19 || n.Scroll != 15 {
		t.Errorf("got Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavEndShortList(t *testing.T) {
	n := NewCursorNav()
	n.End(3, 5)
	if n.Selected != 2 || n.Scroll != 0 {
		t.Errorf("short list: Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestCursorNavWheel(t *testing.T) {
	n := NewCursorNav()
	n.Wheel(3, 20, 5)
	if n.Scroll != 3 {
		t.Errorf("wheel down: Scroll=%d", n.Scroll)
	}
	n.Wheel(-3, 20, 5)
	if n.Scroll != 0 {
		t.Errorf("wheel up: Scroll=%d", n.Scroll)
	}
}

func TestCursorNavEmptyList(t *testing.T) {
	n := NewCursorNav()
	n.Down(0, 5)
	n.Up(0, 5)
	n.PageDown(0, 5)
	n.End(0, 5)
	if n.Selected != 0 || n.Scroll != 0 {
		t.Errorf("empty list: Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}

func TestScrollNavDown(t *testing.T) {
	n := NewScrollNav()
	n.Down(20, 5)
	if n.Scroll != 1 {
		t.Errorf("got Scroll=%d", n.Scroll)
	}
}

func TestScrollNavDownClamp(t *testing.T) {
	n := NewScrollNav()
	for range 30 {
		n.Down(10, 5)
	}
	if n.Scroll != 5 {
		t.Errorf("should clamp: Scroll=%d", n.Scroll)
	}
}

func TestScrollNavPageDown(t *testing.T) {
	n := NewScrollNav()
	n.PageDown(100, 10)
	if n.Scroll != 10 {
		t.Errorf("got Scroll=%d", n.Scroll)
	}
}

func TestScrollNavHome(t *testing.T) {
	n := NewScrollNav()
	n.Scroll = 50
	n.Home()
	if n.Scroll != 0 {
		t.Errorf("got Scroll=%d", n.Scroll)
	}
}

func TestScrollNavEnd(t *testing.T) {
	n := NewScrollNav()
	n.End(100, 10)
	if n.Scroll != 90 {
		t.Errorf("got Scroll=%d", n.Scroll)
	}
}

func TestHandleKeyIntegration(t *testing.T) {
	n := NewCursorNav()
	tests := []struct {
		key     string
		handled bool
	}{
		{"down", true},
		{"up", true},
		{"j", true},
		{"k", true},
		{"pgdown", true},
		{"pgup", true},
		{"home", true},
		{"end", true},
		{"q", false},
		{"enter", false},
	}
	for _, tt := range tests {
		if got := n.HandleKey(tt.key, 10, 5); got != tt.handled {
			t.Errorf("HandleKey(%q) = %v, want %v", tt.key, got, tt.handled)
		}
	}
}

func TestHandleWheelIntegration(t *testing.T) {
	n := NewCursorNav()
	n.HandleWheel(false, 20, 5)
	if n.Scroll != 0 {
		t.Errorf("wheel up from 0: Scroll=%d", n.Scroll)
	}
	n.HandleWheel(true, 20, 5)
	if n.Scroll != 3 {
		t.Errorf("wheel down: Scroll=%d", n.Scroll)
	}
}

func TestCursorNavClampOnShrink(t *testing.T) {
	n := NewCursorNav()
	n.Selected = 5
	n.Up(3, 5)
	if n.Selected != 1 {
		t.Errorf("should clamp then move up: Selected=%d", n.Selected)
	}

	n.Selected = 5
	n.Down(3, 5)
	if n.Selected != 2 {
		t.Errorf("should clamp at end: Selected=%d", n.Selected)
	}
}

func TestNavReset(t *testing.T) {
	n := NewCursorNav()
	n.Selected = 10
	n.Scroll = 5
	n.Reset()
	if n.Selected != 0 || n.Scroll != 0 {
		t.Errorf("after reset: Selected=%d Scroll=%d", n.Selected, n.Scroll)
	}
}
