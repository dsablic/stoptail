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
			"fielddata.memory_size": "100mb",
			"query_cache.memory_size": "50mb",
			"segments.count": "500",
			"disk.used_percent": "60",
			"disk.avail": "400gb",
			"disk.total": "1tb",
			"disk.used": "600gb",
			"shard_stats.total_count": "100"
		},
		{
			"name": "node-2",
			"version": "8.12.0",
			"heap.percent": "55",
			"heap.current": "2.8gb",
			"heap.max": "5gb",
			"fielddata.memory_size": "200mb",
			"query_cache.memory_size": "75mb",
			"segments.count": "750",
			"disk.used_percent": "70",
			"disk.avail": "300gb",
			"disk.total": "1tb",
			"disk.used": "700gb",
			"shard_stats.total_count": "120"
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
	if nodes[0].Shards != "100" {
		t.Errorf("nodes[0].Shards = %q, want %q", nodes[0].Shards, "100")
	}

	if nodes[1].Name != "node-2" {
		t.Errorf("nodes[1].Name = %q, want %q", nodes[1].Name, "node-2")
	}
	if nodes[1].HeapPercent != "55" {
		t.Errorf("nodes[1].HeapPercent = %q, want %q", nodes[1].HeapPercent, "55")
	}
}

func TestParseFielddataNodesStatsResponse(t *testing.T) {
	raw := `{
		"nodes": {
			"node-id-1": {
				"name": "es-node-1",
				"indices": {
					"indices": {
						"products": {
							"fielddata": {
								"memory_size_in_bytes": 104857600,
								"fields": {
									"category": {"memory_size_in_bytes": 52428800},
									"brand": {"memory_size_in_bytes": 52428800}
								}
							}
						},
						"orders": {
							"fielddata": {
								"memory_size_in_bytes": 26214400,
								"fields": {
									"status": {"memory_size_in_bytes": 26214400}
								}
							}
						}
					}
				}
			}
		}
	}`

	var response struct {
		Nodes map[string]struct {
			Name    string `json:"name"`
			Indices struct {
				Indices map[string]struct {
					Fielddata struct {
						MemorySizeInBytes int64 `json:"memory_size_in_bytes"`
						Fields            map[string]struct {
							MemorySizeInBytes int64 `json:"memory_size_in_bytes"`
						} `json:"fields"`
					} `json:"fielddata"`
				} `json:"indices"`
			} `json:"indices"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(response.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(response.Nodes))
	}

	node := response.Nodes["node-id-1"]
	if node.Name != "es-node-1" {
		t.Errorf("node.Name = %q, want %q", node.Name, "es-node-1")
	}

	if len(node.Indices.Indices) != 2 {
		t.Fatalf("got %d indices, want 2", len(node.Indices.Indices))
	}

	products := node.Indices.Indices["products"]
	if products.Fielddata.MemorySizeInBytes != 104857600 {
		t.Errorf("products fielddata = %d, want %d", products.Fielddata.MemorySizeInBytes, 104857600)
	}
	if len(products.Fielddata.Fields) != 2 {
		t.Errorf("products fields = %d, want 2", len(products.Fielddata.Fields))
	}
	if products.Fielddata.Fields["category"].MemorySizeInBytes != 52428800 {
		t.Errorf("category fielddata = %d, want %d", products.Fielddata.Fields["category"].MemorySizeInBytes, 52428800)
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
					},
					"node-id-1:12347": {
						"node": "node-id-1",
						"id": 12347,
						"type": "transport",
						"action": "indices:admin/forcemerge",
						"description": "Force-merge indices [test]",
						"start_time_in_millis": 1700000002000,
						"running_time_in_nanos": 60000000000,
						"cancellable": false
					},
					"node-id-1:12348": {
						"node": "node-id-1",
						"id": 12348,
						"type": "transport",
						"action": "indices:admin/forcemerge[n]",
						"description": "Force-merge indices [test]",
						"start_time_in_millis": 1700000002000,
						"running_time_in_nanos": 60000000000,
						"cancellable": false,
						"parent_task_id": "node-id-1:12347"
					}
				}
			}
		}
	}`

	tasks, err := parseTasksResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2 (reindex + forcemerge parent, excluding child)", len(tasks))
	}

	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}

	if !taskIDs["node-id-1:12345"] {
		t.Error("expected reindex task to be included")
	}
	if !taskIDs["node-id-1:12347"] {
		t.Error("expected forcemerge parent task to be included")
	}
	if taskIDs["node-id-1:12348"] {
		t.Error("expected forcemerge child task (with parent_task_id) to be excluded")
	}
}

