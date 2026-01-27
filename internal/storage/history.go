package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type HistoryEntry struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   string `json:"body"`
	Mode   string `json:"mode,omitempty"`
}

type History struct {
	Entries []HistoryEntry `json:"entries"`
}

func StoptailDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".stoptail"), nil
}

func ensureDir() error {
	dir, err := StoptailDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func historyPath() (string, error) {
	dir, err := StoptailDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

func LoadHistory() (*History, error) {
	path, err := historyPath()
	if err != nil {
		return &History{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &History{}, nil
		}
		return nil, err
	}

	var history History
	if err := json.Unmarshal(data, &history); err != nil {
		return &History{}, nil
	}

	return &history, nil
}

func SaveHistory(history *History) error {
	if err := ensureDir(); err != nil {
		return err
	}

	path, err := historyPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (h *History) Add(entry HistoryEntry) bool {
	if entry.Path == "" {
		return false
	}
	for _, e := range h.Entries {
		if e.Method == entry.Method && e.Path == entry.Path && e.Body == entry.Body && e.Mode == entry.Mode {
			return false
		}
	}
	h.Entries = append(h.Entries, entry)
	return true
}

func (h *History) Last() *HistoryEntry {
	for i := len(h.Entries) - 1; i >= 0; i-- {
		if h.Entries[i].Path != "" {
			return &h.Entries[i]
		}
	}
	return nil
}

func (h *History) LastByMode(mode string) *HistoryEntry {
	for i := len(h.Entries) - 1; i >= 0; i-- {
		entry := &h.Entries[i]
		if entry.Path != "" && entry.Mode == mode {
			return entry
		}
	}
	return nil
}
