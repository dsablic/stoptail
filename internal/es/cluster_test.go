package es

import (
	"encoding/json"
	"testing"
)

func TestParseIndicesResponse(t *testing.T) {
	raw := `[
		{"index":"test-1","health":"green","docs.count":"100","store.size":"1mb","pri":"1","rep":"1"},
		{"index":"test-2","health":"yellow","docs.count":"200","store.size":"2mb","pri":"2","rep":"0"}
	]`

	var indices []IndexInfo
	if err := json.Unmarshal([]byte(raw), &indices); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(indices) != 2 {
		t.Fatalf("got %d indices, want 2", len(indices))
	}

	if indices[0].Name != "test-1" {
		t.Errorf("indices[0].Name = %q, want %q", indices[0].Name, "test-1")
	}
	if indices[0].Health != "green" {
		t.Errorf("indices[0].Health = %q, want %q", indices[0].Health, "green")
	}
}

func TestParseNodesResponse(t *testing.T) {
	raw := `[
		{"name":"node-1","ip":"10.0.0.1","node.role":"dim"},
		{"name":"node-2","ip":"10.0.0.2","node.role":"m"}
	]`

	var nodes []NodeInfo
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	if nodes[0].Name != "node-1" {
		t.Errorf("nodes[0].Name = %q, want %q", nodes[0].Name, "node-1")
	}
}

func TestParseShardsResponse(t *testing.T) {
	raw := `[
		{"index":"test-1","shard":"0","prirep":"p","state":"STARTED","node":"node-1"},
		{"index":"test-1","shard":"0","prirep":"r","state":"STARTED","node":"node-2"}
	]`

	var shards []ShardInfo
	if err := json.Unmarshal([]byte(raw), &shards); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(shards) != 2 {
		t.Fatalf("got %d shards, want 2", len(shards))
	}

	if shards[0].Index != "test-1" {
		t.Errorf("shards[0].Index = %q, want %q", shards[0].Index, "test-1")
	}
	if shards[0].Primary != true {
		t.Error("shards[0] should be primary")
	}
	if shards[1].Primary != false {
		t.Error("shards[1] should be replica")
	}
}
