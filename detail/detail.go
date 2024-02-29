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
  entMap := m.ent.ToMap()

  var closed string

  if entMap["closed"] == "false" {
    closed = "[ ]"
  } else {
    closed = "[x]"
  }

  s.WriteString(fmt.Sprintf("ID: %s\n", entMap["id"]))
  s.WriteString(fmt.Sprintf("Title: %s\n", entMap["title"]))
  s.WriteString(fmt.Sprintf("Author: %s\n\n", entMap["author"]))
  s.WriteString(fmt.Sprintf("Closed: %s\n\n", closed))
  s.WriteString(fmt.Sprintf("Description: %s\n\n", entMap["description"]))
  s.WriteString(fmt.Sprintf("Created: %s\n\n", entMap["created_at"]))
  s.WriteString(fmt.Sprintf("Updated: %s\n\n", entMap["updated_at"]))

  return s.String()
}
