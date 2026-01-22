package es

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ValidateResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func (c *Client) ValidateQuery(ctx context.Context, index string, query json.RawMessage) (*ValidateResult, error) {
	body := map[string]json.RawMessage{"query": query}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling query: %w", err)
	}

	path := fmt.Sprintf("/%s/_validate/query", index)
	if index == "" {
		path = "/_validate/query"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.Host+path, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result ValidateResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}
