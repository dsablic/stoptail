# AWS OpenSearch Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add AWS OpenSearch support with automatic SigV4 authentication detected from URL patterns.

**Architecture:** Auto-detect AWS endpoints from URL domain (*.es.amazonaws.com, *.aoss.amazonaws.com), extract region from URL, use AWS SDK v2 default credential chain with optional profile override, implement custom http.RoundTripper for SigV4 request signing.

**Tech Stack:** aws-sdk-go-v2/config, aws-sdk-go-v2/credentials, aws-sdk-go-v2/aws/signer/v4

---

### Task 1: Add AWS SDK Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add AWS SDK v2 dependencies**

Run:
```bash
go get github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/credentials github.com/aws/aws-sdk-go-v2/aws
```

**Step 2: Verify dependencies added**

Run: `grep aws-sdk go.mod`
Expected: Three aws-sdk-go-v2 packages listed

**Step 3: Tidy modules**

Run: `go mod tidy`

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add AWS SDK v2 for OpenSearch SigV4 signing"
```

---

### Task 2: Add AWS URL Parsing

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestParseAWSEndpoint(t *testing.T) {
	tests := []struct {
		host    string
		region  string
		service string
		ok      bool
	}{
		{"search-my-domain.us-east-1.es.amazonaws.com", "us-east-1", "es", true},
		{"vpc-my-domain.eu-west-1.es.amazonaws.com", "eu-west-1", "es", true},
		{"search-test.ap-southeast-2.es.amazonaws.com", "ap-southeast-2", "es", true},
		{"abc123xyz.us-west-2.aoss.amazonaws.com", "us-west-2", "aoss", true},
		{"collection.eu-central-1.aoss.amazonaws.com", "eu-central-1", "aoss", true},
		{"localhost:9200", "", "", false},
		{"elasticsearch.example.com", "", "", false},
		{"es.amazonaws.com", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			region, service, ok := parseAWSEndpoint(tt.host)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if region != tt.region {
				t.Errorf("region = %q, want %q", region, tt.region)
			}
			if service != tt.service {
				t.Errorf("service = %q, want %q", service, tt.service)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestParseAWSEndpoint -v`
Expected: FAIL with "undefined: parseAWSEndpoint"

**Step 3: Write parseAWSEndpoint function**

Add to `internal/config/config.go` before the `ParseURL` function:

```go
func parseAWSEndpoint(host string) (region, service string, ok bool) {
	host = strings.ToLower(host)

	if strings.HasSuffix(host, ".es.amazonaws.com") {
		parts := strings.Split(host, ".")
		if len(parts) >= 5 {
			return parts[len(parts)-4], "es", true
		}
	}

	if strings.HasSuffix(host, ".aoss.amazonaws.com") {
		parts := strings.Split(host, ".")
		if len(parts) >= 5 {
			return parts[len(parts)-4], "aoss", true
		}
	}

	return "", "", false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestParseAWSEndpoint -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add AWS endpoint URL parsing"
```

---

### Task 3: Extend Config Struct for AWS

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestParseURL_AWS(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantHost   string
		wantRegion string
		wantSvc    string
	}{
		{
			name:       "managed opensearch",
			url:        "https://search-my-domain.us-east-1.es.amazonaws.com",
			wantHost:   "https://search-my-domain.us-east-1.es.amazonaws.com",
			wantRegion: "us-east-1",
			wantSvc:    "es",
		},
		{
			name:       "opensearch serverless",
			url:        "https://abc123.us-west-2.aoss.amazonaws.com",
			wantHost:   "https://abc123.us-west-2.aoss.amazonaws.com",
			wantRegion: "us-west-2",
			wantSvc:    "aoss",
		},
		{
			name:       "vpc endpoint",
			url:        "https://vpc-my-domain.eu-west-1.es.amazonaws.com",
			wantHost:   "https://vpc-my-domain.eu-west-1.es.amazonaws.com",
			wantRegion: "eu-west-1",
			wantSvc:    "es",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.AWSRegion != tt.wantRegion {
				t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, tt.wantRegion)
			}
			if cfg.AWSService != tt.wantSvc {
				t.Errorf("AWSService = %q, want %q", cfg.AWSService, tt.wantSvc)
			}
		})
	}
}

