package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
  "github.com/charmbracelet/log"
  "github.com/blvrd/ubik/entity"
  "github.com/blvrd/ubik/detail"
  "github.com/blvrd/ubik/form"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("105"))

type sessionState int

const (
  detailView sessionState = 1
  formView   sessionState = 2
)

type model struct {
  state        sessionState
  table        table.Model
  issues       []*entity.Issue
  currentIssue *entity.Issue
  detail       detail.Model
  form         form.Model
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

  d := detail.New(&entity.Issue{})
  f := form.New(&entity.Issue{})

  return model{
    table: t,
    detail: d,
    form: f,
    state: detailView,
  }
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
  var cmds []tea.Cmd

  switch msg := msg.(type) {
  case tea.KeyMsg:
    switch msg.String() {
    case "n":
      if m.state != formView {
        m.table.Blur()
        m.state = formView
        m.form = form.New(&entity.Issue{})
        m.form.Init()
      }
    case "enter", "e":
      if m.state != formView {
        m.table.Blur()
        m.state = formView
        m.form = form.New(m.currentIssue)
        m.form.Init()
      }
    case " ":
      if m.state != formView {
        if m.currentIssue.Closed == "true" {
          m.currentIssue.Closed = "false"
        } else {
          m.currentIssue.Closed = "true"
        }

        entity.Update(m.currentIssue)
        return m, GetIssues
      }
    case "d", "backspace":
      if m.state != formView {
        m.currentIssue.Delete()
        return m, GetIssues
      }
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
        m.state = detailView
				m.table.Focus()
			}
		case "q":
      if m.state != formView {
        // formModel, cmd := m.form.Update(msg)
        // m.form = formModel
        // return m, cmd
        return m, tea.Quit
      }

    case "ctrl+c":
			return m, tea.Quit
    }
  case tea.WindowSizeMsg:
    m.table.SetHeight(msg.Height - 5)
  case issuesLoadedMsg:
    var rows []table.Row
    m.issues = msg

    for _, issue := range msg {
      var closed string

      if issue.Closed == "false" {
        closed = "✕"
      } else {
        closed = "✓"
      }

      row := []string{
        issue.Title,
        issue.Author,
        closed,
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

    m.table, cmd = m.table.Update(msg)

    if len(m.issues) > 0 {
      m.currentIssue = m.issues[m.table.Cursor()]
      d := detail.New(m.currentIssue)
      m.detail = d
    }

    return m, cmd
  case form.FormCompletedMsg:
    m.state = detailView
    m.table.Focus()
    return m, GetIssues
  }

	m.table, cmd = m.table.Update(msg)
  cmds = append(cmds, cmd)

  if m.state == formView {
    m.form, cmd = m.form.Update(msg)
    cmds = append(cmds, cmd)
  }

  if len(m.issues) > 0 {
    m.currentIssue = m.issues[m.table.Cursor()]
    d := detail.New(m.currentIssue)
    m.detail = d
  }
  return m, tea.Batch(cmds...)
}

func (m model) View() string {
  table := baseStyle.Render(m.table.View())
  currentIssue := m.currentIssue

  if currentIssue == nil {
    currentIssue = &entity.Issue{Id: "none"}
  }

  var sidebarView string

  if m.state == detailView {
    sidebarView = m.detail.View()
  } else {
    sidebarView = m.form.View()
  }

  view := lipgloss.JoinHorizontal(lipgloss.Top, table, sidebarView)

  return view
}

func Run() error {
  p := tea.NewProgram(NewModel(), tea.WithAltScreen())

  if _, err := p.Run(); err != nil {
    log.Error(err)
    return err
  }

  return nil
}
