package es

import "encoding/json"

type IndexInfo struct {
	Name         string `json:"index"`
	Health       string `json:"health"`
	Status       string `json:"status"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	Version      string `json:"-"`
}

type NodeInfo struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Role    string `json:"node.role"`
	Version string `json:"version"`
	Master  string `json:"master"`
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
	Master         string `json:"master"`
	HeapPercent    string `json:"heap.percent"`
	HeapCurrent    string `json:"heap.current"`
	HeapMax        string `json:"heap.max"`
	FielddataSize  string `json:"fielddata.memory_size"`
	QueryCacheSize string `json:"query_cache.memory_size"`
	SegmentsCount  string `json:"segments.count"`
	DiskPercent    string `json:"disk.used_percent"`
	DiskAvail      string `json:"disk.avail"`
	DiskTotal      string `json:"disk.total"`
	DiskUsed       string `json:"disk.used"`
	Shards         string `json:"shard_stats.total_count"`
}

type FielddataEntry struct {
	Node  string
	Index string
	Field string
	Size  int64
}

type NodesState struct {
	Nodes     []NodeStats
	Fielddata []FielddataEntry
}

type TaskInfo struct {
	ID            string
	Action        string
	Node          string
	RunningTime   string
	RunningTimeMs int64
	Description   string
	Cancellable   bool
}

type AllocationExplain struct {
	Index             string
	Shard             int
	Primary           bool
	CurrentState      string
	UnassignedReason  string
	AllocationStatus  string
	ExplanationDetail string
}

type MappingField struct {
	Name       string
	Type       string
	Properties map[string]string
	Children   []MappingField
}

type AnalyzerInfo struct {
	Name     string
	Kind     string
	Settings map[string]string
}

type IndexMappings struct {
	IndexName  string
	FieldCount int
	Fields     []MappingField
	Analyzers  []AnalyzerInfo
}

type IndexSettings struct {
	IndexName         string
	NumberOfShards    string
	NumberOfReplicas  string
	RefreshInterval   string
	Codec             string
	CreationDate      string
	UUID              string
	Version           string
	RoutingAllocation string
	AllSettings       map[string]string
}

type ClusterSettings struct {
	Persistent map[string]string
	Transient  map[string]string
	Defaults   map[string]string
}

type ThreadPoolInfo struct {
	NodeName  string `json:"node_name"`
	Name      string `json:"name"`
	Active    string `json:"active"`
	Queue     string `json:"queue"`
	Rejected  string `json:"rejected"`
	Completed string `json:"completed"`
	PoolSize  string `json:"pool_size"`
	PoolType  string `json:"type"`
}

type PendingTask struct {
	InsertOrder       int    `json:"insert_order"`
	Priority          string `json:"priority"`
	Source            string `json:"source"`
	Executing         bool   `json:"executing"`
	TimeInQueueMillis int64  `json:"time_in_queue_millis"`
	TimeInQueue       string `json:"time_in_queue"`
}

type IndexTemplate struct {
	Name             string
	IndexPatterns    []string
	Priority         int
	Version          int
	ComposedOf       []string
	NumberOfShards   string
	NumberOfReplicas string
	DataStream       bool
}

type ShardHealthStatus int

const (
	ShardHealthOK ShardHealthStatus = iota
	ShardHealthWarning
	ShardHealthCritical
)

type ShardHealth struct {
	IndexName       string
	Status          ShardHealthStatus
	StatusText      string
	ShardCount      int
	TotalSize       int64
	AvgShardSize    int64
	DocsCount       int64
	AvgDocsPerShard int64
	Issues          []string
}

type DocumentHit struct {
	ID     string
	Index  string
	Source string
	Sort   []interface{}
}

type SearchResult struct {
	Hits  []DocumentHit
	Total int64
}

type ClusterHealth struct {
	Status              string `json:"status"`
	ActivePrimaryShards int    `json:"active_primary_shards"`
	ActiveShards        int    `json:"active_shards"`
	UnassignedShards    int    `json:"unassigned_shards"`
}

type ClusterState struct {
	Health  *ClusterHealth
	Indices []IndexInfo
	Nodes   []NodeInfo
	Shards  []ShardInfo
	Aliases []AliasInfo
}

type RecoveryInfo struct {
	Index       string `json:"index"`
	Shard       string `json:"shard"`
	Type        string `json:"type"`
	Stage       string `json:"stage"`
	SourceNode  string `json:"source_node"`
	TargetNode  string `json:"target_node"`
	BytesTotal  string `json:"bytes_total"`
	BytesPct    string `json:"bytes_percent"`
	FilesTotal  string `json:"files_total"`
	FilesPct    string `json:"files_percent"`
	TranslogOps string `json:"translog_ops_recovered"`
}

type Deprecation struct {
	Level    string `json:"level"`
	Message  string `json:"message"`
	URL      string `json:"url"`
	Details  string `json:"details"`
	Category string `json:"-"`
	Resource string `json:"-"`
}

type DeprecationInfo struct {
	Deprecations []Deprecation
}
