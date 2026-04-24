package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryAdd(t *testing.T) {
	h := &History{}

	added := h.Add(HistoryEntry{Method: "GET", Path: "/_cat/indices"})
	if !added {
		t.Error("expected Add to return true for new entry")
	}
	if len(h.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(h.Entries))
	}

	added = h.Add(HistoryEntry{Method: "GET", Path: "/_cat/indices"})
	if added {
		t.Error("expected Add to return false for duplicate entry")
	}
	if len(h.Entries) != 1 {
		t.Fatalf("expected 1 entry after duplicate, got %d", len(h.Entries))
	}
}

func TestHistoryAddEmptyPath(t *testing.T) {
	h := &History{}

	added := h.Add(HistoryEntry{Method: "GET", Path: ""})
	if added {
		t.Error("expected Add to return false for empty path")
	}
	if len(h.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(h.Entries))
	}
}

func TestHistoryAddDifferentModes(t *testing.T) {
	h := &History{}

	h.Add(HistoryEntry{Method: "GET", Path: "/_search", Mode: "rest"})
	added := h.Add(HistoryEntry{Method: "GET", Path: "/_search", Mode: "dsl"})
	if !added {
		t.Error("expected Add to return true for same path but different mode")
	}
	if len(h.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(h.Entries))
	}
}

func TestHistoryAddDifferentBodies(t *testing.T) {
	h := &History{}

	h.Add(HistoryEntry{Method: "POST", Path: "/_search", Body: `{"query":{"match_all":{}}}`})
	added := h.Add(HistoryEntry{Method: "POST", Path: "/_search", Body: `{"query":{"term":{"status":"active"}}}`})
	if !added {
		t.Error("expected Add to return true for same path but different body")
	}
	if len(h.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(h.Entries))
	}
}

func TestHistoryLast(t *testing.T) {
	h := &History{}
	if h.Last() != nil {
		t.Error("expected nil for empty history")
	}

	h.Add(HistoryEntry{Method: "GET", Path: "/_cat/indices"})
	h.Add(HistoryEntry{Method: "POST", Path: "/_search", Body: "{}"})

	last := h.Last()
	if last == nil {
		t.Fatal("expected non-nil last entry")
	}
	if last.Path != "/_search" {
		t.Errorf("expected path /_search, got %s", last.Path)
	}
}

func TestHistoryLastByMode(t *testing.T) {
	h := &History{}

	h.Add(HistoryEntry{Method: "GET", Path: "/a", Mode: "rest"})
	h.Add(HistoryEntry{Method: "POST", Path: "/b", Mode: "dsl"})
	h.Add(HistoryEntry{Method: "GET", Path: "/c", Mode: "rest"})

	last := h.LastByMode("dsl")
	if last == nil {
		t.Fatal("expected non-nil entry for mode dsl")
	}
	if last.Path != "/b" {
		t.Errorf("expected path /b, got %s", last.Path)
	}

	last = h.LastByMode("rest")
	if last == nil {
		t.Fatal("expected non-nil entry for mode rest")
	}
	if last.Path != "/c" {
		t.Errorf("expected path /c, got %s", last.Path)
	}

	last = h.LastByMode("unknown")
	if last != nil {
		t.Error("expected nil for unknown mode")
	}
}

func TestHistorySaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	h := &History{}
	h.Add(HistoryEntry{Method: "GET", Path: "/_cat/indices"})
	h.Add(HistoryEntry{Method: "POST", Path: "/_search", Body: `{"query":{}}`, Mode: "dsl"})

	if err := SaveHistory(h); err != nil {
		t.Fatalf("SaveHistory error: %v", err)
	}

	stoptailDir := filepath.Join(tmpDir, ".stoptail")
	if _, err := os.Stat(filepath.Join(stoptailDir, "history.json")); err != nil {
		t.Fatalf("history.json not created: %v", err)
	}

	loaded, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory error: %v", err)
	}

	if len(loaded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Path != "/_cat/indices" {
		t.Errorf("expected path /_cat/indices, got %s", loaded.Entries[0].Path)
	}
	if loaded.Entries[1].Mode != "dsl" {
		t.Errorf("expected mode dsl, got %s", loaded.Entries[1].Mode)
	}
}

func TestLoadHistoryMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	h, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory error: %v", err)
	}
	if len(h.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(h.Entries))
	}
}

func TestLoadHistoryCorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	stoptailDir := filepath.Join(tmpDir, ".stoptail")
	os.MkdirAll(stoptailDir, 0755)
	os.WriteFile(filepath.Join(stoptailDir, "history.json"), []byte("not json"), 0644)

	h, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory error: %v", err)
	}
	if len(h.Entries) != 0 {
		t.Errorf("expected 0 entries for corrupt file, got %d", len(h.Entries))
	}
}
