package tui

import (
	"fmt"
	// "strings"
	//  "time"

	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/charmbracelet/bubbles/table"
// 	"github.com/charmbracelet/lipgloss"
//   "github.com/google/uuid"
  "github.com/charmbracelet/log"
)

type model struct {
  message string
}

func NewModel() tea.Model {
  return model{message: "world"}
}


func (m model) Init() tea.Cmd {
  return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c", "q", "esc":
      return m, tea.Quit
    }
  }
  return m, nil
}

func (m model) View() string {
  return fmt.Sprintf("hello, %s", m.message)
}

func Run() error {
  p := tea.NewProgram(NewModel())

  if _, err := p.Run(); err != nil {
    log.Error(err)
    return err
  }

  return nil
}
