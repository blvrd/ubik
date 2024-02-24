package detail

import (
	"fmt"
	"strings"

	"github.com/blvrd/ubik/entity"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
  ent entity.Entity
}

func New(ent entity.Entity) Model {
  return Model{ent: ent}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  return m, nil
}

func (m Model) View() string {
  var s strings.Builder

  json, err := m.ent.Marshal()

  if err != nil {
    return "error"
  }

  s.WriteString(fmt.Sprintf("ID: %s\n\n", m.ent.GetId()))
  s.WriteString(fmt.Sprintf("data: %s\n", string(json)))

  return s.String()
}
