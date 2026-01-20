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
