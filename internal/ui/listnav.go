package ui

const wheelDelta = 3

type ListNav struct {
	Selected int
	Scroll   int
	cursor   bool
}

func NewCursorNav() ListNav {
	return ListNav{cursor: true}
}

func NewScrollNav() ListNav {
	return ListNav{}
}

func (n *ListNav) Reset() {
	n.Selected = 0
	n.Scroll = 0
}

func (n *ListNav) Up(total, visible int) {
	if total <= 0 {
		return
	}
	if n.cursor {
		if n.Selected > 0 {
			n.Selected--
			if n.Selected < n.Scroll {
				n.Scroll = n.Selected
			}
		}
	} else {
		if n.Scroll > 0 {
			n.Scroll--
		}
	}
}

func (n *ListNav) Down(total, visible int) {
	if total <= 0 {
		return
	}
	if n.cursor {
		if n.Selected < total-1 {
			n.Selected++
			if n.Selected >= n.Scroll+visible {
				n.Scroll = n.Selected - visible + 1
			}
		}
	} else {
		ms := navMaxScroll(total, visible)
		if n.Scroll < ms {
			n.Scroll++
		}
	}
}

func (n *ListNav) PageUp(total, visible int) {
	if total <= 0 {
		return
	}
	if n.cursor {
		n.Selected -= visible
		if n.Selected < 0 {
			n.Selected = 0
		}
		if n.Selected < n.Scroll {
			n.Scroll = n.Selected
		}
	} else {
		n.Scroll -= visible
		if n.Scroll < 0 {
			n.Scroll = 0
		}
	}
}

func (n *ListNav) PageDown(total, visible int) {
	if total <= 0 {
		return
	}
	if n.cursor {
		n.Selected += visible
		if n.Selected >= total {
			n.Selected = total - 1
		}
		if n.Selected >= n.Scroll+visible {
			n.Scroll = n.Selected - visible + 1
		}
	} else {
		n.Scroll += visible
		ms := navMaxScroll(total, visible)
		if n.Scroll > ms {
			n.Scroll = ms
		}
	}
}

func (n *ListNav) Home() {
	n.Selected = 0
	n.Scroll = 0
}

func (n *ListNav) End(total, visible int) {
	if total <= 0 {
		return
	}
	if n.cursor {
		n.Selected = total - 1
		ms := navMaxScroll(total, visible)
		if ms > 0 {
			n.Scroll = ms
		}
	} else {
		n.Scroll = navMaxScroll(total, visible)
	}
}

func (n *ListNav) Wheel(delta, total, visible int) {
	n.Scroll += delta
	ms := navMaxScroll(total, visible)
	if n.Scroll < 0 {
		n.Scroll = 0
	}
	if n.Scroll > ms {
		n.Scroll = ms
	}
}

func (n *ListNav) HandleKey(key string, total, visible int) bool {
	switch key {
	case "up", "k":
		n.Up(total, visible)
	case "down", "j":
		n.Down(total, visible)
	case "pgup":
		n.PageUp(total, visible)
	case "pgdown":
		n.PageDown(total, visible)
	case "home":
		n.Home()
	case "end":
		n.End(total, visible)
	default:
		return false
	}
	return true
}

func (n *ListNav) HandleWheel(down bool, total, visible int) {
	delta := wheelDelta
	if !down {
		delta = -wheelDelta
	}
	n.Wheel(delta, total, visible)
}

func navMaxScroll(total, visible int) int {
	ms := total - visible
	if ms < 0 {
		return 0
	}
	return ms
}
