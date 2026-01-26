package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type ShardPicker struct {
	shards       []es.ShardInfo
	selected     int
	width        int
	height       int
	shardsPerRow int
	scrollY      int
}

func NewShardPicker(shards []es.ShardInfo, width, height int) *ShardPicker {
	p := &ShardPicker{
		shards:   shards,
		selected: 0,
	}
	p.SetSize(width, height)
	return p
}

func (p *ShardPicker) SetSize(width, height int) {
	p.width = width
	p.height = height
	shardWidth := 5
	maxPickerWidth := width - 6
	if maxPickerWidth < shardWidth {
		p.shardsPerRow = 4
		return
	}
	p.shardsPerRow = maxPickerWidth / shardWidth
	if p.shardsPerRow > 20 {
		p.shardsPerRow = 20
	}
}

func (p *ShardPicker) Right() {
	if p.selected < len(p.shards)-1 {
		p.selected++
	}
}

func (p *ShardPicker) Left() {
	if p.selected > 0 {
		p.selected--
	}
}

func (p *ShardPicker) Down() {
	next := p.selected + p.shardsPerRow
	if next < len(p.shards) {
		p.selected = next
	}
}

func (p *ShardPicker) Up() {
	next := p.selected - p.shardsPerRow
	if next >= 0 {
		p.selected = next
	}
}

func (p *ShardPicker) Selected() *es.ShardInfo {
	if p.selected >= 0 && p.selected < len(p.shards) {
		return &p.shards[p.selected]
	}
	return nil
}

func (p *ShardPicker) View() string {
	var rows []string
	var currentRow []string

	for i, sh := range p.shards {
		var bgColor lipgloss.Color
		switch sh.State {
		case "STARTED":
			if sh.Primary {
				bgColor = ColorGreen
			} else {
				bgColor = ColorBlue
			}
		case "RELOCATING", "INITIALIZING":
			bgColor = ColorYellow
		case "UNASSIGNED":
			bgColor = ColorRed
		default:
			bgColor = ColorGray
		}

		style := lipgloss.NewStyle().
			Background(bgColor).
			Foreground(ColorOnAccent).
			Width(4).
			Align(lipgloss.Center)

		if i == p.selected {
			style = style.Reverse(true).Bold(true)
		}

		currentRow = append(currentRow, style.Render(sh.Shard))

		if len(currentRow) >= p.shardsPerRow {
			rows = append(rows, strings.Join(currentRow, " "))
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, strings.Join(currentRow, " "))
	}

	pickerChrome := 8
	maxVisibleRows := p.height - pickerChrome
	if maxVisibleRows < 3 {
		maxVisibleRows = 3
	}

	selectedRow := p.selected / p.shardsPerRow
	if selectedRow < p.scrollY {
		p.scrollY = selectedRow
	}
	if selectedRow >= p.scrollY+maxVisibleRows {
		p.scrollY = selectedRow - maxVisibleRows + 1
	}

	visibleRows := rows
	if len(rows) > maxVisibleRows {
		endY := p.scrollY + maxVisibleRows
		if endY > len(rows) {
			endY = len(rows)
		}
		visibleRows = rows[p.scrollY:endY]
	}

	content := strings.Join(visibleRows, "\n")

	scrollIndicator := ""
	if len(rows) > maxVisibleRows {
		scrollIndicator = fmt.Sprintf(" [%d-%d of %d rows]", p.scrollY+1, p.scrollY+len(visibleRows), len(rows))
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(0, 1)

	hint := lipgloss.NewStyle().Foreground(ColorGray).Render("[arrows select, Enter confirm, Esc cancel]" + scrollIndicator)

	return boxStyle.Render(content + "\n" + hint)
}

func RenderShardInfoModal(sh *es.ShardInfo, ae *es.AllocationExplain, ri *es.RecoveryInfo, width, height int) string {
	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorWhite)

	shardType := "replica"
	if sh.Primary {
		shardType = "primary"
	}

	var lines []string
	lines = append(lines, labelStyle.Render("Index:    ")+valueStyle.Render(sh.Index))
	lines = append(lines, labelStyle.Render("Shard:    ")+valueStyle.Render(fmt.Sprintf("%s (%s)", sh.Shard, shardType)))
	lines = append(lines, labelStyle.Render("State:    ")+valueStyle.Render(sh.State))
	if sh.Node != "" {
		lines = append(lines, labelStyle.Render("Node:     ")+valueStyle.Render(sh.Node))
	}

	if ri != nil {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("--- Recovery Progress ---"))
		lines = append(lines, labelStyle.Render("Stage:    ")+valueStyle.Render(ri.Stage))
		lines = append(lines, labelStyle.Render("Type:     ")+valueStyle.Render(ri.Type))
		if ri.SourceNode != "" {
			lines = append(lines, labelStyle.Render("Source:   ")+valueStyle.Render(ri.SourceNode))
		}
		if ri.TargetNode != "" {
			lines = append(lines, labelStyle.Render("Target:   ")+valueStyle.Render(ri.TargetNode))
		}
		lines = append(lines, labelStyle.Render("Bytes:    ")+valueStyle.Render(ri.BytesPct))
		lines = append(lines, labelStyle.Render("Files:    ")+valueStyle.Render(ri.FilesPct))
		if ri.TranslogOps != "" && ri.TranslogOps != "0" {
			lines = append(lines, labelStyle.Render("Translog: ")+valueStyle.Render(ri.TranslogOps+" ops"))
		}
	}

	if ae != nil {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("--- Allocation Explain ---"))
		lines = append(lines, labelStyle.Render("Reason:   ")+valueStyle.Render(ae.UnassignedReason))
		lines = append(lines, labelStyle.Render("Status:   ")+valueStyle.Render(ae.AllocationStatus))
		if ae.ExplanationDetail != "" {
			lines = append(lines, "")
			lines = append(lines, labelStyle.Render("Details:"))
			lines = append(lines, valueStyle.Render(ae.ExplanationDetail))
		}
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(60)

	box := boxStyle.Render(content)
	footer := lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter or Esc to close")

	modal := lipgloss.JoinVertical(lipgloss.Center, box, footer)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
