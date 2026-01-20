package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
	"github.com/labtiva/stoptail/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	renderFlag := flag.String("render", "", "Render a tab and exit (overview, nodes)")
	widthFlag := flag.Int("width", 120, "Terminal width for --render")
	heightFlag := flag.Int("height", 40, "Terminal height for --render")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("stoptail %s", version)
		if commit != "none" {
			if len(commit) >= 7 {
				fmt.Printf(" (%s)", commit[:7])
			} else {
				fmt.Printf(" (%s)", commit)
			}
		}
		if date != "unknown" {
			fmt.Printf(" built %s", date)
		}
		fmt.Println()
		return
	}

	esURL, err := resolveESURL(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.ParseURL(esURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	client, err := es.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client error: %v\n", err)
		os.Exit(1)
	}

	if *renderFlag != "" {
		renderAndExit(client, *renderFlag, *widthFlag, *heightFlag)
		return
	}

	p := tea.NewProgram(ui.New(client, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func resolveESURL(args []string) (string, error) {
	clusters, err := config.LoadClustersConfig()
	if err != nil {
		return "", fmt.Errorf("loading clusters config: %w", err)
	}

	if len(args) > 0 {
		arg := args[0]
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			return arg, nil
		}
		if clusters != nil {
			return clusters.ResolveURL(arg)
		}
		return "", fmt.Errorf("cluster %q not found (no ~/.stoptail.yaml)", arg)
	}

	if envURL := os.Getenv("ES_URL"); envURL != "" {
		return envURL, nil
	}

	if clusters != nil && len(clusters.Clusters) > 0 {
		return selectCluster(clusters)
	}

	return "http://localhost:9200", nil
}

func selectCluster(clusters *config.ClustersConfig) (string, error) {
	names := clusters.ClusterNames()
	sort.Strings(names)

	fmt.Println("Select a cluster:")
	for i, name := range names {
		fmt.Printf("  %d. %s\n", i+1, name)
	}
	fmt.Print("Enter number: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(names) {
		return "", fmt.Errorf("invalid selection: %s", input)
	}

	return clusters.ResolveURL(names[num-1])
}

func renderAndExit(client *es.Client, tab string, width, height int) {
	ctx := context.Background()

	switch tab {
	case "overview":
		state, err := client.FetchClusterState(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching cluster state: %v\n", err)
			os.Exit(1)
		}
		overview := ui.NewOverview()
		overview.SetSize(width, height)
		overview.SetCluster(state)
		fmt.Println(overview.View())

	case "nodes":
		state, err := client.FetchNodesState(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching nodes state: %v\n", err)
			os.Exit(1)
		}
		nodes := ui.NewNodes()
		nodes.SetSize(width, height)
		nodes.SetState(state)
		fmt.Println(nodes.View())

	default:
		fmt.Fprintf(os.Stderr, "Unknown tab: %s (use: overview, nodes)\n", tab)
		os.Exit(1)
	}
}
