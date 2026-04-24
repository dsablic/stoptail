package ui

import (
	"testing"
)

func TestParseHotThread(t *testing.T) {
	tests := []struct {
		name       string
		node       string
		line       string
		wantNil    bool
		wantTotal  string
		wantCPU    string
		wantOther  string
		wantType   string
	}{
		{
			name:      "basic thread",
			node:      "node1",
			line:      "50.5% [cpu=45.2%, other=5.3%] (active) thread 'elasticsearch[node1][search][T#1]'",
			wantTotal: "50.5%",
			wantCPU:   "45.2%",
			wantOther: "5.3%",
			wantType:  "search",
		},
		{
			name:    "no percentage",
			node:    "node1",
			line:    "no percentage here",
			wantNil: true,
		},
		{
			name:      "no brackets",
			node:      "node1",
			line:      "25.0% some text without breakdown",
			wantTotal: "25.0%",
			wantCPU:   "",
			wantOther: "",
			wantType:  "",
		},
		{
			name:      "lucene thread",
			node:      "node1",
			line:      "10.0% [cpu=10.0%] (active) thread 'lucene-merge-0'",
			wantTotal: "10.0%",
			wantCPU:   "10.0%",
			wantType:  "merge",
		},
		{
			name:      "refresh thread",
			node:      "node1",
			line:      "5.0% [cpu=5.0%] (active) thread 'refresh-worker'",
			wantTotal: "5.0%",
			wantCPU:   "5.0%",
			wantType:  "refresh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHotThread(tt.node, tt.line)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.node != tt.node {
				t.Errorf("node = %q, want %q", got.node, tt.node)
			}
			if got.total != tt.wantTotal {
				t.Errorf("total = %q, want %q", got.total, tt.wantTotal)
			}
			if got.cpu != tt.wantCPU {
				t.Errorf("cpu = %q, want %q", got.cpu, tt.wantCPU)
			}
			if got.other != tt.wantOther {
				t.Errorf("other = %q, want %q", got.other, tt.wantOther)
			}
			if got.threadType != tt.wantType {
				t.Errorf("threadType = %q, want %q", got.threadType, tt.wantType)
			}
		})
	}
}

func TestExtractThreadType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"elasticsearch[node1][search][T#1]", "search"},
		{"elasticsearch[node1][write][T#2]", "write"},
		{"elasticsearch[node1][get][T#3]", "get"},
		{"lucene-merge-0", "merge"},
		{"refresh-worker-1", "refresh"},
		{"flush-thread-0", "flush"},
		{"some-unknown-thread", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractThreadType(tt.input)
			if got != tt.want {
				t.Errorf("extractThreadType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAllHotThreads(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		threads := parseAllHotThreads("")
		if threads != nil {
			t.Errorf("expected nil, got %d threads", len(threads))
		}
	})

	t.Run("single node with threads", func(t *testing.T) {
		raw := `::: {node1}
   Hot threads at 2024-01-01:
   50.5% [cpu=45.2%, other=5.3%] (active) thread 'elasticsearch[node1][search][T#1]'
   25.0% [cpu=20.0%, other=5.0%] (active) thread 'elasticsearch[node1][write][T#2]'`

		threads := parseAllHotThreads(raw)
		if len(threads) != 2 {
			t.Fatalf("expected 2 threads, got %d", len(threads))
		}
		if threads[0].node != "node1" {
			t.Errorf("thread[0].node = %q, want %q", threads[0].node, "node1")
		}
		if threads[0].threadType != "search" {
			t.Errorf("thread[0].type = %q, want %q", threads[0].threadType, "search")
		}
	})

	t.Run("multiple nodes", func(t *testing.T) {
		raw := `::: {node1}
   Hot threads at 2024-01-01:
   50.5% [cpu=45.2%] (active) thread 'elasticsearch[node1][search][T#1]'
::: {node2}
   Hot threads at 2024-01-01:
   30.0% [cpu=25.0%] (active) thread 'elasticsearch[node2][write][T#1]'`

		threads := parseAllHotThreads(raw)
		if len(threads) != 2 {
			t.Fatalf("expected 2 threads, got %d", len(threads))
		}
		if threads[0].node != "node1" {
			t.Errorf("thread[0].node = %q, want %q", threads[0].node, "node1")
		}
		if threads[1].node != "node2" {
			t.Errorf("thread[1].node = %q, want %q", threads[1].node, "node2")
		}
	})

	t.Run("idle nodes", func(t *testing.T) {
		raw := `::: {node1}
   Hot threads at 2024-01-01:

::: {node2}
   Hot threads at 2024-01-01:
`

		threads := parseAllHotThreads(raw)
		if len(threads) != 0 {
			t.Errorf("expected 0 threads, got %d", len(threads))
		}
	})
}

func TestParsePercent(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"45.5", 45.5},
		{"100", 100},
		{"  67.8  ", 67.8},
		{"abc", 0},
		{"", 0},
	}

	m := NodesModel{}
	for _, tt := range tests {
		got := m.parsePercent(tt.input)
		if got != tt.want {
			t.Errorf("parsePercent(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
