package detail

import (
	"fmt"
	"strings"
  "time"

	"github.com/blvrd/ubik/entity"
	"github.com/charmbracelet/lipgloss"
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
		closed = lipgloss.NewStyle().Foreground(lipgloss.Color("#838383")).Render("[·] Open")
  } else {
    closed = lipgloss.NewStyle().Foreground(lipgloss.Color("#5db158")).Render("[✓] Closed")
  }

  title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s", entMap["title"]))
  shortcode := lipgloss.NewStyle().Foreground(lipgloss.Color("#838383")).Render(fmt.Sprintf("#%s", entMap["shortcode"]))
  author := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(fmt.Sprintf("%s", entMap["author"]))
  createdAt := entMap["created_at"].(time.Time).Format(time.RFC822)

  s.WriteString(fmt.Sprintf("%s %s\n\n", title, shortcode))
  s.WriteString(fmt.Sprintf("%s - 3 comments\n", closed))
  s.WriteString(fmt.Sprintf("opened by %s on %s\n\n", author, createdAt))
  s.WriteString(fmt.Sprintf("%s", entMap["description"]))
  // s.WriteString(fmt.Sprintf("Author: %s\n\n", entMap["author"]))
  // s.WriteString(fmt.Sprintf("Closed: %s\n\n", closed))
  // s.WriteString(fmt.Sprintf("Description: %s\n\n", entMap["description"]))
  // s.WriteString(fmt.Sprintf("Created: %s\n\n", entMap["created_at"]))
  // s.WriteString(fmt.Sprintf("Updated: %s\n\n", entMap["updated_at"]))
  // s.WriteString(fmt.Sprintf("Deleted: %s\n\n", entMap["deleted_at"]))

  return s.String()
}
