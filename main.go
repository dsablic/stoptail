package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labtiva/stoptail/internal/config"
	"github.com/labtiva/stoptail/internal/es"
	"github.com/labtiva/stoptail/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	themeFlag  string
	renderFlag string
	widthFlag  int
	heightFlag int
	bodyFlag   string
	viewFlag   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "stoptail [cluster]",
		Short: "Elasticsearch TUI - like elasticsearch-head but for your terminal",
		Long: `stoptail is a terminal UI for Elasticsearch.

Connect to a cluster by URL, by name from ~/.stoptail/config.yaml,
or via the ES_URL environment variable.

Examples:
  stoptail                              # Connect to localhost:9200
  stoptail http://localhost:9200        # Connect by URL
  stoptail https://user:pass@host:9200  # Connect with credentials
  stoptail production                   # Connect to named cluster from config`,
		Args:    cobra.MaximumNArgs(1),
		Version: formatVersion(),
		RunE:    run,
	}

	rootCmd.Flags().StringVar(&themeFlag, "theme", "auto", "Color theme: auto, dark, light")
	rootCmd.Flags().StringVar(&renderFlag, "render", "", "Render a tab and exit (overview, nodes, workbench)")
	rootCmd.Flags().IntVar(&widthFlag, "width", 120, "Terminal width for --render")
	rootCmd.Flags().IntVar(&heightFlag, "height", 40, "Terminal height for --render")
	rootCmd.Flags().StringVar(&bodyFlag, "body", "", "JSON body for --render workbench")
	rootCmd.Flags().StringVar(&viewFlag, "view", "", "View for --render nodes (memory, disk, fielddata)")

	rootCmd.SetVersionTemplate("{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func formatVersion() string {
	v := fmt.Sprintf("stoptail %s", version)
	if commit != "none" {
		if len(commit) >= 7 {
			v += fmt.Sprintf(" (%s)", commit[:7])
		} else {
			v += fmt.Sprintf(" (%s)", commit)
		}
	}
	if date != "unknown" {
		v += fmt.Sprintf(" built %s", date)
	}
	return v
}

func run(cmd *cobra.Command, args []string) error {
	ui.SetTheme(themeFlag)

	esURL, err := resolveESURL(args)
	if err != nil {
		return err
	}

	cfg, err := config.ParseURL(esURL)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	client, err := es.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	if renderFlag != "" {
		return renderAndExit(client, renderFlag, widthFlag, heightFlag, bodyFlag, viewFlag)
	}

	p := tea.NewProgram(ui.New(client, cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
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

func renderAndExit(client *es.Client, tab string, width, height int, body, view string) error {
	ctx := context.Background()

	switch tab {
	case "overview":
		state, err := client.FetchClusterState(ctx)
		if err != nil {
			return fmt.Errorf("fetching cluster state: %w", err)
		}
		overview := ui.NewOverview()
		overview.SetSize(width, height)
		overview.SetCluster(state)
		fmt.Println(overview.View())

	case "nodes":
		state, err := client.FetchNodesState(ctx)
		if err != nil {
			return fmt.Errorf("fetching nodes state: %w", err)
		}
		nodes := ui.NewNodes()
		nodes.SetSize(width, height)
		if view != "" {
			nodes.SetView(view)
		}
		nodes.SetState(state)
		fmt.Println(nodes.View())

	case "workbench":
		workbench := ui.NewWorkbench()
		workbench.SetClient(client)
		workbench.SetSize(width, height)
		if body != "" {
			workbench.SetBody(body)
		}
		fmt.Println(workbench.View())

	default:
		return fmt.Errorf("unknown tab: %s (use: overview, nodes, workbench)", tab)
	}
	return nil
}
