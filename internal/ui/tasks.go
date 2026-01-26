package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/labtiva/stoptail/internal/es"
)

type TasksModel struct {
	tasks        []es.TaskInfo
	selectedRow  int
	scrollY      int
	width        int
	height       int
	loading      bool
	confirming   string
	search       SearchBar
	showingModal bool
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
		if m.showingModal {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				m.showingModal = false
			}
			return m, nil
		}

		if m.search.Active() {
			cmd, action := m.search.HandleKey(msg)
			switch action {
			case SearchActionClose:
				// search deactivated
			case SearchActionNext, SearchActionPrev:
				if match := m.search.CurrentMatch(); match >= 0 {
					m.selectedRow = match
					m.scrollY = max(0, match-5)
				}
			case SearchActionNone:
				(&m).updateTaskSearch()
			}
			return m, cmd
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
		case "enter":
			if m.selectedRow >= 0 && m.selectedRow < len(m.tasks) {
				m.showingModal = true
			}
			return m, nil
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
		case "pgup":
			pageSize := m.maxVisibleRows()
			m.selectedRow -= pageSize
			if m.selectedRow < 0 {
				m.selectedRow = 0
			}
			if m.selectedRow < m.scrollY {
				m.scrollY = m.selectedRow
			}
		case "pgdown":
			pageSize := m.maxVisibleRows()
			m.selectedRow += pageSize
			if m.selectedRow >= len(m.tasks) {
				m.selectedRow = len(m.tasks) - 1
			}
			maxVisible := m.maxVisibleRows()
			if m.selectedRow >= m.scrollY+maxVisible {
				m.scrollY = m.selectedRow - maxVisible + 1
			}
		case "c":
			if m.selectedRow >= 0 && m.selectedRow < len(m.tasks) && m.tasks[m.selectedRow].Cancellable {
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

	if m.showingModal {
		return m.renderDetailsModal()
	}

	maxVisible := m.maxVisibleRows()
	endIdx := min(m.scrollY+maxVisible, len(m.tasks))

	var rows [][]string
	var rowStates []string
	for i := m.scrollY; i < endIdx; i++ {
		task := m.tasks[i]
		cancelText := "-"
		if task.Cancellable {
			cancelText = "[c]"
		}
		state := "normal"
		if m.confirming == task.ID {
			cancelText = "y/n?"
			state = "confirming"
		} else if i == m.selectedRow {
			state = "selected"
		}
		rowStates = append(rowStates, state)

		rows = append(rows, []string{
			Truncate(m.truncateAction(task.Action), 25),
			Truncate(task.Node, 20),
			Truncate(task.Description, 30),
			task.RunningTime,
			cancelText,
		})
	}

	t := table.New().
		Headers("action", "node", "description", "running", "cancel").
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorGray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle()
			if col == 3 || col == 4 {
				base = base.Align(lipgloss.Right)
			}
			if row == table.HeaderRow {
				return base.Bold(true).Foreground(ColorWhite)
			}
			if row >= 0 && row < len(rowStates) {
				switch rowStates[row] {
				case "confirming":
					return base.Background(ColorRed).Foreground(ColorOnAccent)
				case "selected":
					return base.Background(ColorBlue).Foreground(ColorOnAccent)
				}
			}
			return base
		})

	content := t.Render()

	if m.confirming != "" {
		content += "\n\n" + lipgloss.NewStyle().Foreground(ColorYellow).Render("Cancel this task? Press 'y' to confirm, 'n' or Esc to abort")
	}

	if m.search.Active() {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.search.View(m.width-4))
	}

	return content
}


func (m TasksModel) truncateAction(action string) string {
	parts := strings.Split(action, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return action
}

func (m TasksModel) renderDetailsModal() string {
	if m.selectedRow < 0 || m.selectedRow >= len(m.tasks) {
		return ""
	}
	task := m.tasks[m.selectedRow]

	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	cancellableText := "No"
	if task.Cancellable {
		cancellableText = "Yes"
	}

	content := strings.Join([]string{
		labelStyle.Render("Task ID:     ") + valueStyle.Render(task.ID),
		labelStyle.Render("Action:      ") + valueStyle.Render(task.Action),
		labelStyle.Render("Node:        ") + valueStyle.Render(task.Node),
		labelStyle.Render("Running:     ") + valueStyle.Render(task.RunningTime),
		labelStyle.Render("Cancellable: ") + valueStyle.Render(cancellableText),
		"",
		labelStyle.Render("Description:"),
		valueStyle.Render(task.Description),
	}, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(70)

	box := boxStyle.Render(content)
	footer := lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter or Esc to close")

	modal := lipgloss.JoinVertical(lipgloss.Center, box, footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m TasksModel) HasModal() bool {
	return m.showingModal
}

