package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/huh"
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
	rootCmd.Flags().StringVar(&renderFlag, "render", "", "Render a tab and exit (overview, workbench, browser, mappings, nodes)")
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

	esURL, err := resolveESURL(args, renderFlag != "")
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
	fmt.Print("\033[?1000l\033[?1002l\033[?1003l\033[?1006l")
	return nil
}

func resolveESURL(args []string, skipUI bool) (string, error) {
	if err := config.EnsureConfigDir(); err != nil {
		return "", fmt.Errorf("creating config dir: %w", err)
	}

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
			return resolveURLWithProgress(clusters, arg, skipUI)
		}
		return "", fmt.Errorf("cluster %q not found (no ~/.stoptail/config.yaml)", arg)
	}

	if envURL := os.Getenv("ES_URL"); envURL != "" {
		return envURL, nil
	}

	if clusters != nil && len(clusters.Clusters) > 0 {
		if skipUI {
			return "", fmt.Errorf("cluster name required with --render when multiple clusters configured")
		}
		return selectCluster(clusters)
	}

	return "http://localhost:9200", nil
}

func selectCluster(clusters *config.ClustersConfig) (string, error) {
	names := clusters.ClusterNames()
	sort.Strings(names)

	if len(names) == 1 {
		return resolveURLWithProgress(clusters, names[0], false)
	}

	picker := newClusterPickerModal(names)
	p := tea.NewProgram(picker, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	m := result.(*clusterPickerModal)
	if m.cancelled {
		return "", fmt.Errorf("cancelled")
	}

	return resolveURLWithProgress(clusters, m.selected, false)
}

type clusterPickerModal struct {
	form      *huh.Form
	selected  string
	cancelled bool
	done      bool
	width     int
	height    int
}

func newClusterPickerModal(names []string) *clusterPickerModal {
	m := &clusterPickerModal{}

	options := make([]huh.Option[string], len(names))
	for i, name := range names {
		options[i] = huh.NewOption(name, name)
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a cluster").
				Options(options...).
				Value(&m.selected),
		),
	).WithShowHelp(false)

	return m
}

func (m *clusterPickerModal) Init() tea.Cmd {
	return m.form.Init()
}

func (m *clusterPickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.done = true
		return m, tea.Quit
	}

	return m, cmd
}

func (m *clusterPickerModal) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	boxWidth := 50
	formView := m.form.View()

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBlue).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(formView)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

type urlResolverModel struct {
	clusters *config.ClustersConfig
	name     string
	spinner  spinner.Model
	url      string
	err      error
	done     bool
	width    int
	height   int
}

type urlResolvedMsg struct {
	url string
	err error
}

func newURLResolver(clusters *config.ClustersConfig, name string) urlResolverModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.SpinnerClr)
	return urlResolverModel{
		clusters: clusters,
		name:     name,
		spinner:  s,
	}
}

func (m urlResolverModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.resolveURL)
}

func (m urlResolverModel) resolveURL() tea.Msg {
	url, err := m.clusters.ResolveURL(m.name)
	return urlResolvedMsg{url: url, err: err}
}

func (m urlResolverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case urlResolvedMsg:
		m.url = msg.url
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m urlResolverModel) View() string {
	if m.done {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return ""
	}

	msgStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)
	content := fmt.Sprintf("%s %s", m.spinner.View(), msgStyle.Render("Fetching cluster URL..."))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBlue).
		Padding(1, 2).
		Width(50)

	box := boxStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func resolveURLWithProgress(clusters *config.ClustersConfig, name string, skipUI bool) (string, error) {
	entry, ok := clusters.Clusters[name]
	if !ok {
		return "", fmt.Errorf("cluster %q not found", name)
	}

	if entry.URL != "" {
		return entry.URL, nil
	}

	if skipUI {
		return clusters.ResolveURL(name)
	}

	m := newURLResolver(clusters, name)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	resolved := result.(urlResolverModel)
	if resolved.err != nil {
		return "", resolved.err
	}
	return resolved.url, nil
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

	case "mappings":
		state, err := client.FetchClusterState(ctx)
		if err != nil {
			return fmt.Errorf("fetching cluster state: %w", err)
		}
		mappings := ui.NewMappings()
		mappings.SetSize(width, height)
		mappings.SetIndices(state.Indices)
		fmt.Println(mappings.View())

	case "browser":
		state, err := client.FetchClusterState(ctx)
		if err != nil {
			return fmt.Errorf("fetching cluster state: %w", err)
		}
		browser := ui.NewBrowser()
		browser.SetClient(client)
		browser.SetSize(width, height)
		browser.SetIndices(state.Indices)
		fmt.Println(browser.View())

	case "tasks":
		tasks, err := client.FetchTasks(ctx)
		if err != nil {
			return fmt.Errorf("fetching tasks: %w", err)
		}
		tasksModel := ui.NewTasks()
		tasksModel.SetSize(width, height)
		tasksModel.SetTasks(tasks)
		fmt.Println(tasksModel.View())

	default:
		return fmt.Errorf("unknown tab: %s (use: overview, workbench, browser, mappings, nodes, tasks)", tab)
	}
	return nil
}
