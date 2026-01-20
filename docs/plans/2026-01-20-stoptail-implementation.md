# stoptail Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a TUI Elasticsearch client with cluster shard grid visualization and request workbench.

**Architecture:** Two-tab bubbletea app. Overview tab shows nodes×indices shard grid. Workbench tab is split-pane request editor + response viewer. ES client wraps go-elasticsearch/v8 with URL-based connection string.

**Tech Stack:** bubbletea, bubbles (textarea, viewport, textinput), lipgloss, go-elasticsearch/v8, chroma

---

## Phase 1: Project Scaffolding

### Task 1.1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `go.sum`

**Step 1: Initialize module**

Run:
```bash
cd /Users/denis/Documents/labtiva/stoptail/.worktrees/implement
go mod init github.com/labtiva/stoptail
```

**Step 2: Add dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/elastic/go-elasticsearch/v8@latest
go get github.com/alecthomas/chroma/v2@latest
```

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: initialize go module with dependencies"
```

---

### Task 1.2: Create Basic Bubbletea Shell

**Files:**
- Create: `main.go`
- Create: `internal/ui/model.go`
- Create: `internal/ui/styles.go`

**Step 1: Create styles**

Create `internal/ui/styles.go`:
```go
package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorGreen  = lipgloss.Color("#22c55e")
	ColorYellow = lipgloss.Color("#eab308")
	ColorRed    = lipgloss.Color("#ef4444")
	ColorBlue   = lipgloss.Color("#3b82f6")
	ColorGray   = lipgloss.Color("#6b7280")
	ColorWhite  = lipgloss.Color("#f9fafb")

	// Base styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(lipgloss.Color("#1f2937")).
			Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	ActiveTabStyle = TabStyle.
			Bold(true).
			Foreground(ColorWhite).
			Background(lipgloss.Color("#374151"))

	InactiveTabStyle = TabStyle.
				Foreground(ColorGray)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorGray)
)
```

**Step 2: Create main model**

Create `internal/ui/model.go`:
```go
package ui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	TabOverview = iota
	TabWorkbench
)

type Model struct {
	activeTab int
	width     int
	height    int
	quitting  bool
}

func New() Model {
	return Model{
		activeTab: TabOverview,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	header := HeaderStyle.Width(m.width).Render("stoptail · not connected")

	// Tabs
	var tabs string
	if m.activeTab == TabOverview {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			ActiveTabStyle.Render("Overview"),
			InactiveTabStyle.Render("Workbench"),
		)
	} else {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			InactiveTabStyle.Render("Overview"),
			ActiveTabStyle.Render("Workbench"),
		)
	}

	// Content placeholder
	contentHeight := m.height - 4 // header + tabs + status
	content := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render("Press Tab to switch views")

	// Status bar
	status := StatusBarStyle.Width(m.width).Render("q: quit  Tab: switch view")

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, status)
}
```

**Step 3: Create main.go**

Create `main.go`:
```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/labtiva/stoptail/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 4: Verify it builds and runs**

Run:
```bash
go build -o stoptail . && ./stoptail
```
Expected: TUI appears with header, tabs, and status bar. Press Tab to switch tabs. Press q to quit.

**Step 5: Commit**

```bash
git add main.go internal/
git commit -m "feat: add basic bubbletea shell with tab switching"
```

---

## Phase 2: ES Client & Connection

### Task 2.1: URL Parser

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing test**

Create `internal/config/config_test.go`:
```go
package config

import "testing"

func TestParseURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantHost string
		wantUser string
		wantPass string
		wantErr  bool
	}{
		{
			name:     "full url with auth",
			url:      "https://elastic:secret@localhost:9200",
			wantHost: "https://localhost:9200",
			wantUser: "elastic",
			wantPass: "secret",
		},
		{
			name:     "url without auth",
			url:      "http://localhost:9200",
			wantHost: "http://localhost:9200",
			wantUser: "",
			wantPass: "",
		},
		{
			name:     "url with special chars in password",
			url:      "https://elastic:p%40ssw0rd@localhost:9200",
			wantHost: "https://localhost:9200",
			wantUser: "elastic",
			wantPass: "p@ssw0rd",
		},
		{
			name:    "invalid url",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Host != tt.wantHost {
				t.Errorf("host = %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.Username != tt.wantUser {
				t.Errorf("username = %q, want %q", cfg.Username, tt.wantUser)
			}
			if cfg.Password != tt.wantPass {
				t.Errorf("password = %q, want %q", cfg.Password, tt.wantPass)
			}
		})
	}
}

func TestMaskedURL(t *testing.T) {
	cfg := &Config{
		Host:     "https://localhost:9200",
		Username: "elastic",
		Password: "secret",
	}
	got := cfg.MaskedURL()
	want := "elastic:***@localhost:9200"
	if got != want {
		t.Errorf("MaskedURL() = %q, want %q", got, want)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/config/... -v
```
Expected: FAIL - package not found

**Step 3: Write implementation**

Create `internal/config/config.go`:
```go
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Host     string
	Username string
	Password string
}

func ParseURL(rawURL string) (*Config, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: missing scheme or host")
	}

	cfg := &Config{
		Host: fmt.Sprintf("%s://%s", u.Scheme, u.Host),
	}

	if u.User != nil {
		cfg.Username = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}

	return cfg, nil
}

