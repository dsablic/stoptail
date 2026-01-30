package ui

import (
	"strings"
	"testing"

	"github.com/labtiva/stoptail/internal/es"
)

func TestRenderShardBoxes_ThreeDigitShards(t *testing.T) {
	m := OverviewModel{}

	shards := []es.ShardInfo{
		{Shard: "0", Primary: true, State: "STARTED"},
		{Shard: "100", Primary: true, State: "STARTED"},
		{Shard: "999", Primary: true, State: "STARTED"},
		{Shard: "1", Primary: false, State: "STARTED"},
	}

	width := 25
	lines := m.renderShardBoxesWithHighlight(shards, width, false, false)

	for i, line := range lines {
		visibleWidth := len([]rune(stripANSI(line)))
		if visibleWidth > width {
			t.Errorf("Line %d width %d exceeds max width %d: %q", i, visibleWidth, width, line)
		}
	}
}

func TestRenderShardBoxes_EmptyShards(t *testing.T) {
	m := OverviewModel{}
	lines := m.renderShardBoxesWithHighlight([]es.ShardInfo{}, 25, false, false)

	if len(lines) != 1 {
		t.Errorf("Expected 1 line for empty shards, got %d", len(lines))
	}
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}
