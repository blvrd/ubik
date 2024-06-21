package main

import (
  "fmt"
  "os"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type status int

const (
	todo       status = 1
	inProgress status = 2
	done       status = 3
	wontDo     status = 4
)

type Issue struct {
	id          string
	author      string
	title       string
	description string
	status      status
}

func (i Issue) FilterValue() string {
	return i.title
}

func (i Issue) Title() string {
	return i.title
}

func (i Issue) Description() string {
	return i.description
}

/* MAIN MODEL */

type Model struct {
	list list.Model
	err  error
}

func InitialModel() *Model {
  return &Model{}
}

func (m Model) Init() tea.Cmd {
  return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var cmd tea.Cmd

  switch msg := msg.(type) {
  case tea.WindowSizeMsg:
    m.initList(msg.Width, msg.Height)
  }

  m.list, cmd = m.list.Update(msg)

  return m, cmd
}

func (m Model) View() string {
  return m.list.View()
}

func (m *Model) initList(width, height int) {
  m.list = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
  m.list.SetShowHelp(false)
  m.list.Title = "Issues"
  m.list.SetItems([]list.Item{
    Issue{
      id: "12345",
      author: "garrett@blvrd.co",
      title: "something is wrong with the code",
      status: 1,
    },
    Issue{
      id: "54321",
      author: "garrett@blvrd.co",
      title: "need to complete this task",
      status: 1,
    },
  })
}

func main() {
  p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
  if _, err := p.Run(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
