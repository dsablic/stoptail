package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type ShardPicker struct {
	shards   []es.ShardInfo
	selected int
}

func NewShardPicker(shards []es.ShardInfo) *ShardPicker {
	return &ShardPicker{
		shards:   shards,
		selected: 0,
	}
}

func (p *ShardPicker) Next() {
	if p.selected < len(p.shards)-1 {
		p.selected++
	}
}

func (p *ShardPicker) Prev() {
	if p.selected > 0 {
		p.selected--
	}
}

func (p *ShardPicker) Selected() *es.ShardInfo {
	if p.selected >= 0 && p.selected < len(p.shards) {
		return &p.shards[p.selected]
	}
	return nil
}

func (p *ShardPicker) View() string {
	var parts []string

	for i, sh := range p.shards {
		label := sh.Shard
		if sh.Primary {
			label += "p"
		} else {
			label += "r"
		}

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
			Padding(0, 1)

		if i == p.selected {
			style = style.Bold(true).Underline(true)
		}

		parts = append(parts, style.Render(label))
	}

	content := strings.Join(parts, " ")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(0, 1)

	hint := lipgloss.NewStyle().Foreground(ColorGray).Render("  [←→ select, Enter confirm, Esc cancel]")

	return boxStyle.Render(content + hint)
}

func RenderShardInfoModal(sh *es.ShardInfo, ae *es.AllocationExplain, width, height int) string {
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
