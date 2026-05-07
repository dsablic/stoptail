package es

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/labtiva/stoptail/internal/config"
)

type Client struct {
	es         *elasticsearch.Client
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.IsAWS() && cfg.IsMTLS() {
		return nil, fmt.Errorf("cannot use both AWS and mTLS authentication")
	}

	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Host},
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	if cfg.IsMTLS() {
		tlsCert, err := tls.X509KeyPair([]byte(cfg.TLSCert), []byte(cfg.TLSKey))
		if err != nil {
			return nil, fmt.Errorf("loading mTLS credentials: %w", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS12,
		}
		if cfg.TLSCA != "" {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM([]byte(cfg.TLSCA)) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = pool
		}
		transport.TLSClientConfig = tlsConfig
		esCfg.Transport = transport
	}

	var httpTransport http.RoundTripper = transport

	if cfg.IsAWS() {
		awsTransport, err := newAWSTransport(cfg)
		if err != nil {
			return nil, err
		}
		esCfg.Transport = awsTransport
		httpTransport = awsTransport
	} else if cfg.Username != "" {
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
		httpClient: &http.Client{Timeout: 30 * time.Second, Transport: httpTransport},
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

func readBody(res *esapi.Response, errContext string) ([]byte, error) {
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", errContext, err)
	}
	return body, nil
}

func checkError(res *esapi.Response) error {
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES error %s: %s", res.Status(), string(body))
	}
	return nil
}
