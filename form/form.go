package form

import (
	"fmt"
	"strings"
	"time"

	"github.com/blvrd/ubik/entity"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
)

type Model struct {
	ent  entity.Entity
	form *huh.Form
}

func New(ent entity.Entity) Model {
	title := ent.(*entity.Issue).Title
	description := ent.(*entity.Issue).Description

	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("title").
				Title("Title").
				Value(&title).
				CharLimit(50),
			huh.NewText().
				Key("description").
				Value(&description).
				Title("Description"),
			huh.NewConfirm().
				Key("done").
				Title("All done?").
				Validate(func(v bool) error {
					if !v {
						return fmt.Errorf("Welp, finish up then")
					}
					return nil
				}).
				Affirmative("Yep").
				Negative("Wait, no"),
		),
	).
		WithWidth(60).
		WithShowHelp(false).
		WithShowErrors(false)

	return Model{
		ent:  ent,
		form: f,
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

type FormCompletedMsg string

func CompleteForm(m Model) tea.Cmd {
	var id string
	var createdAt time.Time
	if m.ent.GetId() != "" {
		id = m.ent.GetId()
		createdAt = m.ent.(*entity.Issue).CreatedAt
	} else {
		id = uuid.NewString()
		createdAt = time.Now().UTC()
	}

	return func() tea.Msg {
		issue := entity.Issue{
			Id:          id,
			Author:      entity.GetAuthorEmail(),
			Title:       m.form.GetString("title"),
			Description: m.form.GetString("description"),
			Closed:      "false",
			RefPath:     entity.IssuesPath,
			CreatedAt:   createdAt,
			UpdatedAt:   time.Now().UTC(),
		}
		entity.Add(&issue)

		return FormCompletedMsg("Form is complete")
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// switch msg := msg.(type) {
	// case tea.KeyMsg:
	//   switch msg.String() {
	//   }
	// }

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		cmds = append(cmds, CompleteForm(m))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(m.form.View())

	return s.String()
}
