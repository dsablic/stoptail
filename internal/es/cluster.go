package es

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

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

	health, err := c.fetchClusterHealth(ctx)
	if err != nil {
		return nil, err
	}
	state.Health = health

	versions, err := c.fetchIndexVersions(ctx)
	if err == nil {
		for i := range state.Indices {
			if v, ok := versions[state.Indices[i].Name]; ok {
				state.Indices[i].Version = v
			}
		}
	}

	return state, nil
}

func (c *Client) fetchClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	res, err := c.es.Cluster.Health(c.es.Cluster.Health.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("fetching cluster health: %w", err)
	}
	defer res.Body.Close()

	if err := checkError(res); err != nil {
		return nil, err
	}

	var health ClusterHealth
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("parsing cluster health: %w", err)
	}

	return &health, nil
}

func (c *Client) fetchIndices(ctx context.Context) ([]IndexInfo, error) {
	res, err := c.es.Cat.Indices(
		c.es.Cat.Indices.WithContext(ctx),
		c.es.Cat.Indices.WithFormat("json"),
		c.es.Cat.Indices.WithExpandWildcards("all"),
		c.es.Cat.Indices.WithH("index", "health", "status", "docs.count", "docs.deleted", "store.size", "pri.store.size", "pri", "rep"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching indices: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "indices")
	if err != nil {
		return nil, err
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
		c.es.Cat.Nodes.WithH("name", "ip", "node.role", "version", "master"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching nodes: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "nodes")
	if err != nil {
		return nil, err
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

	body, err := readBody(res, "shards")
	if err != nil {
		return nil, err
	}

	var shards []ShardInfo
	if err := json.Unmarshal(body, &shards); err != nil {
		return nil, fmt.Errorf("parsing shards: %w", err)
	}

	sortShardsByIndexShardPrimary(shards)

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

	body, err := readBody(res, "aliases")
	if err != nil {
		return nil, err
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

func (c *Client) fetchIndexVersions(ctx context.Context) (map[string]string, error) {
	res, err := c.es.Indices.GetSettings(
		c.es.Indices.GetSettings.WithContext(ctx),
		c.es.Indices.GetSettings.WithFlatSettings(true),
		c.es.Indices.GetSettings.WithName("index.version.created"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching index versions: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "versions")
	if err != nil {
		return nil, err
	}

	var response map[string]struct {
		Settings map[string]string `json:"settings"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing index versions: %w", err)
	}

	versions := make(map[string]string)
	for indexName, indexData := range response {
		if v, ok := indexData.Settings["index.version.created"]; ok {
			versions[indexName] = DecodeESVersion(v)
		}
	}

	return versions, nil
}

func (c *Client) FetchAllocationExplain(ctx context.Context, index string, shard int, primary bool) (*AllocationExplain, error) {
	reqBodyObj := struct {
		Index   string `json:"index"`
		Shard   int    `json:"shard"`
		Primary bool   `json:"primary"`
	}{index, shard, primary}
	reqBytes, _ := json.Marshal(reqBodyObj)
	res, err := c.es.Cluster.AllocationExplain(
		c.es.Cluster.AllocationExplain.WithContext(ctx),
		c.es.Cluster.AllocationExplain.WithBody(strings.NewReader(string(reqBytes))),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching allocation explain: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "allocation explain")
	if err != nil {
		return nil, err
	}

	return parseAllocationExplain(body)
}

func parseAllocationExplain(data []byte) (*AllocationExplain, error) {
	var response struct {
		Index                string `json:"index"`
		Shard                int    `json:"shard"`
		Primary              bool   `json:"primary"`
		CurrentState         string `json:"current_state"`
		UnassignedInfo       *struct {
			Reason string `json:"reason"`
			At     string `json:"at"`
		} `json:"unassigned_info"`
		CanAllocate          string `json:"can_allocate"`
		AllocateExplanation  string `json:"allocate_explanation"`
		CanMoveToOtherNode   string `json:"can_move_to_other_node"`
		MoveExplanation      string `json:"move_explanation"`
		CanRebalanceCluster  string `json:"can_rebalance_cluster"`
		RebalanceExplanation string `json:"rebalance_explanation"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing allocation explain: %w", err)
	}

	result := &AllocationExplain{
		Index:        response.Index,
		Shard:        response.Shard,
		Primary:      response.Primary,
		CurrentState: response.CurrentState,
	}

	if response.UnassignedInfo != nil {
		result.UnassignedReason = response.UnassignedInfo.Reason
	}

	if response.CanAllocate != "" {
		result.AllocationStatus = response.CanAllocate
		result.ExplanationDetail = response.AllocateExplanation
	} else if response.CanMoveToOtherNode != "" {
		result.AllocationStatus = response.CanMoveToOtherNode
		result.ExplanationDetail = response.MoveExplanation
	}

	return result, nil
}

func (c *Client) FetchClusterSettings(ctx context.Context) (*ClusterSettings, error) {
	res, err := c.es.Cluster.GetSettings(
		c.es.Cluster.GetSettings.WithContext(ctx),
		c.es.Cluster.GetSettings.WithFlatSettings(true),
		c.es.Cluster.GetSettings.WithIncludeDefaults(true),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching cluster settings: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "cluster settings")
	if err != nil {
		return nil, err
	}

	return parseClusterSettings(body)
}

func parseClusterSettings(data []byte) (*ClusterSettings, error) {
	var response struct {
		Persistent map[string]any `json:"persistent"`
		Transient  map[string]any `json:"transient"`
		Defaults   map[string]any `json:"defaults"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing cluster settings: %w", err)
	}

	settings := &ClusterSettings{
		Persistent: flattenSettings(response.Persistent),
		Transient:  flattenSettings(response.Transient),
		Defaults:   flattenSettings(response.Defaults),
	}

	return settings, nil
}

func flattenSettings(m map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = formatSettingValue(v)
	}
	return result
}

func (c *Client) FetchDeprecations(ctx context.Context) (*DeprecationInfo, error) {
	res, err := c.es.Migration.Deprecations(c.es.Migration.Deprecations.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("fetching deprecations: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "deprecations")
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing deprecations: %w", err)
	}

	info := &DeprecationInfo{}

	categoryOrder := []string{"cluster_settings", "node_settings", "ml_settings", "ilm_policies", "data_streams", "index_settings", "templates"}

	for _, catName := range categoryOrder {
		rawData, ok := raw[catName]
		if !ok {
			continue
		}

		var deps []Deprecation
		if err := json.Unmarshal(rawData, &deps); err == nil {
			for i := range deps {
				deps[i].Category = catName
			}
			info.Deprecations = append(info.Deprecations, deps...)
		} else {
			var mapDeps map[string][]Deprecation
			if err := json.Unmarshal(rawData, &mapDeps); err == nil {
				for resource, resourceDeps := range mapDeps {
					for i := range resourceDeps {
						resourceDeps[i].Category = catName
						resourceDeps[i].Resource = resource
					}
					info.Deprecations = append(info.Deprecations, resourceDeps...)
				}
			}
		}
	}

	sort.Slice(info.Deprecations, func(i, j int) bool {
		levelOrder := map[string]int{"critical": 0, "warning": 1, "": 2}
		if levelOrder[info.Deprecations[i].Level] != levelOrder[info.Deprecations[j].Level] {
			return levelOrder[info.Deprecations[i].Level] < levelOrder[info.Deprecations[j].Level]
		}
		if info.Deprecations[i].Category != info.Deprecations[j].Category {
			return info.Deprecations[i].Category < info.Deprecations[j].Category
		}
		return info.Deprecations[i].Resource < info.Deprecations[j].Resource
	})

	return info, nil
}
