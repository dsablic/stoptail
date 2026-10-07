package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

func TestWorkbenchDragReleasedOnTabBarKeepsTab(t *testing.T) {
	m := Model{connected: true, activeTab: TabWorkbench, workbench: NewWorkbench(), width: 120, height: 34}
	m.workbench.SetSize(120, 30)
	m.workbench.SetBody("hello world")

	const headerHeight = 2
	press := tea.Mouse{X: editorOffsetX, Y: editorOffsetY + headerHeight, Button: tea.MouseLeft}
	tabBar := tea.Mouse{X: 2, Y: 1, Button: tea.MouseLeft}
	for _, msg := range []tea.Msg{tea.MouseClickMsg(press), tea.MouseMotionMsg(tabBar), tea.MouseReleaseMsg(tabBar)} {
		next, _ := m.Update(msg)
		m = next.(Model)
	}

	if m.activeTab != TabWorkbench {
		t.Errorf("activeTab = %v, want workbench", m.activeTab)
	}
	if m.workbench.Dragging() {
		t.Error("drag should end when released on the tab bar")
	}
}
