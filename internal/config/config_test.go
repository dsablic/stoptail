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

func TestEnsureConfigDir(t *testing.T) {
	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() failed: %v", err)
	}
}

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

func TestMaskedURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "with username and password",
			cfg: &Config{
				Host:     "https://localhost:9200",
				Username: "elastic",
				Password: "secret",
			},
			want: "elastic:***@localhost:9200",
		},
		{
			name: "without credentials",
			cfg: &Config{
				Host: "https://localhost:9200",
			},
			want: "localhost:9200",
		},
		{
			name: "from ParseURL with embedded credentials",
			cfg: func() *Config {
				c, _ := ParseURL("https://myuser:mypass@es.example.com:9200")
				return c
			}(),
			want: "myuser:***@es.example.com:9200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.MaskedURL()
			if got != tt.want {
				t.Errorf("MaskedURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
