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

func TestParseNodeStatsResponse(t *testing.T) {
	raw := `[
		{
			"name": "node-1",
			"version": "8.12.0",
			"heap.percent": "45",
			"heap.current": "2.3gb",
			"heap.max": "5gb",
			"gc.young.count": "1234",
			"gc.young.time": "10s",
			"gc.old.count": "5",
			"gc.old.time": "1s",
			"fielddata.memory_size": "100mb",
			"query_cache.memory_size": "50mb",
			"segments.count": "500",
			"disk.used_percent": "60",
			"disk.avail": "400gb",
			"disk.total": "1tb",
			"disk.used": "600gb",
			"disk.indices": "500gb",
			"shards": "100",
			"pri": "50",
			"rep": "50"
		},
		{
			"name": "node-2",
			"version": "8.12.0",
			"heap.percent": "55",
			"heap.current": "2.8gb",
			"heap.max": "5gb",
			"gc.young.count": "2000",
			"gc.young.time": "15s",
			"gc.old.count": "10",
			"gc.old.time": "2s",
			"fielddata.memory_size": "200mb",
			"query_cache.memory_size": "75mb",
			"segments.count": "750",
			"disk.used_percent": "70",
			"disk.avail": "300gb",
			"disk.total": "1tb",
			"disk.used": "700gb",
			"disk.indices": "650gb",
			"shards": "120",
			"pri": "60",
			"rep": "60"
		}
	]`

	var nodes []NodeStats
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	if nodes[0].Name != "node-1" {
		t.Errorf("nodes[0].Name = %q, want %q", nodes[0].Name, "node-1")
	}
	if nodes[0].Version != "8.12.0" {
		t.Errorf("nodes[0].Version = %q, want %q", nodes[0].Version, "8.12.0")
	}
	if nodes[0].HeapPercent != "45" {
		t.Errorf("nodes[0].HeapPercent = %q, want %q", nodes[0].HeapPercent, "45")
	}
	if nodes[0].HeapCurrent != "2.3gb" {
		t.Errorf("nodes[0].HeapCurrent = %q, want %q", nodes[0].HeapCurrent, "2.3gb")
	}
	if nodes[0].HeapMax != "5gb" {
		t.Errorf("nodes[0].HeapMax = %q, want %q", nodes[0].HeapMax, "5gb")
	}
	if nodes[0].GCYoungCount != "1234" {
		t.Errorf("nodes[0].GCYoungCount = %q, want %q", nodes[0].GCYoungCount, "1234")
	}
	if nodes[0].GCYoungTime != "10s" {
		t.Errorf("nodes[0].GCYoungTime = %q, want %q", nodes[0].GCYoungTime, "10s")
	}
	if nodes[0].GCOldCount != "5" {
		t.Errorf("nodes[0].GCOldCount = %q, want %q", nodes[0].GCOldCount, "5")
	}
	if nodes[0].GCOldTime != "1s" {
		t.Errorf("nodes[0].GCOldTime = %q, want %q", nodes[0].GCOldTime, "1s")
	}
	if nodes[0].FielddataSize != "100mb" {
		t.Errorf("nodes[0].FielddataSize = %q, want %q", nodes[0].FielddataSize, "100mb")
	}
	if nodes[0].QueryCacheSize != "50mb" {
		t.Errorf("nodes[0].QueryCacheSize = %q, want %q", nodes[0].QueryCacheSize, "50mb")
	}
	if nodes[0].SegmentsCount != "500" {
		t.Errorf("nodes[0].SegmentsCount = %q, want %q", nodes[0].SegmentsCount, "500")
	}
	if nodes[0].DiskPercent != "60" {
		t.Errorf("nodes[0].DiskPercent = %q, want %q", nodes[0].DiskPercent, "60")
	}
	if nodes[0].DiskAvail != "400gb" {
		t.Errorf("nodes[0].DiskAvail = %q, want %q", nodes[0].DiskAvail, "400gb")
	}
	if nodes[0].DiskTotal != "1tb" {
		t.Errorf("nodes[0].DiskTotal = %q, want %q", nodes[0].DiskTotal, "1tb")
	}
	if nodes[0].DiskUsed != "600gb" {
		t.Errorf("nodes[0].DiskUsed = %q, want %q", nodes[0].DiskUsed, "600gb")
	}
	if nodes[0].DiskIndices != "500gb" {
		t.Errorf("nodes[0].DiskIndices = %q, want %q", nodes[0].DiskIndices, "500gb")
	}
	if nodes[0].Shards != "100" {
		t.Errorf("nodes[0].Shards = %q, want %q", nodes[0].Shards, "100")
	}
	if nodes[0].PrimaryShards != "50" {
		t.Errorf("nodes[0].PrimaryShards = %q, want %q", nodes[0].PrimaryShards, "50")
	}
	if nodes[0].ReplicaShards != "50" {
		t.Errorf("nodes[0].ReplicaShards = %q, want %q", nodes[0].ReplicaShards, "50")
	}

	if nodes[1].Name != "node-2" {
		t.Errorf("nodes[1].Name = %q, want %q", nodes[1].Name, "node-2")
	}
	if nodes[1].HeapPercent != "55" {
		t.Errorf("nodes[1].HeapPercent = %q, want %q", nodes[1].HeapPercent, "55")
	}
}

