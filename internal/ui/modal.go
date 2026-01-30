package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type ModalType int

const (
	ModalNone ModalType = iota
	ModalCreateIndex
	ModalDeleteIndex
	ModalCloseIndex
	ModalAddAlias
	ModalRemoveAlias
	ModalError
)

type Modal struct {
	modalType ModalType
	form      *huh.Form
	err       string
	done      bool
	cancelled bool

	indexName string
	shards    string
	replicas  string
	aliasName string
	confirmed bool
	aliases   []string
}

func NewCreateIndexModal() *Modal {
	m := &Modal{
		modalType: ModalCreateIndex,
		shards:    "1",
		replicas:  "1",
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Index name").
				Value(&m.indexName).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("Shards").
				Value(&m.shards),
			huh.NewInput().
				Title("Replicas").
				Value(&m.replicas),
		),
	).WithShowHelp(false).WithShowErrors(true)

	return m
}

func NewDeleteIndexModal(indexName string) *Modal {
	var confirmName string
	m := &Modal{
		modalType: ModalDeleteIndex,
		indexName: indexName,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Type '%s' to confirm deletion", indexName)).
				Description("This action cannot be undone.").
				Value(&confirmName).
				Validate(func(s string) error {
					if s != indexName {
						return fmt.Errorf("name does not match")
					}
					return nil
				}),
		),
	).WithShowHelp(false).WithShowErrors(true)

	return m
}

func NewCloseIndexModal(indexName string) *Modal {
	m := &Modal{
		modalType: ModalCloseIndex,
		indexName: indexName,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Close index '%s'?", indexName)).
				Description("Closed indices cannot be read or written to.").
				Affirmative("Close").
				Negative("Cancel").
				Value(&m.confirmed),
		),
	).WithShowHelp(false).WithShowErrors(true)

	return m
}

func NewAddAliasModal(indexName string) *Modal {
	m := &Modal{
		modalType: ModalAddAlias,
		indexName: indexName,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Add alias to '%s'", indexName)).
				Placeholder("alias-name").
				Value(&m.aliasName).
				Validate(huh.ValidateNotEmpty()),
		),
	).WithShowHelp(false).WithShowErrors(true)

	return m
}

func NewRemoveAliasModal(indexName string, aliases []string) *Modal {
	m := &Modal{
		modalType: ModalRemoveAlias,
		indexName: indexName,
		aliases:   aliases,
	}

	if len(aliases) == 0 {
		m.form = huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("No aliases").
					Description(fmt.Sprintf("Index '%s' has no aliases to remove.", indexName)),
			),
		).WithShowHelp(false)
	} else {
		options := make([]huh.Option[string], len(aliases))
		for i, a := range aliases {
			options[i] = huh.NewOption(a, a)
		}

		m.form = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("Remove alias from '%s'", indexName)).
					Options(options...).
					Value(&m.aliasName),
			),
		).WithShowHelp(false)
	}

	return m
}

func NewErrorModal(errMsg string) *Modal {
	m := &Modal{
		modalType: ModalError,
		err:       errMsg,
		confirmed: true,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Error").
				Description(errMsg),
			huh.NewConfirm().
				Affirmative("OK").
				Negative("").
				Value(&m.confirmed),
		),
	).WithShowHelp(false)

	return m
}

func (m *Modal) Type() ModalType {
	return m.modalType
}

func (m *Modal) IndexName() string {
	return m.indexName
}

func (m *Modal) Shards() string {
	if m.shards == "" {
		return "1"
	}
	return m.shards
}

func (m *Modal) Replicas() string {
	if m.replicas == "" {
		return "1"
	}
	return m.replicas
}

func (m *Modal) AliasName() string {
	return m.aliasName
}

func (m *Modal) Confirmed() bool {
	return m.confirmed
}

func (m *Modal) Done() bool {
	return m.done
}

func (m *Modal) Cancelled() bool {
	return m.cancelled
}

func (m *Modal) HasAliases() bool {
	return len(m.aliases) > 0
}

func (m *Modal) Init() tea.Cmd {
	return m.form.Init()
}

func (m *Modal) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
			m.cancelled = true
			return nil
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.done = true
	}

	return cmd
}

func (m *Modal) View(width, height int) string {
	boxWidth := 50

	formView := m.form.View()

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(formView)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
