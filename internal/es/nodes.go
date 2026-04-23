package es

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

func (c *Client) FetchNodeStats(ctx context.Context) ([]NodeStats, error) {
	res, err := c.es.Cat.Nodes(
		c.es.Cat.Nodes.WithContext(ctx),
		c.es.Cat.Nodes.WithFormat("json"),
		c.es.Cat.Nodes.WithH(
			"name", "version", "master", "heap.percent", "heap.current", "heap.max",
			"fielddata.memory_size", "query_cache.memory_size", "segments.count",
			"disk.used_percent", "disk.avail", "disk.total", "disk.used",
			"shard_stats.total_count",
		),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching node stats: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "node stats")
	if err != nil {
		return nil, err
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

func (c *Client) FetchFielddata(ctx context.Context) ([]FielddataEntry, error) {
	res, err := c.es.Nodes.Stats(
		c.es.Nodes.Stats.WithContext(ctx),
		c.es.Nodes.Stats.WithMetric("indices"),
		c.es.Nodes.Stats.WithIndexMetric("fielddata"),
		c.es.Nodes.Stats.WithLevel("indices"),
		c.es.Nodes.Stats.WithFields("*"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching fielddata: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "fielddata")
	if err != nil {
		return nil, err
	}

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

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing fielddata: %w", err)
	}

	var result []FielddataEntry
	for _, node := range response.Nodes {
		for indexName, indexData := range node.Indices.Indices {
			if len(indexData.Fielddata.Fields) > 0 {
				for fieldName, fieldData := range indexData.Fielddata.Fields {
					if fieldData.MemorySizeInBytes > 0 {
						result = append(result, FielddataEntry{
							Node:  node.Name,
							Index: indexName,
							Field: fieldName,
							Size:  fieldData.MemorySizeInBytes,
						})
					}
				}
			} else if indexData.Fielddata.MemorySizeInBytes > 0 {
				result = append(result, FielddataEntry{
					Node:  node.Name,
					Index: indexName,
					Field: "",
					Size:  indexData.Fielddata.MemorySizeInBytes,
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Size != result[j].Size {
			return result[i].Size > result[j].Size
		}
		if result[i].Index != result[j].Index {
			return result[i].Index < result[j].Index
		}
		if result[i].Field != result[j].Field {
			return result[i].Field < result[j].Field
		}
		return result[i].Node < result[j].Node
	})

	return result, nil
}

func (c *Client) FetchNodesState(ctx context.Context) (*NodesState, error) {
	state := &NodesState{}

	nodes, err := c.FetchNodeStats(ctx)
	if err != nil {
		return nil, err
	}
	state.Nodes = nodes

	fielddata, err := c.FetchFielddata(ctx)
	if err != nil {
		return nil, err
	}
	state.Fielddata = fielddata

	return state, nil
}

func (c *Client) FetchThreadPools(ctx context.Context) ([]ThreadPoolInfo, error) {
	res, err := c.es.Cat.ThreadPool(
		c.es.Cat.ThreadPool.WithContext(ctx),
		c.es.Cat.ThreadPool.WithFormat("json"),
		c.es.Cat.ThreadPool.WithH("node_name", "name", "active", "queue", "rejected", "completed", "pool_size", "type"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching thread pools: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "thread pools")
	if err != nil {
		return nil, err
	}

	var pools []ThreadPoolInfo
	if err := json.Unmarshal(body, &pools); err != nil {
		return nil, fmt.Errorf("parsing thread pools: %w", err)
	}

	sort.Slice(pools, func(i, j int) bool {
		if pools[i].NodeName != pools[j].NodeName {
			return pools[i].NodeName < pools[j].NodeName
		}
		return pools[i].Name < pools[j].Name
	})

	return pools, nil
}

func (c *Client) FetchHotThreads(ctx context.Context) (string, error) {
	res, err := c.es.Nodes.HotThreads(
		c.es.Nodes.HotThreads.WithContext(ctx),
		c.es.Nodes.HotThreads.WithThreads(3),
		c.es.Nodes.HotThreads.WithInterval(500*time.Millisecond),
		c.es.Nodes.HotThreads.WithSnapshots(10),
	)
	if err != nil {
		return "", fmt.Errorf("fetching hot threads: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "hot threads")
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *Client) FetchRecovery(ctx context.Context) ([]RecoveryInfo, error) {
	res, err := c.es.Cat.Recovery(
		c.es.Cat.Recovery.WithContext(ctx),
		c.es.Cat.Recovery.WithFormat("json"),
		c.es.Cat.Recovery.WithActiveOnly(true),
		c.es.Cat.Recovery.WithH("index", "shard", "type", "stage", "source_node", "target_node", "bytes_total", "bytes_percent", "files_total", "files_percent", "translog_ops_recovered"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching recovery: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "recovery")
	if err != nil {
		return nil, err
	}

	var recovery []RecoveryInfo
	if err := json.Unmarshal(body, &recovery); err != nil {
		return nil, fmt.Errorf("parsing recovery: %w", err)
	}

	return recovery, nil
}
