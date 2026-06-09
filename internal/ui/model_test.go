package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
)

func TestNewDeferredShowsCenteredResolvingView(t *testing.T) {
	m := NewDeferred(func() (*es.Client, *config.Config, error) {
		return nil, nil, nil
	}, "Fetching cluster credentials...")
	m.width = 80
	m.height = 24

	v := m.View()
	if !v.AltScreen {
		t.Error("resolving view must run in the alt-screen so it shares the main TUI session (no flicker)")
	}

	resolving := m.renderResolving()
	if !strings.Contains(resolving, "Fetching cluster credentials...") {
		t.Errorf("resolving view should show the message, got: %q", resolving)
	}
	if got := strings.Count(resolving, "\n") + 1; got != m.height {
		t.Errorf("resolving view should fill the full height for centering: got %d lines, want %d", got, m.height)
	}
}

func TestClusterResolvedMsgSetsClient(t *testing.T) {
	m := NewDeferred(func() (*es.Client, *config.Config, error) {
		return nil, nil, nil
	}, "Fetching cluster URL...")

	client := &es.Client{}
	cfg := &config.Config{}

	updated, cmd := m.Update(clusterResolvedMsg{client: client, cfg: cfg})
	got := updated.(Model)

	if got.client != client {
		t.Error("resolved client should be stored on the model")
	}
	if got.cfg != cfg {
		t.Error("resolved cfg should be stored on the model")
	}
	if cmd == nil {
		t.Error("a successful resolution should trigger the connect command")
	}
}

func TestClusterResolvedMsgErrorIsSurfaced(t *testing.T) {
	m := NewDeferred(func() (*es.Client, *config.Config, error) {
		return nil, nil, nil
	}, "Fetching cluster URL...")
	m.width = 80
	m.height = 24

	updated, _ := m.Update(clusterResolvedMsg{err: fmt.Errorf("boom")})
	got := updated.(Model)

	if got.err == nil {
		t.Fatal("resolution error should be stored on the model")
	}
	if !strings.Contains(got.renderResolving(), "boom") {
		t.Error("resolution error should be shown in the resolving view")
	}
}
