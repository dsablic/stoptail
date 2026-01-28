# AWS OpenSearch Support Design

## Overview

Add support for AWS OpenSearch (managed and serverless) using AWS SigV4 authentication.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| AWS detection | Auto-detect from URL | Zero config - AWS URLs are distinctive |
| Credentials | AWS SDK default chain + optional profile | Covers 90% of cases; profile override for multi-account |
| Service type | Auto-detect from URL | `*.es.amazonaws.com` vs `*.aoss.amazonaws.com` |
| Region | Auto-detect from URL | Always present in AWS OpenSearch URLs |
| AWS SDK | v2 | Actively developed, modular, better credential handling |

## Configuration

### Config Struct Changes

```go
type Config struct {
    Host       string
    Username   string
    Password   string
    AWSRegion  string  // auto-detected from URL
    AWSService string  // "es" or "aoss", auto-detected
    AWSProfile string  // optional, from cluster config
}

type ClusterEntry struct {
    URL        string `yaml:"url"`
    URLCommand string `yaml:"url_command"`
    AWSProfile string `yaml:"aws_profile"` // new, optional
}
```

### Example Config

```yaml
clusters:
  # Standard ES - works as before
  local:
    url: http://localhost:9200

  # AWS OpenSearch - auto-detected, default credentials
  aws-prod:
    url: https://search-mycluster.us-east-1.es.amazonaws.com

  # AWS with specific profile
  aws-staging:
    url: https://search-staging.us-west-2.es.amazonaws.com
    aws_profile: staging-account
```

## URL Parsing

```go
func parseAWSEndpoint(host string) (region, service string, ok bool) {
    host = strings.ToLower(host)

    // Managed OpenSearch: search-*.us-east-1.es.amazonaws.com
    // Also: vpc-*.us-east-1.es.amazonaws.com
    if strings.HasSuffix(host, ".es.amazonaws.com") {
        parts := strings.Split(host, ".")
        if len(parts) >= 5 {
            return parts[len(parts)-4], "es", true
        }
    }

    // OpenSearch Serverless: *.us-west-2.aoss.amazonaws.com
    if strings.HasSuffix(host, ".aoss.amazonaws.com") {
        parts := strings.Split(host, ".")
        if len(parts) >= 5 {
            return parts[len(parts)-4], "aoss", true
        }
    }

    return "", "", false
}
```

## Client Changes

### New Dependencies

```go
require (
    github.com/aws/aws-sdk-go-v2/config
    github.com/aws/aws-sdk-go-v2/credentials
)
```

### Client Initialization

```go
func NewClient(cfg *config.Config) (*Client, error) {
    esCfg := elasticsearch.Config{
        Addresses: []string{cfg.Host},
    }

    var httpTransport http.RoundTripper = http.DefaultTransport

    if cfg.AWSRegion != "" {
        transport, err := newAWSTransport(cfg)
        if err != nil {
            return nil, fmt.Errorf("creating AWS transport: %w", err)
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

## SigV4 Transport Implementation

New file: `internal/es/aws.go`

```go
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
        body, _ = io.ReadAll(req.Body)
        req.Body = io.NopCloser(bytes.NewReader(body))
    }

    hash := sha256.Sum256(body)
    err = t.signer.SignHTTP(req.Context(), creds, req,
        hex.EncodeToString(hash[:]), t.service, t.region, time.Now())
    if err != nil {
        return nil, fmt.Errorf("signing request: %w", err)
    }

    if body != nil {
        req.Body = io.NopCloser(bytes.NewReader(body))
    }

    return t.wrapped.RoundTrip(req)
}

func newAWSTransport(cfg *config.Config) (http.RoundTripper, error) {
    var opts []func(*awsconfig.LoadOptions) error
    if cfg.AWSProfile != "" {
        opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.AWSProfile))
    }

    awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
    if err != nil {
        return nil, fmt.Errorf("loading AWS config: %w", err)
    }

    if _, err := awsCfg.Credentials.Retrieve(context.Background()); err != nil {
        return nil, fmt.Errorf("AWS credentials: %w (check AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY, ~/.aws/credentials, or IAM role)", err)
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

## Testing

### URL Parsing Tests

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
        {"abc123xyz.us-west-2.aoss.amazonaws.com", "us-west-2", "aoss", true},
        {"localhost", "", "", false},
        {"elasticsearch.example.com", "", "", false},
    }
    for _, tt := range tests {
        region, service, ok := parseAWSEndpoint(tt.host)
        // assertions...
    }
}
```

### Error Handling

1. **Invalid AWS credentials**: Clear message pointing to credential sources
2. **Expired credentials**: AWS SDK handles refresh automatically
3. **Network errors**: Existing error handling unchanged

## Files to Change

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add AWS fields, `parseAWSEndpoint()`, update `ParseURL()` and `MaskedURL()` |
| `internal/config/config_test.go` | Add AWS URL parsing tests |
| `internal/es/client.go` | Use AWS transport conditionally, share with httpClient |
| `go.mod` | Add aws-sdk-go-v2 dependencies |

## New Files

| File | Purpose |
|------|---------|
| `internal/es/aws.go` | `sigv4Transport` and `newAWSTransport()` |

## Documentation Updates

| File | Changes |
|------|---------|
| `README.md` | Add AWS OpenSearch section with example config |
| `CLAUDE.md` | Update config example to show AWS options |

## Implementation Order

1. Add AWS SDK dependencies
2. Create `internal/es/aws.go` with signing transport
3. Update `internal/config/config.go` with AWS detection
4. Update `internal/es/client.go` to use AWS transport
5. Add tests
6. Update documentation
