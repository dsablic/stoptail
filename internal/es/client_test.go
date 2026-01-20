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
