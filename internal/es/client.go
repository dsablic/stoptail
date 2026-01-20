package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labtiva/stoptail/internal/config"
)

type Client struct {
	es         *elasticsearch.Client
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Host},
	}

	if cfg.Username != "" {
		esCfg.Username = cfg.Username
		esCfg.Password = cfg.Password
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("creating ES client: %w", err)
	}

	return &Client{
		es:         es,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Info(c.es.Info.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("connecting to ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES error: %s", res.Status())
	}
	return nil
}

type RequestResult struct {
	StatusCode int
	Body       string
	Duration   time.Duration
	Error      error
}

func (c *Client) Request(ctx context.Context, method, path, body string) RequestResult {
	start := time.Now()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.Host+path, bodyReader)
	if err != nil {
		return RequestResult{Error: err, Duration: time.Since(start)}
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return RequestResult{Error: err, Duration: time.Since(start)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return RequestResult{Error: err, Duration: time.Since(start)}
	}

	// Pretty print JSON
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, respBody, "", "  "); err == nil {
		return RequestResult{
			StatusCode: resp.StatusCode,
			Body:       pretty.String(),
			Duration:   time.Since(start),
		}
	}

	return RequestResult{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Duration:   time.Since(start),
	}
}
