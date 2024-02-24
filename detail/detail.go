package detail

import (
  "fmt"
  tea "github.com/charmbracelet/bubbletea"
  "github.com/blvrd/ubik/entity"
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
  return fmt.Sprintf("%+v", m.ent)
}