func TestParseFielddataInfoResponse(t *testing.T) {
	raw := `[
		{"field": "user.name", "size": "100mb"},
		{"field": "timestamp", "size": "50mb"},
		{"field": "message.keyword", "size": "25mb"}
	]`

	var fielddata []FielddataInfo
	if err := json.Unmarshal([]byte(raw), &fielddata); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(fielddata) != 3 {
		t.Fatalf("got %d fielddata entries, want 3", len(fielddata))
	}

	if fielddata[0].Field != "user.name" {
		t.Errorf("fielddata[0].Field = %q, want %q", fielddata[0].Field, "user.name")
	}
	if fielddata[0].Size != "100mb" {
		t.Errorf("fielddata[0].Size = %q, want %q", fielddata[0].Size, "100mb")
	}

	if fielddata[1].Field != "timestamp" {
		t.Errorf("fielddata[1].Field = %q, want %q", fielddata[1].Field, "timestamp")
	}
	if fielddata[1].Size != "50mb" {
		t.Errorf("fielddata[1].Size = %q, want %q", fielddata[1].Size, "50mb")
	}

	if fielddata[2].Field != "message.keyword" {
		t.Errorf("fielddata[2].Field = %q, want %q", fielddata[2].Field, "message.keyword")
	}
	if fielddata[2].Size != "25mb" {
		t.Errorf("fielddata[2].Size = %q, want %q", fielddata[2].Size, "25mb")
	}
}

func TestGetShardsForIndexAndNode(t *testing.T) {
	state := &ClusterState{
		Shards: []ShardInfo{
			{Index: "test", Shard: "2", PriRep: "r", State: "STARTED", Node: "node-1", Primary: false},
			{Index: "test", Shard: "0", PriRep: "p", State: "STARTED", Node: "node-1", Primary: true},
			{Index: "test", Shard: "1", PriRep: "r", State: "STARTED", Node: "node-1", Primary: false},
			{Index: "test", Shard: "1", PriRep: "p", State: "STARTED", Node: "node-1", Primary: true},
			{Index: "test", Shard: "0", PriRep: "r", State: "STARTED", Node: "node-2", Primary: false},
			{Index: "other", Shard: "0", PriRep: "p", State: "STARTED", Node: "node-1", Primary: true},
		},
	}

	shards := state.GetShardsForIndexAndNode("test", "node-1")

	if len(shards) != 4 {
		t.Fatalf("got %d shards, want 4", len(shards))
	}

	expected := []struct {
		shard   string
		primary bool
	}{
		{"0", true},
		{"1", true},
		{"1", false},
		{"2", false},
	}

	for i, exp := range expected {
		if shards[i].Shard != exp.shard {
			t.Errorf("shards[%d].Shard = %q, want %q", i, shards[i].Shard, exp.shard)
		}
		if shards[i].Primary != exp.primary {
			t.Errorf("shards[%d].Primary = %v, want %v", i, shards[i].Primary, exp.primary)
		}
	}
}