func TestParseIndexSettings(t *testing.T) {
	raw := `{"products":{"settings":{"index.creation_date":"1769163610124","index.number_of_replicas":"1","index.number_of_shards":"2","index.provided_name":"products","index.routing.allocation.include._tier_preference":"data_content","index.uuid":"0iA3VMGtS32YuFMvIPgeWw","index.version.created":"9039003"}}}`

	settings, err := parseIndexSettings("products", []byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if settings.IndexName != "products" {
		t.Errorf("IndexName = %q, want %q", settings.IndexName, "products")
	}
	if settings.NumberOfShards != "2" {
		t.Errorf("NumberOfShards = %q, want %q", settings.NumberOfShards, "2")
	}
	if settings.NumberOfReplicas != "1" {
		t.Errorf("NumberOfReplicas = %q, want %q", settings.NumberOfReplicas, "1")
	}
	if settings.CreationDate != "1769163610124" {
		t.Errorf("CreationDate = %q, want %q", settings.CreationDate, "1769163610124")
	}
	if settings.UUID != "0iA3VMGtS32YuFMvIPgeWw" {
		t.Errorf("UUID = %q, want %q", settings.UUID, "0iA3VMGtS32YuFMvIPgeWw")
	}
	if settings.Version != "9039003" {
		t.Errorf("Version = %q, want %q", settings.Version, "9039003")
	}
	if len(settings.AllSettings) != 7 {
		t.Errorf("AllSettings has %d entries, want 7", len(settings.AllSettings))
	}
}

func TestParseIndexSettingsWithArrays(t *testing.T) {
	raw := `{"patents":{"settings":{"index.analysis.analyzer.custom_english.filter":["lowercase","english_stemmer"],"index.number_of_shards":"5","index.number_of_replicas":"0"}}}`

	settings, err := parseIndexSettings("patents", []byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if settings.NumberOfShards != "5" {
		t.Errorf("NumberOfShards = %q, want %q", settings.NumberOfShards, "5")
	}
	if settings.NumberOfReplicas != "0" {
		t.Errorf("NumberOfReplicas = %q, want %q", settings.NumberOfReplicas, "0")
	}

	filterVal := settings.AllSettings["index.analysis.analyzer.custom_english.filter"]
	expected := "[lowercase, english_stemmer]"
	if filterVal != expected {
		t.Errorf("filter = %q, want %q", filterVal, expected)
	}
}

func TestDecodeESVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"8110099", "8.x"},
		{"7170099", "7.x"},
		{"9039003", "9.x"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tt := range tests {
		got := DecodeESVersion(tt.input)
		if got != tt.expected {
			t.Errorf("DecodeESVersion(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseMappingResponse(t *testing.T) {
	raw := `{
		"products": {
			"mappings": {
				"properties": {
					"title": {"type": "text"},
					"price": {"type": "float"},
					"category": {
						"type": "object",
						"properties": {
							"name": {"type": "keyword"},
							"id": {"type": "integer"}
						}
					},
					"tags": {"type": "keyword"}
				}
			}
		}
	}`

	fields, err := parseMappingResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	expected := []string{"title", "price", "category.name", "category.id", "tags"}
	if len(fields) != len(expected) {
		t.Fatalf("got %d fields, want %d: %v", len(fields), len(expected), fields)
	}

	fieldMap := make(map[string]bool)
	for _, f := range fields {
		fieldMap[f] = true
	}

	for _, e := range expected {
		if !fieldMap[e] {
			t.Errorf("missing field %q", e)
		}
	}
}

func TestAnalyzeShardHealth(t *testing.T) {
	tests := []struct {
		name       string
		idx        IndexInfo
		wantStatus ShardHealthStatus
		wantText   string
	}{
		{
			name: "healthy index",
			idx: IndexInfo{
				Name:         "test",
				Pri:          "3",
				PriStoreSize: "45gb",
				DocsCount:    "15000000",
			},
			wantStatus: ShardHealthOK,
			wantText:   "ok",
		},
		{
			name: "oversized shards",
			idx: IndexInfo{
				Name:         "test",
				Pri:          "2",
				PriStoreSize: "140gb",
				DocsCount:    "50000000",
			},
			wantStatus: ShardHealthCritical,
			wantText:   "oversized",
		},
		{
			name: "undersized shards",
			idx: IndexInfo{
				Name:         "test",
				Pri:          "10",
				PriStoreSize: "1gb",
				DocsCount:    "5000000",
			},
			wantStatus: ShardHealthCritical,
			wantText:   "undersized",
		},
		{
			name: "sparse shards",
			idx: IndexInfo{
				Name:         "test",
				Pri:          "10",
				PriStoreSize: "50gb",
				DocsCount:    "500000",
			},
			wantStatus: ShardHealthWarning,
			wantText:   "sparse",
		},
		{
			name: "small index ok",
			idx: IndexInfo{
				Name:         "test",
				Pri:          "1",
				PriStoreSize: "100mb",
				DocsCount:    "1000",
			},
			wantStatus: ShardHealthOK,
			wantText:   "ok",
		},
		{
			name: "multiple issues undersized and sparse",
			idx: IndexInfo{
				Name:         "test",
				Pri:          "10",
				PriStoreSize: "1gb",
				DocsCount:    "500000",
			},
			wantStatus: ShardHealthCritical,
			wantText:   "undersized, sparse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := AnalyzeShardHealth(tt.idx)
			if health.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", health.Status, tt.wantStatus)
			}
			if health.StatusText != tt.wantText {
				t.Errorf("StatusText = %q, want %q", health.StatusText, tt.wantText)
			}
		})
	}
}

