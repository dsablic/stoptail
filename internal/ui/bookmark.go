package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/storage"
)

type BookmarkMode int

const (
	BookmarkModeNone BookmarkMode = iota
	BookmarkModeSave
	BookmarkModeLoad
)

type BookmarkUI struct {
	mode       BookmarkMode
	bookmarks  *storage.Bookmarks
	nameInput  textinput.Model
	selected   int
	scrollY    int
	maxVisible int
}

func NewBookmarkUI() BookmarkUI {
	ti := textinput.New()
	ti.Placeholder = "Bookmark name..."
	ti.CharLimit = 50
	ti.Width = 30

	bookmarks, _ := storage.LoadBookmarks()

	return BookmarkUI{
		mode:       BookmarkModeNone,
		bookmarks:  bookmarks,
		nameInput:  ti,
		maxVisible: 10,
	}
}

func (b *BookmarkUI) Active() bool {
	return b.mode != BookmarkModeNone
}

func (b *BookmarkUI) Mode() BookmarkMode {
	return b.mode
}

func (b *BookmarkUI) OpenSave() tea.Cmd {
	b.mode = BookmarkModeSave
	b.nameInput.SetValue("")
	b.nameInput.Focus()
	return textinput.Blink
}

func (b *BookmarkUI) OpenLoad() {
	b.mode = BookmarkModeLoad
	b.selected = 0
	b.scrollY = 0
	b.bookmarks, _ = storage.LoadBookmarks()
}

func (b *BookmarkUI) Close() {
	b.mode = BookmarkModeNone
	b.nameInput.Blur()
}

func (b *BookmarkUI) HandleKey(msg tea.KeyMsg) (action BookmarkAction, bookmark *storage.Bookmark) {
	switch msg.String() {
	case "esc":
		b.Close()
		return BookmarkActionCancel, nil
	case "enter":
		if b.mode == BookmarkModeSave {
			name := strings.TrimSpace(b.nameInput.Value())
			if name != "" {
				b.Close()
				return BookmarkActionSave, &storage.Bookmark{Name: name}
			}
		} else if b.mode == BookmarkModeLoad {
			if len(b.bookmarks.Items) > 0 && b.selected < len(b.bookmarks.Items) {
				bm := b.bookmarks.Items[b.selected]
				b.Close()
				return BookmarkActionLoad, &bm
			}
		}
	case "up", "k":
		if b.mode == BookmarkModeLoad && b.selected > 0 {
			b.selected--
			if b.selected < b.scrollY {
				b.scrollY = b.selected
			}
		}
	case "down", "j":
		if b.mode == BookmarkModeLoad && b.selected < len(b.bookmarks.Items)-1 {
			b.selected++
			if b.selected >= b.scrollY+b.maxVisible {
				b.scrollY = b.selected - b.maxVisible + 1
			}
		}
	case "d", "delete", "backspace":
		if b.mode == BookmarkModeLoad && len(b.bookmarks.Items) > 0 {
			name := b.bookmarks.Items[b.selected].Name
			b.bookmarks.Delete(name)
			_ = storage.SaveBookmarks(b.bookmarks)
			if b.selected >= len(b.bookmarks.Items) && b.selected > 0 {
				b.selected--
			}
			if len(b.bookmarks.Items) == 0 {
				b.Close()
				return BookmarkActionCancel, nil
			}
		}
	default:
		if b.mode == BookmarkModeSave {
			b.nameInput, _ = b.nameInput.Update(msg)
		}
	}
	return BookmarkActionNone, nil
}

func (b *BookmarkUI) View(width, height int) string {
	if b.mode == BookmarkModeNone {
		return ""
	}

	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	hintStyle := lipgloss.NewStyle().Foreground(ColorGray)

	var content string
	var title string

	if b.mode == BookmarkModeSave {
		title = "Save Bookmark"
		content = lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("Name:"),
			b.nameInput.View(),
			"",
			hintStyle.Render("Enter to save, Esc to cancel"),
		)
	} else {
		title = "Load Bookmark"
		if len(b.bookmarks.Items) == 0 {
			content = lipgloss.JoinVertical(lipgloss.Left,
				labelStyle.Render("No bookmarks saved"),
				"",
				hintStyle.Render("Esc to close"),
			)
		} else {
			var items []string
			end := b.scrollY + b.maxVisible
			if end > len(b.bookmarks.Items) {
				end = len(b.bookmarks.Items)
			}
			for i := b.scrollY; i < end; i++ {
				bm := b.bookmarks.Items[i]
				line := bm.Name
				style := lipgloss.NewStyle()
				if i == b.selected {
					style = style.Background(ColorBlue).Foreground(ColorOnAccent)
				}
				items = append(items, style.Render(" "+line+" "))
			}
			content = lipgloss.JoinVertical(lipgloss.Left,
				strings.Join(items, "\n"),
				"",
				hintStyle.Render("Enter load, d delete, Esc cancel"),
			)
		}
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2)

	box := boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		"",
		content,
	))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

type BookmarkAction int

const (
	BookmarkActionNone BookmarkAction = iota
	BookmarkActionSave
	BookmarkActionLoad
	BookmarkActionCancel
)