func TestGetUnassignedShardsForIndex(t *testing.T) {
	state := &ClusterState{
		Shards: []ShardInfo{
			{Index: "test", Shard: "0", PriRep: "p", State: "STARTED", Node: "node-1", Primary: true},
			{Index: "test", Shard: "0", PriRep: "r", State: "UNASSIGNED", Node: "", Primary: false},
			{Index: "test", Shard: "1", PriRep: "p", State: "STARTED", Node: "node-1", Primary: true},
			{Index: "test", Shard: "1", PriRep: "r", State: "UNASSIGNED", Node: "", Primary: false},
			{Index: "test", Shard: "2", PriRep: "r", State: "UNASSIGNED", Node: "", Primary: false},
			{Index: "other", Shard: "0", PriRep: "r", State: "UNASSIGNED", Node: "", Primary: false},
		},
	}

	shards := state.GetUnassignedShardsForIndex("test")

	if len(shards) != 3 {
		t.Fatalf("got %d shards, want 3", len(shards))
	}

	expectedShards := []string{"0", "1", "2"}
	for i, exp := range expectedShards {
		if shards[i].Shard != exp {
			t.Errorf("shards[%d].Shard = %q, want %q", i, shards[i].Shard, exp)
		}
		if shards[i].State != "UNASSIGNED" {
			t.Errorf("shards[%d].State = %q, want UNASSIGNED", i, shards[i].State)
		}
	}
}

func TestParseNodesStatsFielddataByIndexResponse(t *testing.T) {
	raw := `{
		"nodes": {
			"node-id-1": {
				"indices": {
					"indices": {
						"logs-2024-01": {
							"fielddata": {
								"memory_size_in_bytes": 1048576
							}
						},
						"logs-2024-02": {
							"fielddata": {
								"memory_size_in_bytes": 2097152
							}
						}
					}
				}
			},
			"node-id-2": {
				"indices": {
					"indices": {
						"logs-2024-01": {
							"fielddata": {
								"memory_size_in_bytes": 524288
							}
						},
						"logs-2024-03": {
							"fielddata": {
								"memory_size_in_bytes": 3145728
							}
						}
					}
				}
			}
		}
	}`

	var response struct {
		Nodes map[string]struct {
			Indices struct {
				Indices map[string]struct {
					Fielddata struct {
						MemorySizeInBytes int64 `json:"memory_size_in_bytes"`
					} `json:"fielddata"`
				} `json:"indices"`
			} `json:"indices"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(response.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(response.Nodes))
	}

	indexSizes := make(map[string]int64)
	for _, node := range response.Nodes {
		for indexName, indexData := range node.Indices.Indices {
			indexSizes[indexName] += indexData.Fielddata.MemorySizeInBytes
		}
	}

	if len(indexSizes) != 3 {
		t.Fatalf("got %d unique indices, want 3", len(indexSizes))
	}

	if indexSizes["logs-2024-01"] != 1572864 {
		t.Errorf("logs-2024-01 size = %d, want %d", indexSizes["logs-2024-01"], 1572864)
	}
	if indexSizes["logs-2024-02"] != 2097152 {
		t.Errorf("logs-2024-02 size = %d, want %d", indexSizes["logs-2024-02"], 2097152)
	}
	if indexSizes["logs-2024-03"] != 3145728 {
		t.Errorf("logs-2024-03 size = %d, want %d", indexSizes["logs-2024-03"], 3145728)
	}
}

func TestParseTasksResponse(t *testing.T) {
	raw := `{
		"nodes": {
			"node-id-1": {
				"name": "es-node-1",
				"tasks": {
					"node-id-1:12345": {
						"node": "node-id-1",
						"id": 12345,
						"type": "transport",
						"action": "indices:data/write/reindex",
						"description": "reindex from [source] to [dest]",
						"start_time_in_millis": 1700000000000,
						"running_time_in_nanos": 120000000000,
						"cancellable": true,
						"cancelled": false
					},
					"node-id-1:12346": {
						"node": "node-id-1",
						"id": 12346,
						"type": "transport",
						"action": "cluster:monitor/tasks/lists",
						"start_time_in_millis": 1700000001000,
						"running_time_in_nanos": 1000000,
						"cancellable": false
					}
				}
			}
		}
	}`

	tasks, err := parseTasksResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1 (only cancellable reindex)", len(tasks))
	}

	if tasks[0].ID != "node-id-1:12345" {
		t.Errorf("ID = %q, want %q", tasks[0].ID, "node-id-1:12345")
	}
	if tasks[0].Action != "indices:data/write/reindex" {
		t.Errorf("Action = %q, want %q", tasks[0].Action, "indices:data/write/reindex")
	}
	if tasks[0].Node != "es-node-1" {
		t.Errorf("Node = %q, want %q", tasks[0].Node, "es-node-1")
	}
	if tasks[0].RunningTime != "2m 0s" {
		t.Errorf("RunningTime = %q, want %q", tasks[0].RunningTime, "2m 0s")
	}
	if tasks[0].Cancellable != true {
		t.Error("Cancellable should be true")
	}
}
