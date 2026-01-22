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