func TestConfig_IsAWS(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{"aws config", &Config{AWSRegion: "us-east-1"}, true},
		{"standard config", &Config{Host: "http://localhost:9200"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsAWS(); got != tt.want {
				t.Errorf("IsAWS() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run "TestParseURL_AWS|TestConfig_IsAWS" -v`
Expected: FAIL with "cfg.AWSRegion undefined" or similar

**Step 3: Add AWS fields to Config struct**

Modify `internal/config/config.go`, update the Config struct:

```go
type Config struct {
	Host       string
	Username   string
	Password   string
	AWSRegion  string
	AWSService string
	AWSProfile string
}
```

**Step 4: Add IsAWS helper method**

Add to `internal/config/config.go`:

```go
func (c *Config) IsAWS() bool {
	return c.AWSRegion != ""
}
```

**Step 5: Update ParseURL to detect AWS**

Modify the `ParseURL` function in `internal/config/config.go`. After creating the cfg, add AWS detection:

```go
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

	if region, service, ok := parseAWSEndpoint(u.Host); ok {
		cfg.AWSRegion = region
		cfg.AWSService = service
	} else if u.User != nil {
		cfg.Username = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}

	return cfg, nil
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/config -run "TestParseURL_AWS|TestConfig_IsAWS" -v`
Expected: PASS

**Step 7: Run all config tests**

Run: `go test ./internal/config -v`
Expected: All tests PASS

**Step 8: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): extend Config struct with AWS fields"
```

---

### Task 4: Add AWSProfile to ClusterEntry

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestLoadClustersConfig_AWSProfile(t *testing.T) {
	yaml := `clusters:
  aws-prod:
    url: https://search-test.us-east-1.es.amazonaws.com
    aws_profile: production
  aws-default:
    url: https://search-dev.us-west-2.es.amazonaws.com
`
	var cfg ClustersConfig
	if err := yaml2.Unmarshal([]byte(yaml), &cfg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if cfg.Clusters["aws-prod"].AWSProfile != "production" {
		t.Errorf("AWSProfile = %q, want %q", cfg.Clusters["aws-prod"].AWSProfile, "production")
	}
	if cfg.Clusters["aws-default"].AWSProfile != "" {
		t.Errorf("AWSProfile = %q, want empty", cfg.Clusters["aws-default"].AWSProfile)
	}
}
```

Also add the import alias at the top of the test file:

```go
import (
	"testing"

	yaml2 "gopkg.in/yaml.v3"
)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestLoadClustersConfig_AWSProfile -v`
Expected: FAIL with "cfg.Clusters[...].AWSProfile undefined"

**Step 3: Add AWSProfile to ClusterEntry**

Modify `internal/config/config.go`, update ClusterEntry:

```go
type ClusterEntry struct {
	URL        string `yaml:"url"`
	URLCommand string `yaml:"url_command"`
	AWSProfile string `yaml:"aws_profile"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestLoadClustersConfig_AWSProfile -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add aws_profile field to ClusterEntry"
```

---

### Task 5: Update MaskedURL for AWS

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1: Write the failing test**

Add test case to `TestMaskedURL` in `internal/config/config_test.go`:

```go
{
	name: "AWS OpenSearch",
	cfg: &Config{
		Host:       "https://search-my-domain.us-east-1.es.amazonaws.com",
		AWSRegion:  "us-east-1",
		AWSService: "es",
	},
	want: "search-my-domain.us-east-1.es.amazonaws.com (AWS us-east-1)",
},
{
	name: "AWS OpenSearch Serverless",
	cfg: &Config{
		Host:       "https://abc123.us-west-2.aoss.amazonaws.com",
		AWSRegion:  "us-west-2",
		AWSService: "aoss",
	},
	want: "abc123.us-west-2.aoss.amazonaws.com (AWS us-west-2)",
},
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestMaskedURL -v`
Expected: FAIL with output mismatch

**Step 3: Update MaskedURL method**

Modify `MaskedURL` in `internal/config/config.go`:

```go
func (c *Config) MaskedURL() string {
	u, err := url.Parse(c.Host)
	if err != nil || u == nil {
		return c.Host
	}
	if c.IsAWS() {
		return fmt.Sprintf("%s (AWS %s)", strings.TrimPrefix(u.Host, "www."), c.AWSRegion)
	}
	if c.Username != "" {
		return fmt.Sprintf("%s:***@%s", c.Username, u.Host)
	}
	return u.Host
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestMaskedURL -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): update MaskedURL for AWS endpoints"
```

---

### Task 6: Create AWS SigV4 Transport

**Files:**
- Create: `internal/es/aws.go`

**Step 1: Create the aws.go file**

Create `internal/es/aws.go`:

```go
package es

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/labtiva/stoptail/internal/config"
)

type sigv4Transport struct {
	wrapped http.RoundTripper
	signer  *v4.Signer
	creds   aws.CredentialsProvider
	region  string
	service string
}

func (t *sigv4Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	creds, err := t.creds.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("retrieving AWS credentials: %w", err)
	}

	var body []byte
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	hash := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(hash[:])

	err = t.signer.SignHTTP(req.Context(), creds, req, payloadHash, t.service, t.region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("signing request: %w", err)
	}

	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return t.wrapped.RoundTrip(req)
}

func newAWSTransport(cfg *config.Config) (http.RoundTripper, error) {
	ctx := context.Background()

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.AWSRegion))

	if cfg.AWSProfile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.AWSProfile))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	if _, err := awsCfg.Credentials.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("AWS credentials not found: %w (configure AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY, ~/.aws/credentials, or IAM role)", err)
	}

	return &sigv4Transport{
		wrapped: http.DefaultTransport,
		signer:  v4.NewSigner(),
		creds:   awsCfg.Credentials,
		region:  cfg.AWSRegion,
		service: cfg.AWSService,
	}, nil
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/es`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/es/aws.go
git commit -m "feat(es): add AWS SigV4 signing transport"
```

