package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/labtiva/stoptail/internal/storage"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Host       string
	Username   string
	Password   string
	AWSRegion  string
	AWSService string
	AWSProfile string
	TLSCert    string
	TLSKey     string
	TLSCA      string
}

func (c *Config) IsAWS() bool {
	return c.AWSRegion != ""
}

func (c *Config) IsMTLS() bool {
	return c.TLSCert != "" && c.TLSKey != ""
}

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

func (c *Config) MaskedURL() string {
	u, err := url.Parse(c.Host)
	if err != nil || u == nil {
		return c.Host
	}
	if c.IsAWS() {
		return fmt.Sprintf("%s (AWS %s)", strings.TrimPrefix(u.Host, "www."), c.AWSRegion)
	}
	if c.IsMTLS() {
		return fmt.Sprintf("%s (mTLS)", strings.TrimPrefix(u.Host, "www."))
	}
	if c.Username != "" {
		return fmt.Sprintf("%s:***@%s", c.Username, u.Host)
	}
	return u.Host
}

func (c *Config) DisplayHost() string {
	u, err := url.Parse(c.Host)
	if err != nil || u == nil {
		return c.Host
	}
	return strings.TrimPrefix(u.Host, "www.")
}

type ClusterEntry struct {
	URL                string `yaml:"url"`
	URLCommand         string `yaml:"url_command"`
	CredentialsCommand string `yaml:"credentials_command"`
	AWSProfile         string `yaml:"aws_profile"`
}

type ResolvedCluster struct {
	URL     string
	TLSCert string
	TLSKey  string
	TLSCA   string
}

type ClustersConfig struct {
	Clusters map[string]ClusterEntry `yaml:"clusters"`
}

func EnsureConfigDir() error {
	dir, err := storage.StoptailDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		stub := `# stoptail configuration
# Add your Elasticsearch clusters here
#
# clusters:
#   production:
#     url: https://user:pass@es-prod:9200
#   staging:
#     url_command: "vault read -field=url secret/es-staging"
#   mtls-cluster:
#     credentials_command: "aws secretsmanager get-secret-value --secret-id my-project/es-credentials --query SecretString --output text"
`
		if err := os.WriteFile(configPath, []byte(stub), 0644); err != nil {
			return err
		}
	}

	return nil
}

func LoadClustersConfig() (*ClustersConfig, error) {
	dir, err := storage.StoptailDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, ".stoptail.yaml")
			data, err = os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, nil
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var cfg ClustersConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &cfg, nil
}

func (c *ClustersConfig) ClusterNames() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.Clusters))
	for name := range c.Clusters {
		names = append(names, name)
	}
	return names
}

func (c *ClustersConfig) Resolve(name string) (*ResolvedCluster, error) {
	entry, ok := c.Clusters[name]
	if !ok {
		return nil, fmt.Errorf("cluster %q not found", name)
	}

	if entry.URL != "" {
		return &ResolvedCluster{URL: entry.URL}, nil
	}

	if entry.CredentialsCommand != "" {
		return resolveCredentialsCommand(name, entry.CredentialsCommand)
	}

	if entry.URLCommand != "" {
		url, err := runCommand(name, entry.URLCommand)
		if err != nil {
			return nil, err
		}
		return &ResolvedCluster{URL: url}, nil
	}

	return nil, fmt.Errorf("cluster %q has no url, url_command, or credentials_command", name)
}

type mtlsCredentials struct {
	Cert     string `json:"cert"`
	Key      string `json:"key"`
	CA       string `json:"ca"`
	Endpoint string `json:"endpoint"`
}

func resolveCredentialsCommand(name, command string) (*ResolvedCluster, error) {
	output, err := runCommand(name, command)
	if err != nil {
		return nil, err
	}

	var creds mtlsCredentials
	if err := json.Unmarshal([]byte(output), &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials_command output for %q: %w", name, err)
	}

	if creds.Endpoint == "" {
		return nil, fmt.Errorf("credentials_command for %q: missing endpoint", name)
	}
	if creds.Cert == "" {
		return nil, fmt.Errorf("credentials_command for %q: missing cert", name)
	}
	if creds.Key == "" {
		return nil, fmt.Errorf("credentials_command for %q: missing key", name)
	}

	return &ResolvedCluster{
		URL:     creds.Endpoint,
		TLSCert: creds.Cert,
		TLSKey:  creds.Key,
		TLSCA:   creds.CA,
	}, nil
}

func runCommand(name, command string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("sh", "-c", command)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("running command for %q: %w\n%s", name, err, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("running command for %q: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}
