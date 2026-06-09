package ui

import (
	"strconv"
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
	lines := m.renderShardBoxesWithHighlight(shards, width, false, false, 4)

	for i, line := range lines {
		visibleWidth := len([]rune(stripANSI(line)))
		if visibleWidth > width {
			t.Errorf("Line %d width %d exceeds max width %d: %q", i, visibleWidth, width, line)
		}
	}
}

func TestRenderShardBoxes_EmptyShards(t *testing.T) {
	m := OverviewModel{}
	lines := m.renderShardBoxesWithHighlight([]es.ShardInfo{}, 25, false, false, 4)

	if len(lines) != 1 {
		t.Errorf("Expected 1 line for empty shards, got %d", len(lines))
	}
}

func manyShards(n int) []es.ShardInfo {
	shards := make([]es.ShardInfo, n)
	for i := range shards {
		shards[i] = es.ShardInfo{Shard: strconv.Itoa(i), Primary: true, State: "STARTED"}
	}
	return shards
}

func TestRenderShardBoxes_CapShowsAllWhenRoomy(t *testing.T) {
	m := OverviewModel{}
	shards := manyShards(25)

	lines := m.renderShardBoxesWithHighlight(shards, 25, false, false, 10)

	if len(lines) > 10 {
		t.Fatalf("expected at most 10 lines, got %d", len(lines))
	}
	joined := stripANSI(strings.Join(lines, " "))
	if strings.Contains(joined, "+") {
		t.Errorf("did not expect overflow indicator when all shards fit: %q", joined)
	}
	for i := 0; i < 25; i++ {
		if !strings.Contains(joined, strconv.Itoa(i)) {
			t.Errorf("expected shard %d to be displayed, output: %q", i, joined)
		}
	}
}

func TestRenderShardBoxes_CapTruncatesWithIndicator(t *testing.T) {
	m := OverviewModel{}
	shards := manyShards(25)

	maxLines := 2
	lines := m.renderShardBoxesWithHighlight(shards, 25, false, false, maxLines)

	if len(lines) != maxLines {
		t.Fatalf("expected exactly %d lines, got %d", maxLines, len(lines))
	}
	last := stripANSI(lines[len(lines)-1])
	if !strings.Contains(last, "+") {
		t.Errorf("expected overflow indicator on last line, got %q", last)
	}
}

func TestNodeRowLayout_SingleNodeGetsAllHeight(t *testing.T) {
	m := OverviewModel{
		height:  50,
		cluster: &es.ClusterState{Nodes: []es.NodeInfo{{Name: "n1"}}},
	}

	linesPerNode, visible := m.nodeRowLayout()

	if visible != 1 {
		t.Errorf("expected 1 visible node, got %d", visible)
	}
	if linesPerNode < 10 {
		t.Errorf("expected a single node to receive ample lines, got %d", linesPerNode)
	}
}

func TestNodeRowLayout_ManyNodesShareHeight(t *testing.T) {
	nodes := make([]es.NodeInfo, 30)
	for i := range nodes {
		nodes[i] = es.NodeInfo{Name: "n" + strconv.Itoa(i)}
	}
	m := OverviewModel{height: 50, cluster: &es.ClusterState{Nodes: nodes}}

	linesPerNode, visible := m.nodeRowLayout()

	if linesPerNode < 2 {
		t.Errorf("expected at least 2 lines per node, got %d", linesPerNode)
	}
	if visible < 1 || visible > len(nodes) {
		t.Errorf("unexpected visible node count %d", visible)
	}
	if visible*linesPerNode > m.height {
		t.Errorf("node rows %d x %d lines exceed height %d", visible, linesPerNode, m.height)
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
