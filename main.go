package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	themeFlag := flag.String("theme", "auto", "Color theme: auto, dark, light")
	renderFlag := flag.String("render", "", "Render a tab and exit (overview, nodes, workbench)")
	widthFlag := flag.Int("width", 120, "Terminal width for --render")
	heightFlag := flag.Int("height", 40, "Terminal height for --render")
	flag.Parse()

	ui.SetTheme(*themeFlag)

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

	p := tea.NewProgram(ui.New(client, cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
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
		return "", fmt.Errorf("cluster %q not found (no ~/.stoptail/config.yaml)", arg)
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

	if len(names) == 1 {
		return clusters.ResolveURL(names[0])
	}

	picker := newClusterPicker(names)
	p := tea.NewProgram(picker)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	m := result.(clusterPickerModel)
	if m.cancelled {
		return "", fmt.Errorf("cancelled")
	}

	return clusters.ResolveURL(names[m.selected])
}

type clusterPickerModel struct {
	names     []string
	selected  int
	cancelled bool
}

func newClusterPicker(names []string) clusterPickerModel {
	return clusterPickerModel{names: names}
}

func (m clusterPickerModel) Init() tea.Cmd {
	return nil
}

func (m clusterPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.names)-1 {
				m.selected++
			}
		case "enter":
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m clusterPickerModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Select a cluster:"))
	b.WriteString("\n\n")

	for i, name := range m.names {
		if i == m.selected {
			b.WriteString(cursorStyle.Render("  > "))
			b.WriteString(selectedStyle.Render(name))
		} else {
			b.WriteString("    ")
			b.WriteString(normalStyle.Render(name))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate  Enter: select  q: quit"))
	b.WriteString("\n")
	return b.String()
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

	case "workbench":
		workbench := ui.NewWorkbench()
		workbench.SetClient(client)
		workbench.SetSize(width, height)
		fmt.Println(workbench.View())

	default:
		fmt.Fprintf(os.Stderr, "Unknown tab: %s (use: overview, nodes, workbench)\n", tab)
		os.Exit(1)
	}
}
