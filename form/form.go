package form

import (
	"fmt"
	"strings"
	// "time"

	"github.com/blvrd/ubik/entity"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	// "github.com/charmbracelet/log"
)

type Model struct {
	issue     *entity.Issue
	form      *huh.Form
	persisted bool
}

func New(issue *entity.Issue) Model {
	title := issue.Title
	description := issue.Description

	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("title").
				Title("Title").
				Value(&title).
				CharLimit(100),
			huh.NewText().
				Key("description").
				Value(&description).
				Title("Description").
				CharLimit(600),
			huh.NewConfirm().
				Key("done").
				Title("All done?").
				Validate(func(v bool) error {
					if !v {
						return fmt.Errorf("Welp, finish up then")
					}
					return nil
				}).
				Affirmative("Save").
				Negative("Wait, no"),
		),
	).
		WithWidth(60).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(&huh.KeyMap{
			Input: huh.InputKeyMap{
				AcceptSuggestion: key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "complete")),
				Prev:             key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
				Next:             key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
				Submit:           key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "submit")),
			},
			Text: huh.TextKeyMap{
				Next:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
				Prev:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
				Submit:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "submit")),
				NewLine: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "new line")),
				Editor:  key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "open editor")),
			},
			Confirm: huh.ConfirmKeyMap{
				Prev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
				Next:   key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "next")),
				Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
				Toggle: key.NewBinding(key.WithKeys("h", "l", "right", "left"), key.WithHelp("←/→", "toggle")),
			},
		})

	return Model{
		issue: issue,
		form:  f,
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

type FormCompletedMsg string

func CompleteForm(m Model) tea.Cmd {
	return func() tea.Msg {
		title := m.form.GetString("title")
		description := m.form.GetString("description")

		m.issue.Title = title
		m.issue.Description = description

		if m.issue.IsPersisted() {
			entity.Update(m.issue)
		} else {
			entity.Add(m.issue)
		}

		return FormCompletedMsg("Form is complete")
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		// Don't try to submit the same completed form twice
		if !m.persisted {
			cmds = append(cmds, CompleteForm(m))
			m.persisted = true
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(m.form.View())

	return s.String()
}
