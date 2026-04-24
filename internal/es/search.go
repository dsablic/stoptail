package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) SearchDocuments(ctx context.Context, index string, after []interface{}, size int) (*SearchResult, error) {
	query := map[string]interface{}{
		"size": size,
		"sort": []map[string]string{{"_doc": "asc"}},
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}
	if len(after) > 0 {
		query["search_after"] = after
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshaling query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(queryBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("searching documents: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "search")
	if err != nil {
		return nil, err
	}

	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string          `json:"_id"`
				Index  string          `json:"_index"`
				Source json.RawMessage `json:"_source"`
				Sort   []interface{}   `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	result := &SearchResult{
		Total: response.Hits.Total.Value,
		Hits:  make([]DocumentHit, len(response.Hits.Hits)),
	}

	for i, hit := range response.Hits.Hits {
		result.Hits[i] = DocumentHit{
			ID:     hit.ID,
			Index:  hit.Index,
			Source: string(hit.Source),
			Sort:   hit.Sort,
		}
	}

	return result, nil
}
