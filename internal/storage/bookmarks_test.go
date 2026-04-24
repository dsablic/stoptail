package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBookmarksAdd(t *testing.T) {
	b := &Bookmarks{}

	added := b.Add(Bookmark{Name: "search", Method: "POST", Path: "/_search", Body: "{}"})
	if !added {
		t.Error("expected Add to return true for new bookmark")
	}
	if len(b.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(b.Items))
	}
}

func TestBookmarksAddEmptyNameOrPath(t *testing.T) {
	b := &Bookmarks{}

	if b.Add(Bookmark{Name: "", Path: "/_search"}) {
		t.Error("expected false for empty name")
	}
	if b.Add(Bookmark{Name: "test", Path: ""}) {
		t.Error("expected false for empty path")
	}
	if len(b.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(b.Items))
	}
}

func TestBookmarksAddUpdatesExisting(t *testing.T) {
	b := &Bookmarks{}

	b.Add(Bookmark{Name: "search", Method: "GET", Path: "/old"})
	b.Add(Bookmark{Name: "search", Method: "POST", Path: "/new"})

	if len(b.Items) != 1 {
		t.Fatalf("expected 1 item after update, got %d", len(b.Items))
	}
	if b.Items[0].Path != "/new" {
		t.Errorf("expected path /new, got %s", b.Items[0].Path)
	}
	if b.Items[0].Method != "POST" {
		t.Errorf("expected method POST, got %s", b.Items[0].Method)
	}
}

func TestBookmarksAddSortsByName(t *testing.T) {
	b := &Bookmarks{}

	b.Add(Bookmark{Name: "charlie", Path: "/c"})
	b.Add(Bookmark{Name: "alpha", Path: "/a"})
	b.Add(Bookmark{Name: "bravo", Path: "/b"})

	if len(b.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(b.Items))
	}
	if b.Items[0].Name != "alpha" {
		t.Errorf("expected first item alpha, got %s", b.Items[0].Name)
	}
	if b.Items[1].Name != "bravo" {
		t.Errorf("expected second item bravo, got %s", b.Items[1].Name)
	}
	if b.Items[2].Name != "charlie" {
		t.Errorf("expected third item charlie, got %s", b.Items[2].Name)
	}
}

func TestBookmarksDelete(t *testing.T) {
	b := &Bookmarks{}
	b.Add(Bookmark{Name: "alpha", Path: "/a"})
	b.Add(Bookmark{Name: "bravo", Path: "/b"})

	deleted := b.Delete("alpha")
	if !deleted {
		t.Error("expected Delete to return true")
	}
	if len(b.Items) != 1 {
		t.Fatalf("expected 1 item after delete, got %d", len(b.Items))
	}
	if b.Items[0].Name != "bravo" {
		t.Errorf("expected remaining item bravo, got %s", b.Items[0].Name)
	}
}

func TestBookmarksDeleteNotFound(t *testing.T) {
	b := &Bookmarks{}
	b.Add(Bookmark{Name: "alpha", Path: "/a"})

	deleted := b.Delete("nonexistent")
	if deleted {
		t.Error("expected Delete to return false for nonexistent bookmark")
	}
	if len(b.Items) != 1 {
		t.Fatalf("expected 1 item unchanged, got %d", len(b.Items))
	}
}

func TestBookmarksGet(t *testing.T) {
	b := &Bookmarks{}
	b.Add(Bookmark{Name: "search", Method: "POST", Path: "/_search", Body: `{"query":{}}`})

	got := b.Get("search")
	if got == nil {
		t.Fatal("expected non-nil bookmark")
	}
	if got.Path != "/_search" {
		t.Errorf("expected path /_search, got %s", got.Path)
	}
	if got.Body != `{"query":{}}` {
		t.Errorf("expected body {\"query\":{}}, got %s", got.Body)
	}
}

func TestBookmarksGetNotFound(t *testing.T) {
	b := &Bookmarks{}

	got := b.Get("nonexistent")
	if got != nil {
		t.Error("expected nil for nonexistent bookmark")
	}
}

func TestBookmarksSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	b := &Bookmarks{}
	b.Add(Bookmark{Name: "search", Method: "POST", Path: "/_search", Body: `{"query":{}}`})
	b.Add(Bookmark{Name: "indices", Method: "GET", Path: "/_cat/indices"})

	if err := SaveBookmarks(b); err != nil {
		t.Fatalf("SaveBookmarks error: %v", err)
	}

	stoptailDir := filepath.Join(tmpDir, ".stoptail")
	if _, err := os.Stat(filepath.Join(stoptailDir, "bookmarks.json")); err != nil {
		t.Fatalf("bookmarks.json not created: %v", err)
	}

	loaded, err := LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks error: %v", err)
	}

	if len(loaded.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(loaded.Items))
	}
	if loaded.Items[0].Name != "indices" {
		t.Errorf("expected first item indices, got %s", loaded.Items[0].Name)
	}
	if loaded.Items[1].Name != "search" {
		t.Errorf("expected second item search, got %s", loaded.Items[1].Name)
	}
}

func TestLoadBookmarksMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	b, err := LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks error: %v", err)
	}
	if len(b.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(b.Items))
	}
}

func TestLoadBookmarksCorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	stoptailDir := filepath.Join(tmpDir, ".stoptail")
	os.MkdirAll(stoptailDir, 0755)
	os.WriteFile(filepath.Join(stoptailDir, "bookmarks.json"), []byte("{bad json"), 0644)

	b, err := LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks error: %v", err)
	}
	if len(b.Items) != 0 {
		t.Errorf("expected 0 items for corrupt file, got %d", len(b.Items))
	}
}
