package form

import (
	"strings"

	"github.com/blvrd/ubik/entity"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type Model struct {
  ent entity.Entity
  titleInput textinput.Model
}

func New(ent entity.Entity) Model {
  ti := textinput.New()
  ti.CharLimit = 80
  ti.Width = 30
  ti.Focus()

  return Model{
    ent: ent,
    titleInput: ti,
  }
}

func (m Model) Init() tea.Cmd {
  return textinput.Blink
}

type FormCancelMsg string

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
  var cmd tea.Cmd


  // switch msg := msg.(type) {
  // }

  m.titleInput, cmd = m.titleInput.Update(msg)
  log.Infof("%+v", m.titleInput)

  return m, cmd
}

func (m Model) View() string {
  var s strings.Builder

  s.WriteString(m.titleInput.View())

  return s.String()
}
