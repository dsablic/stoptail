package es

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading indices response: %w", err)
	}

	var indices []IndexInfo
	if err := json.Unmarshal(body, &indices); err != nil {
		return nil, fmt.Errorf("parsing indices: %w", err)
	}

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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading nodes response: %w", err)
	}

	var nodes []NodeInfo
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("parsing nodes: %w", err)
	}

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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading shards response: %w", err)
	}

	var shards []ShardInfo
	if err := json.Unmarshal(body, &shards); err != nil {
		return nil, fmt.Errorf("parsing shards: %w", err)
	}

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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading aliases response: %w", err)
	}

	var aliases []AliasInfo
	if err := json.Unmarshal(body, &aliases); err != nil {
		return nil, fmt.Errorf("parsing aliases: %w", err)
	}

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
