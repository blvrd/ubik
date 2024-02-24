package tui

import (
	"fmt"
	// "strings"
	//  "time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
//   "github.com/google/uuid"
  "github.com/charmbracelet/log"
  "github.com/blvrd/ubik/entity"
  "github.com/blvrd/ubik/detail"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("105"))

type model struct {
  table        table.Model
  issues       []*entity.Issue
  currentIssue *entity.Issue
  detailView   detail.Model
  loading      bool
}

func NewModel() tea.Model {
	columns := []table.Column{
		{Title: "Title", Width: 20},
		{Title: "Author", Width: 20},
		{Title: "Closed", Width: 10},
		{Title: "Created", Width: 20},
		{Title: "Updated", Width: 20},
	}

  rows := []table.Row{
    []string{"Loading..."},
  }

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

  return model{table: t}
}

type issuesLoadedMsg []*entity.Issue

func GetIssues() tea.Msg {
  refPath := entity.IssuesPath
  notes := entity.GetNotes(refPath)
  issues := entity.IssuesFromGitNotes(notes)

  return issuesLoadedMsg(issues)
}

func (m model) Init() tea.Cmd {
  return GetIssues
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var cmd tea.Cmd

  switch msg := msg.(type) {
  case tea.KeyMsg:
    switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
    }
  case tea.WindowSizeMsg:
    m.table.SetHeight(msg.Height - 5)
  case issuesLoadedMsg:
    var rows []table.Row
    m.issues = msg

    for _, issue := range msg {
      row := []string{
        issue.Title,
        issue.Author,
        issue.Closed,
        issue.CreatedAt.String(),
        issue.UpdatedAt.String(),
      }

      rows = append(rows, row)
    }

    if len(rows) == 0 {
      rows = []table.Row{
        []string{"No issues found"},
      }
    }

    m.table.SetRows(rows)
    m.loading = false

    tea.Printf("%+v\n", m.table.Rows())
    m.table, cmd = m.table.Update(msg)

    if len(m.issues) > 0 {
      m.currentIssue = m.issues[m.table.Cursor()]
      // m.currentIssue = m.issues[0]
    }

    return m, cmd
  }

	m.table, cmd = m.table.Update(msg)
  if len(m.issues) > 0 {
    m.currentIssue = m.issues[m.table.Cursor()]
    // m.currentIssue = m.issues[0]
  }
  return m, nil
}

func (m model) View() string {
  table := baseStyle.Render(m.table.View())
  currentIssue := m.currentIssue

  if currentIssue == nil {
    currentIssue = &entity.Issue{Id: "none"}
  }

  view := lipgloss.JoinHorizontal(lipgloss.Top, table, fmt.Sprintf("%s", currentIssue.Id))
  // view := lipgloss.JoinHorizontal(lipgloss.Top, table, "hi")
  return view
}

func Run() error {
  p := tea.NewProgram(NewModel(), tea.WithAltScreen())
  // p := tea.NewProgram(NewModel())

  if _, err := p.Run(); err != nil {
    log.Error(err)
    return err
  }

  return nil
}
