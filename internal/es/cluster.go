package es

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

type IndexInfo struct {
	Name      string `json:"index"`
	Health    string `json:"health"`
	DocsCount string `json:"docs.count"`
	StoreSize string `json:"store.size"`
	Pri       string `json:"pri"`
	Rep       string `json:"rep"`
}

type NodeInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Role string `json:"node.role"`
}

type ShardInfo struct {
	Index   string `json:"index"`
	Shard   string `json:"shard"`
	PriRep  string `json:"prirep"`
	State   string `json:"state"`
	Node    string `json:"node"`
	Primary bool   `json:"-"`
}

func (s *ShardInfo) UnmarshalJSON(data []byte) error {
	type Alias ShardInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Primary = s.PriRep == "p"
	return nil
}

type AliasInfo struct {
	Alias string `json:"alias"`
	Index string `json:"index"`
}

type NodeStats struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	HeapPercent    string `json:"heap.percent"`
	HeapCurrent    string `json:"heap.current"`
	HeapMax        string `json:"heap.max"`
	GCYoungCount   string `json:"gc.young.count"`
	GCYoungTime    string `json:"gc.young.time"`
	GCOldCount     string `json:"gc.old.count"`
	GCOldTime      string `json:"gc.old.time"`
	FielddataSize  string `json:"fielddata.memory_size"`
	QueryCacheSize string `json:"query_cache.memory_size"`
	SegmentsCount  string `json:"segments.count"`
	DiskPercent    string `json:"disk.used_percent"`
	DiskAvail      string `json:"disk.avail"`
	DiskTotal      string `json:"disk.total"`
	DiskUsed       string `json:"disk.used"`
	DiskIndices    string `json:"disk.indices"`
	Shards         string `json:"shards"`
	PrimaryShards  string `json:"pri"`
	ReplicaShards  string `json:"rep"`
}

type FielddataInfo struct {
	Field string `json:"field"`
	Size  string `json:"size"`
}

type FielddataByIndex struct {
	Index string
	Size  int64
}

type NodesState struct {
	Nodes            []NodeStats
	FielddataByField []FielddataInfo
	FielddataByIndex []FielddataByIndex
}

type TaskInfo struct {
	ID            string
	Action        string
	Node          string
	Index         string
	RunningTime   string
	RunningTimeMs int64
	Description   string
	Cancellable   bool
}

type ClusterState struct {
	Indices []IndexInfo
	Nodes   []NodeInfo
	Shards  []ShardInfo
	Aliases []AliasInfo
}

func (c *Client) FetchClusterState(ctx context.Context) (*ClusterState, error) {
	state := &ClusterState{}

	indices, err := c.fetchIndices(ctx)
	if err != nil {
		return nil, err
	}
	state.Indices = indices

	nodes, err := c.fetchNodes(ctx)
	if err != nil {
		return nil, err
	}
	state.Nodes = nodes

	shards, err := c.fetchShards(ctx)
	if err != nil {
		return nil, err
	}
	state.Shards = shards

	aliases, err := c.fetchAliases(ctx)
	if err != nil {
		return nil, err
	}
	state.Aliases = aliases

	return state, nil
}

