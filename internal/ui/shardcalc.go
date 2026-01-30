package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ShardCalcModel struct {
	sizeInput  textinput.Model
	docsInput  textinput.Model
	nodesInput textinput.Model
	focusIndex int
	width      int
	height     int
}

func NewShardCalc() ShardCalcModel {
	sizeInput := textinput.New()
	sizeInput.Placeholder = "e.g., 100gb, 1.5tb"
	sizeInput.CharLimit = 20
	sizeInput.Focus()

	docsInput := textinput.New()
	docsInput.Placeholder = "e.g., 50000000"
	docsInput.CharLimit = 20

	nodesInput := textinput.New()
	nodesInput.Placeholder = "optional"
	nodesInput.CharLimit = 5

	return ShardCalcModel{
		sizeInput:  sizeInput,
		docsInput:  docsInput,
		nodesInput: nodesInput,
		focusIndex: 0,
	}
}

func (m *ShardCalcModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *ShardCalcModel) Reset() {
	m.sizeInput.SetValue("")
	m.docsInput.SetValue("")
	m.nodesInput.SetValue("")
	m.focusIndex = 0
	m.sizeInput.Focus()
	m.docsInput.Blur()
	m.nodesInput.Blur()
}

func (m ShardCalcModel) Update(msg tea.Msg) (ShardCalcModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % 3
			m.updateFocus()
		case "shift+tab", "up":
			m.focusIndex = (m.focusIndex + 2) % 3
			m.updateFocus()
		}
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.sizeInput, cmd = m.sizeInput.Update(msg)
	case 1:
		m.docsInput, cmd = m.docsInput.Update(msg)
	case 2:
		m.nodesInput, cmd = m.nodesInput.Update(msg)
	}

	return m, cmd
}

func (m *ShardCalcModel) updateFocus() {
	m.sizeInput.Blur()
	m.docsInput.Blur()
	m.nodesInput.Blur()

	switch m.focusIndex {
	case 0:
		m.sizeInput.Focus()
	case 1:
		m.docsInput.Focus()
	case 2:
		m.nodesInput.Focus()
	}
}

func (m ShardCalcModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	labelStyle := lipgloss.NewStyle().Width(12)
	inputStyle := lipgloss.NewStyle().Width(25)
	resultLabelStyle := lipgloss.NewStyle().Width(20).Foreground(ColorGray)
	resultValueStyle := lipgloss.NewStyle().Bold(true)

	var b strings.Builder

	b.WriteString(titleStyle.Render("Shard Calculator"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Total Size:"))
	b.WriteString(inputStyle.Render(m.sizeInput.View()))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Total Docs:"))
	b.WriteString(inputStyle.Render(m.docsInput.View()))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Nodes:"))
	b.WriteString(inputStyle.Render(m.nodesInput.View()))
	b.WriteString("\n\n")

	result := m.calculate()
	resultLines := 0
	if result.err != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorRed).Render(result.err))
		resultLines = 1
	} else if result.totalSize > 0 {
		summaryParts := []string{formatSize(result.totalSize)}
		if result.totalDocs > 0 {
			summaryParts = append(summaryParts, formatDocs(result.totalDocs)+" docs")
		}
		if result.nodes > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d nodes", result.nodes))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render("For: " + strings.Join(summaryParts, " / ")))
		b.WriteString("\n\n")
		resultLines += 2

		b.WriteString(resultLabelStyle.Render("Primary shards:"))
		b.WriteString(resultValueStyle.Render(fmt.Sprintf("%d", result.primaryShards)))
		b.WriteString("\n")
		resultLines++

		b.WriteString(resultLabelStyle.Render("Shard size:"))
		b.WriteString(resultValueStyle.Render(formatSize(result.shardSize)))
		b.WriteString("\n")
		resultLines++

		if result.totalDocs > 0 {
			b.WriteString(resultLabelStyle.Render("Docs per shard:"))
			b.WriteString(resultValueStyle.Render(formatDocs(result.docsPerShard)))
			b.WriteString("\n")
			resultLines++
		}

		if result.nodes > 0 {
			b.WriteString(resultLabelStyle.Render("Shards per node:"))
			shardsPerNode := float64(result.primaryShards) / float64(result.nodes)
			if result.primaryShards%result.nodes == 0 {
				b.WriteString(resultValueStyle.Render(fmt.Sprintf("%d", result.primaryShards/result.nodes)))
			} else {
				b.WriteString(resultValueStyle.Render(fmt.Sprintf("%.1f", shardsPerNode)))
			}
			b.WriteString("\n")
			resultLines++
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render("Tab: next field | Esc: close"))

	modalWidth := 45
	modalHeight := 12 + resultLines

	content := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(modalWidth).
		Render(b.String())

	return OverlayModal("", content, m.width, modalHeight)
}

type calcResult struct {
	totalSize     int64
	totalDocs     int64
	nodes         int
	primaryShards int
	shardSize     int64
	docsPerShard  int64
	err           string
}

func (m ShardCalcModel) calculate() calcResult {
	var result calcResult

	sizeStr := strings.TrimSpace(m.sizeInput.Value())
	if sizeStr == "" {
		return result
	}

	totalSize := parseSize(sizeStr)
	if totalSize <= 0 {
		result.err = "Invalid size format"
		return result
	}
	result.totalSize = totalSize

	docsStr := strings.TrimSpace(m.docsInput.Value())
	if docsStr != "" {
		docs, err := strconv.ParseInt(strings.ReplaceAll(docsStr, ",", ""), 10, 64)
		if err != nil || docs < 0 {
			result.err = "Invalid docs format"
			return result
		}
		result.totalDocs = docs
	}

	nodesStr := strings.TrimSpace(m.nodesInput.Value())
	if nodesStr != "" {
		nodes, err := strconv.Atoi(nodesStr)
		if err != nil || nodes < 1 {
			result.err = "Invalid nodes"
			return result
		}
		result.nodes = nodes
	}

	const targetShardSize int64 = 30 * 1024 * 1024 * 1024

	primaryShards := int(totalSize / targetShardSize)
	if primaryShards < 1 {
		primaryShards = 1
	}

	if result.nodes > 0 && primaryShards < result.nodes {
		primaryShards = result.nodes
	}

	result.primaryShards = primaryShards
	result.shardSize = totalSize / int64(primaryShards)

	if result.totalDocs > 0 {
		result.docsPerShard = result.totalDocs / int64(primaryShards)
	}

	return result
}

func parseSize(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0
	}

	var multiplier int64 = 1
	var numStr string

	switch {
	case strings.HasSuffix(s, "tb"):
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "tb")
	case strings.HasSuffix(s, "gb"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "mb"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "t"):
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "t")
	case strings.HasSuffix(s, "g"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "m"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "m")
	default:
		numStr = s
	}

	numStr = strings.TrimSpace(numStr)
	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	return int64(value * float64(multiplier))
}

func formatSize(bytes int64) string {
	const (
		gb = 1024 * 1024 * 1024
		mb = 1024 * 1024
	)

	if bytes >= gb {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
}

func formatDocs(docs int64) string {
	if docs >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(docs)/1000000)
	}
	if docs >= 1000 {
		return fmt.Sprintf("%.1fK", float64(docs)/1000)
	}
	return fmt.Sprintf("%d", docs)
}
