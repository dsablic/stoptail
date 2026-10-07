package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/labtiva/stoptail/internal/es"
)

func pressKey(m TasksModel, key string) (TasksModel, tea.Cmd) {
	return m.Update(tea.KeyPressMsg{Code: []rune(key)[0], Text: key})
}

func TestTasksCancelSnapshotSendsFullTask(t *testing.T) {
	snapshot := es.TaskInfo{
		ID:                 "node:384",
		Action:             "cluster:admin/snapshot/create",
		Cancellable:        true,
		SnapshotRepository: "repo",
		SnapshotName:       "snap-1",
	}
	m := NewTasks()
	m.SetSize(120, 40)
	m.SetTasks([]es.TaskInfo{snapshot})

	m, _ = pressKey(m, "c")
	if m.confirming != snapshot.ID {
		t.Fatalf("confirming = %q, want %q", m.confirming, snapshot.ID)
	}
	if !strings.Contains(m.View(), "Abort snapshot repo:snap-1") {
		t.Error("expected snapshot-specific confirmation prompt")
	}

	m, cmd := pressKey(m, "y")
	if m.confirming != "" {
		t.Errorf("confirming = %q, want empty after confirm", m.confirming)
	}
	if cmd == nil {
		t.Fatal("expected cancel request command")
	}
	req, ok := cmd().(taskCancelRequestMsg)
	if !ok {
		t.Fatalf("got %T, want taskCancelRequestMsg", cmd())
	}
	if req.task.SnapshotName != "snap-1" || req.task.SnapshotRepository != "repo" {
		t.Errorf("request task = %+v, want snapshot repo:snap-1", req.task)
	}
}

func TestTasksCancelIgnoresNonCancellable(t *testing.T) {
	m := NewTasks()
	m.SetSize(120, 40)
	m.SetTasks([]es.TaskInfo{{ID: "node:1", Action: "indices:admin/forcemerge"}})

	m, _ = pressKey(m, "c")
	if m.confirming != "" {
		t.Errorf("confirming = %q, want empty for non-cancellable task", m.confirming)
	}
}