func (c *Client) fetchIndices(ctx context.Context) ([]IndexInfo, error) {
	res, err := c.es.Cat.Indices(
		c.es.Cat.Indices.WithContext(ctx),
		c.es.Cat.Indices.WithFormat("json"),
		c.es.Cat.Indices.WithH("index", "health", "docs.count", "store.size", "pri", "rep"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching indices: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading indices response: %w", err)
	}

	var indices []IndexInfo
	if err := json.Unmarshal(body, &indices); err != nil {
		return nil, fmt.Errorf("parsing indices: %w", err)
	}

	sort.Slice(indices, func(i, j int) bool {
		return indices[i].Name < indices[j].Name
	})

	return indices, nil
}

func (c *Client) fetchNodes(ctx context.Context) ([]NodeInfo, error) {
	res, err := c.es.Cat.Nodes(
		c.es.Cat.Nodes.WithContext(ctx),
		c.es.Cat.Nodes.WithFormat("json"),
		c.es.Cat.Nodes.WithH("name", "ip", "node.role"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching nodes: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading nodes response: %w", err)
	}

	var nodes []NodeInfo
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("parsing nodes: %w", err)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

func (c *Client) fetchShards(ctx context.Context) ([]ShardInfo, error) {
	res, err := c.es.Cat.Shards(
		c.es.Cat.Shards.WithContext(ctx),
		c.es.Cat.Shards.WithFormat("json"),
		c.es.Cat.Shards.WithH("index", "shard", "prirep", "state", "node"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching shards: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading shards response: %w", err)
	}

	var shards []ShardInfo
	if err := json.Unmarshal(body, &shards); err != nil {
		return nil, fmt.Errorf("parsing shards: %w", err)
	}

	sort.Slice(shards, func(i, j int) bool {
		if shards[i].Index != shards[j].Index {
			return shards[i].Index < shards[j].Index
		}
		if shards[i].Shard != shards[j].Shard {
			return shards[i].Shard < shards[j].Shard
		}
		return shards[i].Primary && !shards[j].Primary
	})

	return shards, nil
}

func (c *Client) fetchAliases(ctx context.Context) ([]AliasInfo, error) {
	res, err := c.es.Cat.Aliases(
		c.es.Cat.Aliases.WithContext(ctx),
		c.es.Cat.Aliases.WithFormat("json"),
		c.es.Cat.Aliases.WithH("alias", "index"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching aliases: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading aliases response: %w", err)
	}

	var aliases []AliasInfo
	if err := json.Unmarshal(body, &aliases); err != nil {
		return nil, fmt.Errorf("parsing aliases: %w", err)
	}

	sort.Slice(aliases, func(i, j int) bool {
		if aliases[i].Alias != aliases[j].Alias {
			return aliases[i].Alias < aliases[j].Alias
		}
		return aliases[i].Index < aliases[j].Index
	})

	return aliases, nil
}

func (s *ClusterState) GetAliasesForIndex(index string) []string {
	var aliases []string
	for _, a := range s.Aliases {
		if a.Index == index {
			aliases = append(aliases, a.Alias)
		}
	}
	return aliases
}

func (s *ClusterState) GetShardsForIndexAndNode(index, node string) []ShardInfo {
	var shards []ShardInfo
	for _, sh := range s.Shards {
		if sh.Index == index && sh.Node == node {
			shards = append(shards, sh)
		}
	}
	sort.Slice(shards, func(i, j int) bool {
		if shards[i].Shard != shards[j].Shard {
			return shards[i].Shard < shards[j].Shard
		}
		return shards[i].Primary && !shards[j].Primary
	})
	return shards
}

func (s *ClusterState) GetUnassignedShardsForIndex(index string) []ShardInfo {
	var shards []ShardInfo
	for _, sh := range s.Shards {
		if sh.Index == index && (sh.Node == "" || sh.State == "UNASSIGNED") {
			shards = append(shards, sh)
		}
	}
	sort.Slice(shards, func(i, j int) bool {
		if shards[i].Shard != shards[j].Shard {
			return shards[i].Shard < shards[j].Shard
		}
		return shards[i].Primary && !shards[j].Primary
	})
	return shards
}

func (s *ClusterState) UniqueAliases() []string {
	seen := make(map[string]bool)
	var aliases []string
	for _, a := range s.Aliases {
		if !seen[a.Alias] {
			seen[a.Alias] = true
			aliases = append(aliases, a.Alias)
		}
	}
	return aliases
}

func (c *Client) FetchNodeStats(ctx context.Context) ([]NodeStats, error) {
	res, err := c.es.Cat.Nodes(
		c.es.Cat.Nodes.WithContext(ctx),
		c.es.Cat.Nodes.WithFormat("json"),
		c.es.Cat.Nodes.WithH(
			"name", "version", "heap.percent", "heap.current", "heap.max",
			"gc.young.count", "gc.young.time", "gc.old.count", "gc.old.time",
			"fielddata.memory_size", "query_cache.memory_size", "segments.count",
			"disk.used_percent", "disk.avail", "disk.total", "disk.used", "disk.indices",
			"shards", "pri", "rep",
		),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching node stats: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading node stats response: %w", err)
	}

	var nodes []NodeStats
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("parsing node stats: %w", err)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

func (c *Client) FetchFielddataByField(ctx context.Context) ([]FielddataInfo, error) {
	res, err := c.es.Cat.Fielddata(
		c.es.Cat.Fielddata.WithContext(ctx),
		c.es.Cat.Fielddata.WithFormat("json"),
		c.es.Cat.Fielddata.WithFields("*"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching fielddata: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading fielddata response: %w", err)
	}

	var fielddata []FielddataInfo
	if err := json.Unmarshal(body, &fielddata); err != nil {
		return nil, fmt.Errorf("parsing fielddata: %w", err)
	}

	sort.Slice(fielddata, func(i, j int) bool {
		sizeI := parseSizeToBytes(fielddata[i].Size)
		sizeJ := parseSizeToBytes(fielddata[j].Size)
		if sizeI != sizeJ {
			return sizeI > sizeJ
		}
		return fielddata[i].Field < fielddata[j].Field
	})

	return fielddata, nil
}

func (c *Client) FetchFielddataByIndex(ctx context.Context) ([]FielddataByIndex, error) {
	res, err := c.es.Nodes.Stats(
		c.es.Nodes.Stats.WithContext(ctx),
		c.es.Nodes.Stats.WithMetric("indices"),
		c.es.Nodes.Stats.WithIndexMetric("fielddata"),
		c.es.Nodes.Stats.WithLevel("indices"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching fielddata by index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading fielddata by index response: %w", err)
	}

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

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing fielddata by index: %w", err)
	}

	indexSizes := make(map[string]int64)
	for _, node := range response.Nodes {
		for indexName, indexData := range node.Indices.Indices {
			indexSizes[indexName] += indexData.Fielddata.MemorySizeInBytes
		}
	}

	var result []FielddataByIndex
	for indexName, size := range indexSizes {
		result = append(result, FielddataByIndex{
			Index: indexName,
			Size:  size,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Size != result[j].Size {
			return result[i].Size > result[j].Size
		}
		return result[i].Index < result[j].Index
	})

	if len(result) > 20 {
		result = result[:20]
	}

	return result, nil
}

func (c *Client) FetchNodesState(ctx context.Context) (*NodesState, error) {
	state := &NodesState{}

	nodes, err := c.FetchNodeStats(ctx)
	if err != nil {
		return nil, err
	}
	state.Nodes = nodes

	fielddata, err := c.FetchFielddataByField(ctx)
	if err != nil {
		return nil, err
	}
	state.FielddataByField = fielddata

	fielddataByIndex, err := c.FetchFielddataByIndex(ctx)
	if err != nil {
		return nil, err
	}
	state.FielddataByIndex = fielddataByIndex

	return state, nil
}

func (c *Client) FetchTasks(ctx context.Context) ([]TaskInfo, error) {
	res, err := c.es.Tasks.List(
		c.es.Tasks.List.WithContext(ctx),
		c.es.Tasks.List.WithDetailed(true),
		c.es.Tasks.List.WithActions("*reindex*", "*byquery*", "*forcemerge*", "*snapshot*"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching tasks: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tasks response: %w", err)
	}

	return parseTasksResponse(body)
}

func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	res, err := c.es.Tasks.Cancel(
		c.es.Tasks.Cancel.WithContext(ctx),
		c.es.Tasks.Cancel.WithTaskID(taskID),
	)
	if err != nil {
		return fmt.Errorf("cancelling task: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}

	return nil
}

func parseTasksResponse(data []byte) ([]TaskInfo, error) {
	var response struct {
		Nodes map[string]struct {
			Name  string `json:"name"`
			Tasks map[string]struct {
				Node             string `json:"node"`
				ID               int64  `json:"id"`
				Action           string `json:"action"`
				Description      string `json:"description"`
				RunningTimeNanos int64  `json:"running_time_in_nanos"`
				Cancellable      bool   `json:"cancellable"`
			} `json:"tasks"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing tasks response: %w", err)
	}

	actionPrefixes := []string{
		"indices:data/write/reindex",
		"indices:data/write/update/byquery",
		"indices:data/write/delete/byquery",
		"indices:admin/forcemerge",
		"cluster:admin/snapshot",
	}

	isTargetAction := func(action string) bool {
		for _, prefix := range actionPrefixes {
			if len(action) >= len(prefix) && action[:len(prefix)] == prefix {
				return true
			}
		}
		return false
	}

	var tasks []TaskInfo
	for nodeID, nodeData := range response.Nodes {
		for taskID, task := range nodeData.Tasks {
			if !task.Cancellable || !isTargetAction(task.Action) {
				continue
			}

			runningMs := task.RunningTimeNanos / 1_000_000
			tasks = append(tasks, TaskInfo{
				ID:            taskID,
				Action:        task.Action,
				Node:          nodeData.Name,
				Description:   task.Description,
				RunningTime:   formatDuration(runningMs),
				RunningTimeMs: runningMs,
				Cancellable:   task.Cancellable,
			})
			_ = nodeID
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].RunningTimeMs > tasks[j].RunningTimeMs
	})

	return tasks, nil
}

func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func parseMappingResponse(data []byte) ([]string, error) {
	var response map[string]struct {
		Mappings struct {
			Properties map[string]interface{} `json:"properties"`
		} `json:"mappings"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing mapping response: %w", err)
	}

	var fields []string
	for _, indexData := range response {
		fields = extractFields(indexData.Mappings.Properties, "")
		break
	}

	return fields, nil
}

func extractFields(properties map[string]interface{}, prefix string) []string {
	var fields []string

	for name, prop := range properties {
		fieldName := name
		if prefix != "" {
			fieldName = prefix + "." + name
		}

		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}

		if nested, ok := propMap["properties"].(map[string]interface{}); ok {
			fields = append(fields, extractFields(nested, fieldName)...)
		} else {
			fields = append(fields, fieldName)
		}
	}

	return fields
}

func (c *Client) FetchMapping(ctx context.Context, index string) ([]string, error) {
	res, err := c.es.Indices.GetMapping(
		c.es.Indices.GetMapping.WithContext(ctx),
		c.es.Indices.GetMapping.WithIndex(index),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching mapping: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES error %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading mapping response: %w", err)
	}

	return parseMappingResponse(body)
}

func parseSizeToBytes(size string) int64 {
	size = strings.TrimSpace(strings.ToLower(size))
	if size == "" || size == "-" {
		return 0
	}

	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"tb", 1024 * 1024 * 1024 * 1024},
		{"gb", 1024 * 1024 * 1024},
		{"mb", 1024 * 1024},
		{"kb", 1024},
		{"b", 1},
	}

	for _, s := range suffixes {
		if strings.HasSuffix(size, s.suffix) {
			numStr := strings.TrimSuffix(size, s.suffix)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0
			}
			return int64(num * float64(s.mult))
		}
	}

	num, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return 0
	}
	return num
}
