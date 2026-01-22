package config

import (
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

func (c *Config) MaskedURL() string {
	u, err := url.Parse(c.Host)
	if err != nil || u == nil {
		return c.Host
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
	URL        string `yaml:"url"`
	URLCommand string `yaml:"url_command"`
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

func (c *ClustersConfig) ResolveURL(name string) (string, error) {
	entry, ok := c.Clusters[name]
	if !ok {
		return "", fmt.Errorf("cluster %q not found", name)
	}

	if entry.URL != "" {
		return entry.URL, nil
	}

	if entry.URLCommand != "" {
		out, err := exec.Command("sh", "-c", entry.URLCommand).Output()
		if err != nil {
			return "", fmt.Errorf("running url_command for %q: %w", name, err)
		}
		return strings.TrimSpace(string(out)), nil
	}

	return "", fmt.Errorf("cluster %q has no url or url_command", name)
}
