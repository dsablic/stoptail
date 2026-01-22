package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type TasksModel struct {
	tasks       []es.TaskInfo
	selectedRow int
	scrollY     int
	width       int
	height      int
	loading     bool
	confirming  string
	search      SearchBar
}

func NewTasks() TasksModel {
	return TasksModel{
		loading: true,
		search:  NewSearchBar(),
	}
}

func (m *TasksModel) SetTasks(tasks []es.TaskInfo) {
	m.tasks = tasks
	m.loading = false
	if m.selectedRow >= len(tasks) {
		m.selectedRow = max(0, len(tasks)-1)
	}
}

func (m *TasksModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m TasksModel) SelectedTaskID() string {
	if m.selectedRow >= 0 && m.selectedRow < len(m.tasks) {
		return m.tasks[m.selectedRow].ID
	}
	return ""
}

func (m *TasksModel) ClearConfirming() {
	m.confirming = ""
}

func (m TasksModel) Update(msg tea.Msg) (TasksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.search.Active() {
			switch msg.String() {
			case "esc":
				m.search.Deactivate()
				return m, nil
			case "enter":
				if match := m.search.NextMatch(); match >= 0 {
					m.selectedRow = match
					m.scrollY = max(0, match-5)
				}
				return m, nil
			case "shift+enter":
				if match := m.search.PrevMatch(); match >= 0 {
					m.selectedRow = match
					m.scrollY = max(0, match-5)
				}
				return m, nil
			default:
				cmd := m.search.Update(msg)
				(&m).updateTaskSearch()
				return m, cmd
			}
		}

		if m.confirming != "" {
			switch msg.String() {
			case "y", "Y":
				taskID := m.confirming
				m.confirming = ""
				return m, func() tea.Msg {
					return taskCancelRequestMsg{taskID: taskID}
				}
			case "n", "N", "esc":
				m.confirming = ""
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.selectedRow > 0 {
				m.selectedRow--
				if m.selectedRow < m.scrollY {
					m.scrollY = m.selectedRow
				}
			}
		case "down", "j":
			if m.selectedRow < len(m.tasks)-1 {
				m.selectedRow++
				maxVisible := m.maxVisibleRows()
				if m.selectedRow >= m.scrollY+maxVisible {
					m.scrollY = m.selectedRow - maxVisible + 1
				}
			}
		case "c":
			if m.selectedRow >= 0 && m.selectedRow < len(m.tasks) {
				m.confirming = m.tasks[m.selectedRow].ID
			}
		case "ctrl+f":
			if m.confirming == "" {
				m.search.Activate()
				return m, nil
			}
		}
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			scrollAmount := 3
			maxScroll := m.maxScroll()
			if msg.Button == tea.MouseButtonWheelUp {
				m.scrollY = max(0, m.scrollY-scrollAmount)
			} else {
				m.scrollY = min(maxScroll, m.scrollY+scrollAmount)
			}
		}
	}
	return m, nil
}

func (m TasksModel) maxVisibleRows() int {
	rows := m.height - 6
	if rows < 1 {
		return 10
	}
	return rows
}

func (m TasksModel) maxScroll() int {
	maxScroll := len(m.tasks) - m.maxVisibleRows()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m *TasksModel) updateTaskSearch() {
	var lines []string
	for _, task := range m.tasks {
		lines = append(lines, task.Action+" "+task.Description+" "+task.Index+" "+task.Node)
	}
	m.search.FindMatches(lines)
	if match := m.search.CurrentMatch(); match >= 0 {
		m.selectedRow = match
		m.scrollY = max(0, match-5)
	}
}

type taskCancelRequestMsg struct {
	taskID string
}

func (m TasksModel) View() string {
	if m.loading {
		return "Loading tasks..."
	}

	if len(m.tasks) == 0 {
		return lipgloss.NewStyle().
			Foreground(ColorGray).
			Padding(2).
			Render("No long-running tasks found.\n\nThis view shows: reindex, update-by-query, delete-by-query,\nforce merge, and snapshot operations.")
	}

	var b strings.Builder

	colWidths := []int{35, 25, 15, 12, 8}
	headers := []string{"action", "node", "description", "running", "cancel"}
	b.WriteString(m.renderHeader(headers, colWidths))

	maxVisible := m.maxVisibleRows()
	endIdx := min(m.scrollY+maxVisible, len(m.tasks))

	for i := m.scrollY; i < endIdx; i++ {
		task := m.tasks[i]
		isSelected := i == m.selectedRow
		isConfirming := m.confirming == task.ID

		rowStyle := lipgloss.NewStyle()
		if isConfirming {
			rowStyle = rowStyle.Background(ColorRed).Foreground(ColorOnAccent)
		} else if isSelected {
			rowStyle = rowStyle.Background(ColorBlue).Foreground(ColorOnAccent)
		}

		action := m.truncateAction(task.Action)
		desc := task.Description
		if len(desc) > colWidths[2]-2 {
			desc = desc[:colWidths[2]-5] + "..."
		}

		cancelText := "[c]"
		if isConfirming {
			cancelText = "y/n?"
		}

		row := fmt.Sprintf("%-*s %-*s %-*s %*s %*s",
			colWidths[0], action,
			colWidths[1], m.truncate(task.Node, colWidths[1]),
			colWidths[2], m.truncate(desc, colWidths[2]),
			colWidths[3], task.RunningTime,
			colWidths[4], cancelText,
		)

		b.WriteString(rowStyle.Render(row))
		b.WriteString("\n")
	}

	if m.confirming != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(ColorYellow).Render("Cancel this task? Press 'y' to confirm, 'n' or Esc to abort"))
	}

	content := b.String()

	if m.search.Active() {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.search.View(m.width-4))
	}

	return content
}

func (m TasksModel) renderHeader(headers []string, widths []int) string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	var parts []string
	for i, h := range headers {
		parts = append(parts, fmt.Sprintf("%-*s", widths[i], h))
	}
	header := headerStyle.Render(strings.Join(parts, " "))

	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}
	totalWidth += len(widths) - 1

	return header + "\n" + strings.Repeat("-", totalWidth) + "\n"
}

func (m TasksModel) truncateAction(action string) string {
	parts := strings.Split(action, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return action
}

func (m TasksModel) truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-3]) + "..."
}