---

### Task 7: Integrate AWS Transport into Client

**Files:**
- Modify: `internal/es/client.go`

**Step 1: Update NewClient to use AWS transport**

Replace the `NewClient` function in `internal/es/client.go`:

```go
func NewClient(cfg *config.Config) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Host},
	}

	var httpTransport http.RoundTripper = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	if cfg.IsAWS() {
		transport, err := newAWSTransport(cfg)
		if err != nil {
			return nil, err
		}
		esCfg.Transport = transport
		httpTransport = transport
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
```

**Step 2: Verify it compiles**

Run: `go build ./internal/es`
Expected: No errors

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/es/client.go
git commit -m "feat(es): integrate AWS transport into client"
```

---

### Task 8: Wire AWSProfile Through main.go

**Files:**
- Modify: `main.go`

**Step 1: Find where config is resolved from cluster entry**

Look for where `ResolveURL` is called and the config is created. We need to pass `AWSProfile` through.

**Step 2: Update config creation to include AWSProfile**

After `ParseURL` is called with the resolved URL, set the AWSProfile from the ClusterEntry if present.

Find the code that looks like:
```go
cfg, err := config.ParseURL(resolvedURL)
```

And after it, add:
```go
if entry, ok := clusters.Clusters[clusterName]; ok && entry.AWSProfile != "" {
    cfg.AWSProfile = entry.AWSProfile
}
```

The exact location will depend on main.go structure. The key is that after parsing the URL and before creating the ES client, we copy the AWSProfile from the cluster entry to the config.

**Step 3: Test manually with build**

Run: `go build .`
Expected: No errors

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire AWSProfile from cluster config to ES client"
```

---

### Task 9: Run Linters and Fix Issues

**Files:**
- Potentially any modified files

**Step 1: Run go vet**

Run: `go vet ./...`
Expected: No issues

**Step 2: Run staticcheck**

Run: `staticcheck ./...`
Expected: No issues (fix any that appear)

**Step 3: Commit fixes if any**

```bash
git add -A
git commit -m "fix: address linter warnings"
```

---

### Task 10: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Step 1: Update README.md**

Add after the "Multi-cluster Configuration" section:

```markdown
### AWS OpenSearch

stoptail automatically detects AWS OpenSearch endpoints and uses AWS SigV4 authentication:

```yaml
clusters:
  # AWS OpenSearch (managed) - auto-detected from URL
  aws-prod:
    url: https://search-mycluster.us-east-1.es.amazonaws.com

  # AWS OpenSearch Serverless - also auto-detected
  aws-serverless:
    url: https://abc123xyz.us-west-2.aoss.amazonaws.com

  # With specific AWS profile
  aws-staging:
    url: https://search-staging.eu-west-1.es.amazonaws.com
    aws_profile: staging-account
```

**Credentials** are resolved using the standard AWS SDK chain:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (for EC2/ECS/Lambda)
4. SSO credentials (`aws sso login`)

Use `aws_profile` to select a specific profile from `~/.aws/credentials` or `~/.aws/config`.
```

**Step 2: Update CLAUDE.md**

Update the config example in CLAUDE.md to include AWS:

```yaml
clusters:
  production:
    url: https://user:pass@es-prod:9200
  staging:
    url_command: "vault read -field=url secret/es-staging"
  # AWS OpenSearch (auto-detected from URL)
  aws-prod:
    url: https://search-mycluster.us-east-1.es.amazonaws.com
    aws_profile: production  # optional
```

**Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: add AWS OpenSearch configuration examples"
```

---

### Task 11: Final Verification

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 2: Run linters**

Run: `go vet ./... && staticcheck ./...`
Expected: No issues

**Step 3: Build and verify**

Run: `go build . && ./stoptail --help`
Expected: Builds successfully, help shows

**Step 4: Test render mode (if ES available)**

Run: `./stoptail --render overview --width 120 --height 40`
Expected: Renders without errors (or shows connection error if no ES)