func Load(flagURL string) (*Config, error) {
	rawURL := flagURL
	if rawURL == "" {
		rawURL = os.Getenv("ES_URL")
	}
	if rawURL == "" {
		rawURL = "http://localhost:9200"
	}
	return ParseURL(rawURL)
}

func (c *Config) MaskedURL() string {
	u, _ := url.Parse(c.Host)
	if c.Username != "" {
		return fmt.Sprintf("%s:***@%s", c.Username, u.Host)
	}
	return u.Host
}

func (c *Config) DisplayHost() string {
	u, _ := url.Parse(c.Host)
	return strings.TrimPrefix(u.Host, "www.")
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/config/... -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add URL parser for ES connection config"
```

---

### Task 2.2: ES Client Wrapper

**Files:**
- Create: `internal/es/client.go`
- Create: `internal/es/client_test.go`

**Step 1: Write failing test**

Create `internal/es/client_test.go`:
```go
package es

import (
	"testing"

	"github.com/labtiva/stoptail/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Host:     "http://localhost:9200",
		Username: "elastic",
		Password: "test",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/es/... -v
```
Expected: FAIL - package not found

**Step 3: Write implementation**

Create `internal/es/client.go`:
```go
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
	es  *elasticsearch.Client
	cfg *config.Config
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

	return &Client{es: es, cfg: cfg}, nil
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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/es/... -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/es/
git commit -m "feat: add ES client wrapper with request execution"
```

---

### Task 2.3: ES Data Fetchers (Indices, Nodes, Shards, Aliases)

**Files:**
- Create: `internal/es/cluster.go`
- Create: `internal/es/cluster_test.go`

**Step 1: Write failing test**

Create `internal/es/cluster_test.go`:
```go
package es

import (
	"encoding/json"
	"testing"
)

func TestParseIndicesResponse(t *testing.T) {
	raw := `[
		{"index":"test-1","health":"green","docs.count":"100","store.size":"1mb","pri":"1","rep":"1"},
		{"index":"test-2","health":"yellow","docs.count":"200","store.size":"2mb","pri":"2","rep":"0"}
	]`

	var indices []IndexInfo
	if err := json.Unmarshal([]byte(raw), &indices); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(indices) != 2 {
		t.Fatalf("got %d indices, want 2", len(indices))
	}

	if indices[0].Name != "test-1" {
		t.Errorf("indices[0].Name = %q, want %q", indices[0].Name, "test-1")
	}
	if indices[0].Health != "green" {
		t.Errorf("indices[0].Health = %q, want %q", indices[0].Health, "green")
	}
}

func TestParseNodesResponse(t *testing.T) {
	raw := `[
		{"name":"node-1","ip":"10.0.0.1","node.role":"dim"},
		{"name":"node-2","ip":"10.0.0.2","node.role":"m"}
	]`

	var nodes []NodeInfo
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	if nodes[0].Name != "node-1" {
		t.Errorf("nodes[0].Name = %q, want %q", nodes[0].Name, "node-1")
	}
}

func TestParseShardsResponse(t *testing.T) {
	raw := `[
		{"index":"test-1","shard":"0","prirep":"p","state":"STARTED","node":"node-1"},
		{"index":"test-1","shard":"0","prirep":"r","state":"STARTED","node":"node-2"}
	]`

	var shards []ShardInfo
	if err := json.Unmarshal([]byte(raw), &shards); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(shards) != 2 {
		t.Fatalf("got %d shards, want 2", len(shards))
	}

	if shards[0].Index != "test-1" {
		t.Errorf("shards[0].Index = %q, want %q", shards[0].Index, "test-1")
	}
	if shards[0].Primary != true {
		t.Error("shards[0] should be primary")
	}
	if shards[1].Primary != false {
		t.Error("shards[1] should be replica")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/es/... -v
```
Expected: FAIL - types not found

**Step 3: Write implementation**

Create `internal/es/cluster.go`:
```go
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

	// Fetch indices
	res, err := c.es.Cat.Indices(
		c.es.Cat.Indices.WithContext(ctx),
		c.es.Cat.Indices.WithFormat("json"),
		c.es.Cat.Indices.WithH("index", "health", "docs.count", "store.size", "pri", "rep"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching indices: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &state.Indices); err != nil {
		return nil, fmt.Errorf("parsing indices: %w", err)
	}

	// Fetch nodes
	res, err = c.es.Cat.Nodes(
		c.es.Cat.Nodes.WithContext(ctx),
		c.es.Cat.Nodes.WithFormat("json"),
		c.es.Cat.Nodes.WithH("name", "ip", "node.role"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching nodes: %w", err)
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &state.Nodes); err != nil {
		return nil, fmt.Errorf("parsing nodes: %w", err)
	}

	// Fetch shards
	res, err = c.es.Cat.Shards(
		c.es.Cat.Shards.WithContext(ctx),
		c.es.Cat.Shards.WithFormat("json"),
		c.es.Cat.Shards.WithH("index", "shard", "prirep", "state", "node"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching shards: %w", err)
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &state.Shards); err != nil {
		return nil, fmt.Errorf("parsing shards: %w", err)
	}

	// Fetch aliases
	res, err = c.es.Cat.Aliases(
		c.es.Cat.Aliases.WithContext(ctx),
		c.es.Cat.Aliases.WithFormat("json"),
		c.es.Cat.Aliases.WithH("alias", "index"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching aliases: %w", err)
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &state.Aliases); err != nil {
		return nil, fmt.Errorf("parsing aliases: %w", err)
	}

	return state, nil
}

// GetAliasesForIndex returns all aliases for a given index
func (s *ClusterState) GetAliasesForIndex(index string) []string {
	var aliases []string
	for _, a := range s.Aliases {
		if a.Index == index {
			aliases = append(aliases, a.Alias)
		}
	}
	return aliases
}

// GetShardsForIndexAndNode returns shards for a given index on a given node
func (s *ClusterState) GetShardsForIndexAndNode(index, node string) []ShardInfo {
	var shards []ShardInfo
	for _, sh := range s.Shards {
		if sh.Index == index && sh.Node == node {
			shards = append(shards, sh)
		}
	}
	return shards
}

// UniqueAliases returns deduplicated list of all aliases
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
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/es/... -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/es/cluster.go internal/es/cluster_test.go
git commit -m "feat: add cluster state fetchers for indices, nodes, shards, aliases"
```

---

### Task 2.4: Wire Up Connection to Main

**Files:**
- Modify: `main.go`
- Modify: `internal/ui/model.go`

**Step 1: Update main.go with CLI flag**

Replace `main.go`:
```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
	"github.com/labtiva/stoptail/internal/ui"
)

func main() {
	urlFlag := flag.String("url", "", "Elasticsearch URL (e.g., https://user:pass@localhost:9200)")
	flag.Parse()

	cfg, err := config.Load(*urlFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	client, err := es.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(client, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Update model to accept client and config**

Replace `internal/ui/model.go`:
```go
package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
)

const (
	TabOverview = iota
	TabWorkbench
)

type Model struct {
	client    *es.Client
	cfg       *config.Config
	cluster   *es.ClusterState
	activeTab int
	width     int
	height    int
	connected bool
	err       error
	quitting  bool
}

type connectedMsg struct{ state *es.ClusterState }
type errMsg struct{ err error }

func New(client *es.Client, cfg *config.Config) Model {
	return Model{
		client:    client,
		cfg:       cfg,
		activeTab: TabOverview,
	}
}

func (m Model) Init() tea.Cmd {
	return m.connect()
}

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := m.client.Ping(ctx); err != nil {
			return errMsg{err}
		}
		state, err := m.client.FetchClusterState(ctx)
		if err != nil {
			return errMsg{err}
		}
		return connectedMsg{state}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connectedMsg:
		m.connected = true
		m.cluster = msg.state
		m.err = nil
	case errMsg:
		m.err = msg.err
		m.connected = false
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
		case "r":
			return m, m.connect()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	status := "connecting..."
	if m.connected {
		status = "connected"
	}
	if m.err != nil {
		status = "error"
	}
	headerText := fmt.Sprintf("stoptail · %s [%s]", m.cfg.MaskedURL(), status)
	header := HeaderStyle.Width(m.width).Render(headerText)

	// Tabs
	var tabs string
	if m.activeTab == TabOverview {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			ActiveTabStyle.Render("Overview"),
			InactiveTabStyle.Render("Workbench"),
		)
	} else {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			InactiveTabStyle.Render("Overview"),
			ActiveTabStyle.Render("Workbench"),
		)
	}

	// Content
	contentHeight := m.height - 4
	var content string
	if m.err != nil {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Foreground(ColorRed).
			Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("Connection error:\n%v\n\nPress 'r' to retry", m.err))
	} else if !m.connected {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Connecting...")
	} else {
		info := fmt.Sprintf("Connected!\n\nIndices: %d\nNodes: %d\nShards: %d\nAliases: %d",
			len(m.cluster.Indices),
			len(m.cluster.Nodes),
			len(m.cluster.Shards),
			len(m.cluster.Aliases))
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(info)
	}

	// Status bar
	status = "q: quit  Tab: switch view  r: refresh"
	statusBar := StatusBarStyle.Width(m.width).Render(status)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}
```

**Step 3: Verify it builds**

Run:
```bash
go build -o stoptail .
```
Expected: Builds without errors

**Step 4: Commit**

```bash
git add main.go internal/ui/model.go
git commit -m "feat: wire up ES client to main with connection on startup"
```

---

## Phase 3: Overview Tab - Shard Grid

### Task 3.1: Overview Model Structure

**Files:**
- Create: `internal/ui/overview.go`

**Step 1: Create overview model**

Create `internal/ui/overview.go`:
```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type OverviewModel struct {
	cluster       *es.ClusterState
	filter        textinput.Model
	filterActive  bool
	aliasFilters  map[string]bool
	scrollX       int
	scrollY       int
	selectedIndex int
	width         int
	height        int
}

func NewOverview() OverviewModel {
	ti := textinput.New()
	ti.Placeholder = "Filter indices..."
	ti.CharLimit = 50

	return OverviewModel{
		filter:       ti,
		aliasFilters: make(map[string]bool),
	}
}

func (m *OverviewModel) SetCluster(cluster *es.ClusterState) {
	m.cluster = cluster
}

func (m *OverviewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m OverviewModel) Update(msg tea.Msg) (OverviewModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "esc", "enter":
				m.filterActive = false
				m.filter.Blur()
				return m, nil
			}
			m.filter, cmd = m.filter.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "/":
			m.filterActive = true
			m.filter.Focus()
			return m, textinput.Blink
		case "esc":
			m.filter.SetValue("")
			m.aliasFilters = make(map[string]bool)
		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			m.scrollY++
		case "left", "h":
			if m.scrollX > 0 {
				m.scrollX--
			}
		case "right", "l":
			m.scrollX++
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.cluster != nil {
				aliases := m.cluster.UniqueAliases()
				idx := int(msg.String()[0] - '1')
				if idx < len(aliases) {
					alias := aliases[idx]
					m.aliasFilters[alias] = !m.aliasFilters[alias]
				}
			}
		}
	}
	return m, nil
}

func (m OverviewModel) filteredIndices() []es.IndexInfo {
	if m.cluster == nil {
		return nil
	}

	var filtered []es.IndexInfo
	filterText := strings.ToLower(m.filter.Value())

	for _, idx := range m.cluster.Indices {
		// Text filter
		if filterText != "" {
			match := false
			if strings.Contains(strings.ToLower(idx.Name), filterText) {
				match = true
			}
			// Wildcard support
			if strings.HasSuffix(filterText, "*") {
				prefix := strings.TrimSuffix(filterText, "*")
				if strings.HasPrefix(strings.ToLower(idx.Name), prefix) {
					match = true
				}
			}
			if !match {
				continue
			}
		}

		// Alias filter
		if len(m.aliasFilters) > 0 {
			hasActiveAlias := false
			for _, active := range m.aliasFilters {
				if active {
					hasActiveAlias = true
					break
				}
			}
			if hasActiveAlias {
				indexAliases := m.cluster.GetAliasesForIndex(idx.Name)
				match := false
				for _, a := range indexAliases {
					if m.aliasFilters[a] {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
		}

		filtered = append(filtered, idx)
	}
	return filtered
}

func (m OverviewModel) SelectedIndex() string {
	indices := m.filteredIndices()
	if m.selectedIndex >= 0 && m.selectedIndex < len(indices) {
		return indices[m.selectedIndex].Name
	}
	return ""
}

func (m OverviewModel) View() string {
	if m.cluster == nil {
		return "Loading cluster state..."
	}

	var b strings.Builder

	// Filter bar
	filterStyle := lipgloss.NewStyle().Padding(0, 1)
	if m.filterActive {
		b.WriteString(filterStyle.Render("Filter: " + m.filter.View()))
	} else if m.filter.Value() != "" {
		b.WriteString(filterStyle.Render("Filter: " + m.filter.Value() + " (/ to edit, Esc to clear)"))
	} else {
		b.WriteString(filterStyle.Render("/ to filter"))
	}

	// Alias toggles
	aliases := m.cluster.UniqueAliases()
	if len(aliases) > 0 {
		b.WriteString("  Aliases: ")
		for i, alias := range aliases {
			if i >= 9 {
				break
			}
			style := lipgloss.NewStyle().Padding(0, 1)
			if m.aliasFilters[alias] {
				style = style.Background(ColorBlue).Foreground(ColorWhite)
			} else {
				style = style.Foreground(ColorGray)
			}
			b.WriteString(style.Render(string('1'+i) + ":" + alias))
			b.WriteString(" ")
		}
	}
	b.WriteString("\n\n")

	// Shard grid
	b.WriteString(m.renderGrid())

	return b.String()
}

func (m OverviewModel) renderGrid() string {
	if m.cluster == nil || len(m.cluster.Nodes) == 0 {
		return "No nodes found"
	}

	indices := m.filteredIndices()
	if len(indices) == 0 {
		return "No indices match filter"
	}

	nodes := m.cluster.Nodes

	// Calculate column widths
	nodeColWidth := 15
	indexColWidth := 20
	shardBoxWidth := 4

	var b strings.Builder

	// Header row - index names
	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
			healthColor := ColorGreen
			switch idx.Health {
			case "yellow":
				healthColor = ColorYellow
			case "red":
				healthColor = ColorRed
			}
			nameStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(healthColor).
				Bold(true)
			b.WriteString(nameStyle.Render(truncate(idx.Name, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	// Header row - stats
	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
			statsStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorGray)
			stats := idx.StoreSize + " · " + idx.DocsCount
			b.WriteString(statsStyle.Render(truncate(stats, indexColWidth-2)))
		}
	}
	b.WriteString("\n")

	// Header row - aliases
	b.WriteString(strings.Repeat(" ", nodeColWidth+2))
	for i, idx := range indices {
		if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
			aliases := m.cluster.GetAliasesForIndex(idx.Name)
			aliasStyle := lipgloss.NewStyle().
				Width(indexColWidth).
				Foreground(ColorBlue)
			aliasText := ""
			if len(aliases) > 0 {
				aliasText = "[" + strings.Join(aliases, ",") + "]"
			}
			b.WriteString(aliasStyle.Render(truncate(aliasText, indexColWidth-2)))
		}
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	// Node rows
	visibleNodes := nodes
	if m.scrollY < len(nodes) {
		visibleNodes = nodes[m.scrollY:]
	}
	maxRows := (m.height - 8) / 2
	if maxRows > len(visibleNodes) {
		maxRows = len(visibleNodes)
	}

	for _, node := range visibleNodes[:maxRows] {
		// Node name
		nodeStyle := lipgloss.NewStyle().Width(nodeColWidth)
		b.WriteString(nodeStyle.Render(truncate(node.Name, nodeColWidth-2)))
		b.WriteString("│ ")

		// Shards for each index
		for i, idx := range indices {
			if i >= m.scrollX && i < m.scrollX+((m.width-nodeColWidth)/indexColWidth) {
				shards := m.cluster.GetShardsForIndexAndNode(idx.Name, node.Name)
				shardStr := m.renderShardBoxes(shards, indexColWidth)
				b.WriteString(shardStr)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m OverviewModel) renderShardBoxes(shards []es.ShardInfo, width int) string {
	if len(shards) == 0 {
		return lipgloss.NewStyle().Width(width).Render("")
	}

	var boxes []string
	for _, sh := range shards {
		var style lipgloss.Style
		if sh.Primary {
			style = lipgloss.NewStyle().
				Background(ColorGreen).
				Foreground(ColorWhite).
				Padding(0, 0).
				Width(3)
		} else {
			style = lipgloss.NewStyle().
				Background(ColorBlue).
				Foreground(ColorWhite).
				Padding(0, 0).
				Width(3)
		}

		// Color by state
		switch sh.State {
		case "RELOCATING":
			style = style.Background(ColorYellow)
		case "UNASSIGNED":
			style = style.Background(ColorRed)
		}

		boxes = append(boxes, style.Render(sh.Shard))
	}

	result := strings.Join(boxes, " ")
	return lipgloss.NewStyle().Width(width).Render(result)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
```

**Step 2: Verify it builds**

Run:
```bash
go build -o stoptail .
```
Expected: Builds without errors

**Step 3: Commit**

```bash
git add internal/ui/overview.go
git commit -m "feat: add overview model with shard grid rendering"
```

---

### Task 3.2: Integrate Overview into Main Model

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Update model.go to use overview**

Replace `internal/ui/model.go`:
```go
package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
)

const (
	TabOverview = iota
	TabWorkbench
)

type Model struct {
	client    *es.Client
	cfg       *config.Config
	cluster   *es.ClusterState
	overview  OverviewModel
	activeTab int
	width     int
	height    int
	connected bool
	err       error
	quitting  bool
}

type connectedMsg struct{ state *es.ClusterState }
type errMsg struct{ err error }

func New(client *es.Client, cfg *config.Config) Model {
	return Model{
		client:    client,
		cfg:       cfg,
		overview:  NewOverview(),
		activeTab: TabOverview,
	}
}

func (m Model) Init() tea.Cmd {
	return m.connect()
}

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := m.client.Ping(ctx); err != nil {
			return errMsg{err}
		}
		state, err := m.client.FetchClusterState(ctx)
		if err != nil {
			return errMsg{err}
		}
		return connectedMsg{state}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case connectedMsg:
		m.connected = true
		m.cluster = msg.state
		m.overview.SetCluster(msg.state)
		m.err = nil
	case errMsg:
		m.err = msg.err
		m.connected = false
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.activeTab == TabOverview && m.overview.filterActive {
				// Let overview handle it
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case "tab":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.activeTab = (m.activeTab + 1) % 2
				return m, nil
			}
		case "r":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				return m, m.connect()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overview.SetSize(msg.Width, msg.Height-4)
	}

	// Delegate to active tab
	if m.activeTab == TabOverview && m.connected {
		m.overview, cmd = m.overview.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	status := "connecting..."
	if m.connected {
		status = "connected"
	}
	if m.err != nil {
		status = "error"
	}
	headerText := fmt.Sprintf("stoptail · %s [%s]", m.cfg.MaskedURL(), status)
	header := HeaderStyle.Width(m.width).Render(headerText)

	// Tabs
	var tabs string
	if m.activeTab == TabOverview {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			ActiveTabStyle.Render("Overview"),
			InactiveTabStyle.Render("Workbench"),
		)
	} else {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			InactiveTabStyle.Render("Overview"),
			ActiveTabStyle.Render("Workbench"),
		)
	}

	// Content
	contentHeight := m.height - 4
	var content string
	if m.err != nil {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Foreground(ColorRed).
			Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("Connection error:\n%v\n\nPress 'r' to retry", m.err))
	} else if !m.connected {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Connecting...")
	} else if m.activeTab == TabOverview {
		content = m.overview.View()
	} else {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Workbench (coming soon)")
	}

	// Status bar
	statusText := "q: quit  Tab: switch view  r: refresh"
	if m.activeTab == TabOverview {
		statusText = "q: quit  Tab: switch  r: refresh  /: filter  ←→↑↓: scroll"
	}
	statusBar := StatusBarStyle.Width(m.width).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}
```

**Step 2: Build and verify**

Run:
```bash
go build -o stoptail .
```
Expected: Builds without errors

**Step 3: Commit**

```bash
git add internal/ui/model.go
git commit -m "feat: integrate overview tab with shard grid into main model"
```

---

## Phase 4: Workbench Tab - Request Tool

### Task 4.1: Workbench Model

**Files:**
- Create: `internal/ui/workbench.go`

**Step 1: Create workbench model**

Create `internal/ui/workbench.go`:
```go
package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/es"
)

type WorkbenchFocus int

const (
	FocusMethod WorkbenchFocus = iota
	FocusPath
	FocusBody
	FocusResponse
)

var methods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}

type WorkbenchModel struct {
	client       *es.Client
	methodIdx    int
	path         textinput.Model
	body         textarea.Model
	response     viewport.Model
	responseText string
	statusCode   int
	duration     string
	focus        WorkbenchFocus
	width        int
	height       int
	executing    bool
	err          error
}

type executeResultMsg struct {
	result es.RequestResult
}

func NewWorkbench() WorkbenchModel {
	path := textinput.New()
	path.Placeholder = "/_search"
	path.CharLimit = 200
	path.Width = 40

	body := textarea.New()
	body.Placeholder = `{
  "query": {
    "match_all": {}
  }
}`
	body.CharLimit = 50000
	body.ShowLineNumbers = false

	vp := viewport.New(40, 10)

	return WorkbenchModel{
		methodIdx: 0,
		path:      path,
		body:      body,
		response:  vp,
		focus:     FocusPath,
	}
}

func (m *WorkbenchModel) SetClient(client *es.Client) {
	m.client = client
}

func (m *WorkbenchModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Split panes
	paneWidth := (width - 3) / 2 // -3 for divider and padding
	bodyHeight := height - 6      // -6 for method/path row and status

	m.path.Width = paneWidth - 10 // -10 for method selector
	m.body.SetWidth(paneWidth)
	m.body.SetHeight(bodyHeight)
	m.response.Width = paneWidth
	m.response.Height = bodyHeight
}

func (m *WorkbenchModel) Prefill(index string) {
	m.methodIdx = 0 // GET
	m.path.SetValue("/" + index + "/_search")
	m.body.SetValue("{}")
}

func (m *WorkbenchModel) Focus() {
	m.path.Focus()
	m.focus = FocusPath
}

func (m *WorkbenchModel) Blur() {
	m.path.Blur()
	m.body.Blur()
}

func (m WorkbenchModel) isValidJSON() bool {
	val := m.body.Value()
	if val == "" {
		return true
	}
	return json.Valid([]byte(val))
}

func (m WorkbenchModel) Update(msg tea.Msg) (WorkbenchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case executeResultMsg:
		m.executing = false
		if msg.result.Error != nil {
			m.err = msg.result.Error
			m.responseText = fmt.Sprintf("Error: %v", msg.result.Error)
		} else {
			m.err = nil
			m.statusCode = msg.result.StatusCode
			m.duration = msg.result.Duration.String()
			m.responseText = highlightJSON(msg.result.Body)
		}
		m.response.SetContent(m.responseText)
		m.response.GotoTop()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+m":
			m.methodIdx = (m.methodIdx + 1) % len(methods)
			return m, nil
		case "ctrl+enter":
			if m.client != nil && !m.executing {
				m.executing = true
				return m, m.execute()
			}
		case "ctrl+l":
			m.body.SetValue("")
			return m, nil
		case "ctrl+p":
			m.prettyPrintBody()
			return m, nil
		case "tab":
			m.cycleFocus()
			return m, nil
		case "esc":
			m.path.Blur()
			m.body.Blur()
			return m, nil
		}

		// Delegate to focused component
		switch m.focus {
		case FocusPath:
			m.path, cmd = m.path.Update(msg)
			cmds = append(cmds, cmd)
		case FocusBody:
			m.body, cmd = m.body.Update(msg)
			cmds = append(cmds, cmd)
		case FocusResponse:
			m.response, cmd = m.response.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *WorkbenchModel) cycleFocus() {
	m.path.Blur()
	m.body.Blur()

	m.focus = (m.focus + 1) % 4
	switch m.focus {
	case FocusMethod:
		// No component to focus
	case FocusPath:
		m.path.Focus()
	case FocusBody:
		m.body.Focus()
	case FocusResponse:
		// Viewport doesn't need focus call
	}
}

func (m *WorkbenchModel) prettyPrintBody() {
	val := m.body.Value()
	if val == "" {
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(val), "", "  "); err == nil {
		m.body.SetValue(pretty.String())
	}
}

func (m WorkbenchModel) execute() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		method := methods[m.methodIdx]
		path := m.path.Value()
		body := m.body.Value()
		result := m.client.Request(ctx, method, path, body)
		return executeResultMsg{result}
	}
}

func (m WorkbenchModel) View() string {
	// Method + Path row
	methodStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)
	if m.focus == FocusMethod {
		methodStyle = methodStyle.Background(ColorBlue).Foreground(ColorWhite)
	}
	methodView := methodStyle.Render(methods[m.methodIdx] + " ▼")

	pathStyle := lipgloss.NewStyle()
	if m.focus == FocusPath {
		pathStyle = pathStyle.Border(lipgloss.RoundedBorder()).BorderForeground(ColorBlue)
	}
	pathView := pathStyle.Render(m.path.View())

	topRow := lipgloss.JoinHorizontal(lipgloss.Center, methodView, " ", pathView)

	// Split panes
	paneWidth := (m.width - 3) / 2

	// Left pane - body
	bodyBorder := lipgloss.RoundedBorder()
	bodyBorderColor := ColorGray
	if m.focus == FocusBody {
		bodyBorderColor = ColorBlue
	}
	bodyPane := lipgloss.NewStyle().
		Border(bodyBorder).
		BorderForeground(bodyBorderColor).
		Width(paneWidth).
		Height(m.height - 6).
		Render(m.body.View())

	// Right pane - response
	responseBorder := lipgloss.RoundedBorder()
	responseBorderColor := ColorGray
	if m.focus == FocusResponse {
		responseBorderColor = ColorBlue
	}

	responseHeader := "Response"
	if m.statusCode > 0 {
		statusColor := ColorGreen
		if m.statusCode >= 400 {
			statusColor = ColorRed
		}
		responseHeader = fmt.Sprintf("Response  %s %s",
			lipgloss.NewStyle().Foreground(statusColor).Render(fmt.Sprintf("%d", m.statusCode)),
			lipgloss.NewStyle().Foreground(ColorGray).Render(m.duration))
	}
	if m.executing {
		responseHeader = "Executing..."
	}

	responsePane := lipgloss.NewStyle().
		Border(responseBorder).
		BorderForeground(responseBorderColor).
		Width(paneWidth).
		Height(m.height - 6).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render(responseHeader),
			m.response.View()))

	panes := lipgloss.JoinHorizontal(lipgloss.Top, bodyPane, " ", responsePane)

	// Status bar
	validIndicator := lipgloss.NewStyle().Foreground(ColorGreen).Render("✓ Valid JSON")
	if !m.isValidJSON() {
		validIndicator = lipgloss.NewStyle().Foreground(ColorRed).Render("✗ Invalid JSON")
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Center,
		validIndicator,
		strings.Repeat(" ", m.width-50),
		HelpStyle.Render("Ctrl+Enter: Execute  Ctrl+M: Method  Ctrl+P: Pretty"))

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "", panes, statusBar)
}

func highlightJSON(input string) string {
	var buf bytes.Buffer
	err := quick.Highlight(&buf, input, "json", "terminal256", "monokai")
	if err != nil {
		return input
	}
	return buf.String()
}
```

**Step 2: Build and verify**

Run:
```bash
go build -o stoptail .
```
Expected: Builds without errors

**Step 3: Commit**

```bash
git add internal/ui/workbench.go
git commit -m "feat: add workbench model with split pane request editor"
```

---

### Task 4.2: Integrate Workbench into Main Model

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Update model.go to use workbench**

Replace `internal/ui/model.go`:
```go
package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
)

const (
	TabOverview = iota
	TabWorkbench
)

type Model struct {
	client    *es.Client
	cfg       *config.Config
	cluster   *es.ClusterState
	overview  OverviewModel
	workbench WorkbenchModel
	activeTab int
	width     int
	height    int
	connected bool
	err       error
	quitting  bool
}

type connectedMsg struct{ state *es.ClusterState }
type errMsg struct{ err error }

func New(client *es.Client, cfg *config.Config) Model {
	wb := NewWorkbench()
	wb.SetClient(client)

	return Model{
		client:    client,
		cfg:       cfg,
		overview:  NewOverview(),
		workbench: wb,
		activeTab: TabOverview,
	}
}

func (m Model) Init() tea.Cmd {
	return m.connect()
}

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := m.client.Ping(ctx); err != nil {
			return errMsg{err}
		}
		state, err := m.client.FetchClusterState(ctx)
		if err != nil {
			return errMsg{err}
		}
		return connectedMsg{state}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case connectedMsg:
		m.connected = true
		m.cluster = msg.state
		m.overview.SetCluster(msg.state)
		m.err = nil
	case errMsg:
		m.err = msg.err
		m.connected = false
	case executeResultMsg:
		m.workbench, cmd = m.workbench.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			// Only quit if not in a focused input
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.quitting = true
				return m, tea.Quit
			}
			if m.activeTab == TabWorkbench && m.workbench.focus != FocusPath && m.workbench.focus != FocusBody {
				m.quitting = true
				return m, tea.Quit
			}
		case "tab":
			// Global tab to switch views, unless in focused input
			if m.activeTab == TabOverview && !m.overview.filterActive {
				m.activeTab = TabWorkbench
				m.workbench.Focus()
				return m, nil
			}
			if m.activeTab == TabWorkbench {
				// Tab cycles through workbench components, not views
			}
		case "shift+tab":
			// Switch back to overview
			if m.activeTab == TabWorkbench {
				m.activeTab = TabOverview
				m.workbench.Blur()
				return m, nil
			}
		case "r":
			if m.activeTab == TabOverview && !m.overview.filterActive {
				return m, m.connect()
			}
		case "enter":
			// From overview, enter on index switches to workbench
			if m.activeTab == TabOverview && !m.overview.filterActive && m.overview.filterActive == false {
				if idx := m.overview.SelectedIndex(); idx != "" {
					m.workbench.Prefill(idx)
					m.activeTab = TabWorkbench
					m.workbench.Focus()
					return m, nil
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overview.SetSize(msg.Width, msg.Height-4)
		m.workbench.SetSize(msg.Width, msg.Height-4)
	}

	// Delegate to active tab
	if m.connected {
		if m.activeTab == TabOverview {
			m.overview, cmd = m.overview.Update(msg)
		} else {
			m.workbench, cmd = m.workbench.Update(msg)
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	status := "connecting..."
	if m.connected {
		status = "connected"
	}
	if m.err != nil {
		status = "error"
	}
	headerText := fmt.Sprintf("stoptail · %s [%s]", m.cfg.MaskedURL(), status)
	header := HeaderStyle.Width(m.width).Render(headerText)

	// Tabs
	var tabs string
	if m.activeTab == TabOverview {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			ActiveTabStyle.Render("Overview"),
			InactiveTabStyle.Render("Workbench"),
		)
	} else {
		tabs = lipgloss.JoinHorizontal(lipgloss.Top,
			InactiveTabStyle.Render("Overview"),
			ActiveTabStyle.Render("Workbench"),
		)
	}

	// Content
	contentHeight := m.height - 4
	var content string
	if m.err != nil {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Foreground(ColorRed).
			Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("Connection error:\n%v\n\nPress 'r' to retry", m.err))
	} else if !m.connected {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Connecting...")
	} else if m.activeTab == TabOverview {
		content = m.overview.View()
	} else {
		content = m.workbench.View()
	}

	// Status bar
	statusText := "q: quit  Tab: switch view  r: refresh"
	if m.activeTab == TabOverview {
		statusText = "q: quit  Tab: workbench  r: refresh  /: filter  ↑↓←→: scroll  Enter: open index"
	} else {
		statusText = "Shift+Tab: overview  Tab: cycle focus  Ctrl+Enter: execute"
	}
	statusBar := StatusBarStyle.Width(m.width).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, statusBar)
}
```

**Step 2: Build and verify**

Run:
```bash
go build -o stoptail .
```
Expected: Builds without errors

**Step 3: Commit**

```bash
git add internal/ui/model.go
git commit -m "feat: integrate workbench tab with index selection from overview"
```

---

## Phase 5: Polish

### Task 5.1: Help Overlay

**Files:**
- Create: `internal/ui/help.go`
- Modify: `internal/ui/model.go`

**Step 1: Create help view**

Create `internal/ui/help.go`:
```go
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func renderHelp(width, height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorWhite).
		MarginBottom(1).
		Render("stoptail - Elasticsearch TUI")

	sections := []struct {
		header string
		keys   [][]string
	}{
		{
			header: "Global",
			keys: [][]string{
				{"Tab / Shift+Tab", "Switch views"},
				{"q / Ctrl+C", "Quit"},
				{"?", "Toggle help"},
				{"r", "Refresh data"},
			},
		},
		{
			header: "Overview",
			keys: [][]string{
				{"/", "Focus filter"},
				{"Esc", "Clear filter"},
				{"↑↓←→", "Navigate grid"},
				{"Enter", "Open index in Workbench"},
				{"1-9", "Toggle alias filters"},
			},
		},
		{
			header: "Workbench",
			keys: [][]string{
				{"Tab", "Cycle focus"},
				{"Ctrl+M", "Cycle HTTP method"},
				{"Ctrl+Enter", "Execute request"},
				{"Ctrl+L", "Clear body"},
				{"Ctrl+P", "Pretty-print JSON"},
			},
		},
	}

	keyStyle := lipgloss.NewStyle().Foreground(ColorBlue).Width(20)
	descStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorYellow).MarginTop(1)

	var content string
	content += title + "\n\n"

	for _, section := range sections {
		content += headerStyle.Render(section.header) + "\n"
		for _, kv := range section.keys {
			content += keyStyle.Render(kv[0]) + descStyle.Render(kv[1]) + "\n"
		}
	}

	content += "\n" + lipgloss.NewStyle().Foreground(ColorGray).Render("Press ? to close")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(50)

	box := boxStyle.Render(content)

	// Center the box
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
```

**Step 2: Update model.go with help toggle**

In `internal/ui/model.go`, add `showHelp bool` to Model struct and handle `?` key:

Add to Model struct:
```go
showHelp  bool
```

Add to Update switch on tea.KeyMsg:
```go
case "?":
    m.showHelp = !m.showHelp
    return m, nil
```

Add to View before return:
```go
if m.showHelp {
    return renderHelp(m.width, m.height)
}
```

**Step 3: Build and verify**

Run:
```bash
go build -o stoptail .
```
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/ui/help.go internal/ui/model.go
git commit -m "feat: add help overlay with ? toggle"
```

---

### Task 5.2: Final Integration Test

**Step 1: Run the complete app**

Run:
```bash
./stoptail --url "http://localhost:9200"
```

**Step 2: Verify functionality**

Test checklist:
- [ ] App starts and connects
- [ ] Overview shows shard grid with indices, nodes, shards
- [ ] Filter with `/` works
- [ ] Alias number toggles work
- [ ] Tab switches to Workbench
- [ ] Workbench method selector cycles with Ctrl+M
- [ ] Workbench executes requests with Ctrl+Enter
- [ ] Response is syntax highlighted
- [ ] Help overlay shows with `?`
- [ ] Quit with `q` works

**Step 3: Final commit**

```bash
git add -A
git commit -m "chore: final polish and integration"
```

---

## Summary

This plan implements stoptail in 5 phases:

1. **Scaffolding** - Go module, basic bubbletea shell
2. **ES Client** - URL parser, client wrapper, cluster data fetchers
3. **Overview** - Shard grid with filters
4. **Workbench** - Split pane request editor
5. **Polish** - Help overlay, final integration

Each task is TDD-driven with explicit file paths, complete code, and verification steps.
