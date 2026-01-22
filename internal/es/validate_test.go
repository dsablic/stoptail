package es

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labtiva/stoptail/internal/config"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		wantValid  bool
		wantErr    string
	}{
		{
			name:       "valid query",
			response:   `{"valid": true}`,
			statusCode: 200,
			wantValid:  true,
		},
		{
			name:       "invalid query",
			response:   `{"valid": false, "error": "unknown field [matchh]"}`,
			statusCode: 200,
			wantValid:  false,
			wantErr:    "unknown field [matchh]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			cfg := &config.Config{Host: server.URL}
			client, _ := NewClient(cfg)
			result, err := client.ValidateQuery(context.Background(), "test-index", json.RawMessage(`{}`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tt.wantValid {
				t.Errorf("got valid=%v, want %v", result.Valid, tt.wantValid)
			}
			if result.Error != tt.wantErr {
				t.Errorf("got error=%q, want %q", result.Error, tt.wantErr)
			}
		})
	}
}
