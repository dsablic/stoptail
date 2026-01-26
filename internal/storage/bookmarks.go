package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type Bookmark struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   string `json:"body"`
}

type Bookmarks struct {
	Items []Bookmark `json:"bookmarks"`
}

func bookmarksPath() (string, error) {
	dir, err := StoptailDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bookmarks.json"), nil
}

func LoadBookmarks() (*Bookmarks, error) {
	path, err := bookmarksPath()
	if err != nil {
		return &Bookmarks{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Bookmarks{}, nil
		}
		return nil, err
	}

	var bookmarks Bookmarks
	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return &Bookmarks{}, nil
	}

	return &bookmarks, nil
}

func SaveBookmarks(bookmarks *Bookmarks) error {
	if err := ensureDir(); err != nil {
		return err
	}

	path, err := bookmarksPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (b *Bookmarks) Add(bookmark Bookmark) bool {
	if bookmark.Name == "" || bookmark.Path == "" {
		return false
	}
	for i, existing := range b.Items {
		if existing.Name == bookmark.Name {
			b.Items[i] = bookmark
			return true
		}
	}
	b.Items = append(b.Items, bookmark)
	sort.Slice(b.Items, func(i, j int) bool {
		return b.Items[i].Name < b.Items[j].Name
	})
	return true
}

func (b *Bookmarks) Delete(name string) bool {
	for i, existing := range b.Items {
		if existing.Name == name {
			b.Items = append(b.Items[:i], b.Items[i+1:]...)
			return true
		}
	}
	return false
}

func (b *Bookmarks) Get(name string) *Bookmark {
	for _, item := range b.Items {
		if item.Name == name {
			return &item
		}
	}
	return nil
}
