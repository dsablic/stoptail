package es

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	BytesKB int64 = 1024
	BytesMB       = BytesKB * 1024
	BytesGB       = BytesMB * 1024
	BytesTB       = BytesGB * 1024
)

func ParseSize(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0
	}

	var multiplier int64 = 1
	var numStr string

	switch {
	case strings.HasSuffix(s, "tb"):
		multiplier = BytesTB
		numStr = strings.TrimSuffix(s, "tb")
	case strings.HasSuffix(s, "gb"):
		multiplier = BytesGB
		numStr = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "mb"):
		multiplier = BytesMB
		numStr = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "kb"):
		multiplier = BytesKB
		numStr = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "b"):
		multiplier = 1
		numStr = strings.TrimSuffix(s, "b")
	case strings.HasSuffix(s, "t"):
		multiplier = BytesTB
		numStr = strings.TrimSuffix(s, "t")
	case strings.HasSuffix(s, "g"):
		multiplier = BytesGB
		numStr = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "m"):
		multiplier = BytesMB
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

func FormatBytes(b int64) string {
	switch {
	case b >= BytesTB:
		return fmt.Sprintf("%.1ftb", float64(b)/float64(BytesTB))
	case b >= BytesGB:
		return fmt.Sprintf("%.1fgb", float64(b)/float64(BytesGB))
	case b >= BytesMB:
		return fmt.Sprintf("%.1fmb", float64(b)/float64(BytesMB))
	case b >= BytesKB:
		return fmt.Sprintf("%.1fkb", float64(b)/float64(BytesKB))
	default:
		return fmt.Sprintf("%db", b)
	}
}

func AnalyzeShardHealth(idx IndexInfo) ShardHealth {
	shardCount, _ := strconv.Atoi(idx.Pri)
	docsCount, _ := strconv.ParseInt(idx.DocsCount, 10, 64)
	totalSize := ParseSize(idx.PriStoreSize)

	health := ShardHealth{
		IndexName:  idx.Name,
		Status:     ShardHealthOK,
		StatusText: "ok",
		ShardCount: shardCount,
		TotalSize:  totalSize,
		DocsCount:  docsCount,
	}

	if shardCount > 0 {
		health.AvgShardSize = totalSize / int64(shardCount)
		health.AvgDocsPerShard = docsCount / int64(shardCount)
	}

	if shardCount == 0 || totalSize == 0 {
		return health
	}

	var issues []string
	var statusTerms []string

	if health.AvgShardSize > 65*BytesGB {
		issues = append(issues, fmt.Sprintf("shards oversized (%s avg)", FormatBytes(health.AvgShardSize)))
		statusTerms = append(statusTerms, "oversized")
		health.Status = ShardHealthCritical
	} else if health.AvgShardSize > 50*BytesGB {
		issues = append(issues, fmt.Sprintf("shards large (%s avg)", FormatBytes(health.AvgShardSize)))
		statusTerms = append(statusTerms, "oversized")
		health.Status = ShardHealthWarning
	}

	if shardCount > 3 {
		if health.AvgShardSize < 500*BytesMB {
			issues = append(issues, fmt.Sprintf("shards undersized (%s avg)", FormatBytes(health.AvgShardSize)))
			statusTerms = append(statusTerms, "undersized")
			health.Status = ShardHealthCritical
		} else if health.AvgShardSize < 1*BytesGB {
			issues = append(issues, fmt.Sprintf("shards small (%s avg)", FormatBytes(health.AvgShardSize)))
			statusTerms = append(statusTerms, "undersized")
			if health.Status == ShardHealthOK {
				health.Status = ShardHealthWarning
			}
		}

		if health.AvgDocsPerShard < 100000 {
			issues = append(issues, fmt.Sprintf("sparse shards (%d docs/shard)", health.AvgDocsPerShard))
			statusTerms = append(statusTerms, "sparse")
			if health.Status == ShardHealthOK {
				health.Status = ShardHealthWarning
			}
		}
	}

	if len(issues) > 1 {
		health.Status = ShardHealthCritical
	}

	if len(statusTerms) > 0 {
		health.StatusText = strings.Join(statusTerms, ", ")
	}

	health.Issues = issues
	return health
}

func sortShardsByIndexShardPrimary(shards []ShardInfo) {
	sort.Slice(shards, func(i, j int) bool {
		if shards[i].Index != shards[j].Index {
			return shards[i].Index < shards[j].Index
		}
		if shards[i].Shard != shards[j].Shard {
			return shards[i].Shard < shards[j].Shard
		}
		return shards[i].Primary && !shards[j].Primary
	})
}

func sortShardsByShardPrimary(shards []ShardInfo) {
	sort.Slice(shards, func(i, j int) bool {
		if shards[i].Shard != shards[j].Shard {
			return shards[i].Shard < shards[j].Shard
		}
		return shards[i].Primary && !shards[j].Primary
	})
}

func DecodeESVersion(versionID string) string {
	var v int
	if _, err := fmt.Sscanf(versionID, "%d", &v); err != nil {
		return versionID
	}
	major := v / 1000000
	return fmt.Sprintf("%d.x", major)
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
		shardNode := sh.Node
		if idx := strings.Index(shardNode, " -> "); idx != -1 {
			shardNode = shardNode[:idx]
		}
		if sh.Index == index && shardNode == node {
			shards = append(shards, sh)
		}
	}
	sortShardsByShardPrimary(shards)
	return shards
}

func (s *ClusterState) GetUnassignedShardsForIndex(index string) []ShardInfo {
	var shards []ShardInfo
	for _, sh := range s.Shards {
		if sh.Index == index && (sh.Node == "" || sh.State == "UNASSIGNED") {
			shards = append(shards, sh)
		}
	}
	sortShardsByShardPrimary(shards)
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

func formatSettingValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}
