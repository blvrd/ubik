package tui

import (
  "fmt"
	"github.com/blvrd/ubik/detail"
	"github.com/blvrd/ubik/entity"
	"github.com/blvrd/ubik/form"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("105")).
	Margin(2, 2)

type focusedView int

const (
	listView focusedView = 1
	formView focusedView = 2
)

type model struct {
	focusState   focusedView
	list         list.Model
	issues       []*entity.Issue
	currentIssue *entity.Issue
	detail       detail.Model
	form         form.Model
	loading      bool
}

type li struct {
  title, desc, closed string
}
func (i li) Title() string {
  var closed string

  if i.closed == "true" {
    closed = "✓"
  } else {
    closed = "✕"
  }
  return fmt.Sprintf("%s (%s)", i.title, closed)
}
func (i li) Description() string { return i.desc }
func (i li) FilterValue() string { return i.title }

func NewModel() tea.Model {
	d := detail.New(&entity.Issue{})
	f := form.New(&entity.Issue{})
  l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
  l.Title = "Issues"

	return model{
		list:       l,
		detail:     d,
		form:       f,
		focusState: listView,
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

func handleListViewMsg(m model, msg tea.Msg) (model, []tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

  if m.list.SettingFilter() {
    m.list, cmd = m.list.Update(msg)
    if len(m.issues) > 0 {
      m.currentIssue = m.issues[m.list.Index()]
      d := detail.New(m.currentIssue)
      m.detail = d
    }
    cmds = append(cmds, cmd)

    return m, cmds
  }

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			m.focusState = formView
			m.form = form.New(&entity.Issue{})
			m.form.Init()
		case "enter", "e":
			m.focusState = formView
			m.form = form.New(m.currentIssue)
			m.form.Init()
		case " ":
      log.Infof("current issue: %+v", m.currentIssue)
			if m.currentIssue.Closed == "true" {
        m.currentIssue.Open()
			} else {
        m.currentIssue.Close()
			}

			cmds = append(cmds, GetIssues)
			return m, cmds
		case "d", "backspace":
      m.currentIssue.Delete()
      cmds = append(cmds, GetIssues)
      return m, cmds
		case "q", "ctrl+c":
			cmds = append(cmds, tea.Quit)
			return m, cmds
    case "r":
      m.currentIssue.Restore()
      cmds = append(cmds, GetIssues)
      return m, cmds
		}
	case tea.WindowSizeMsg:
		_, y := baseStyle.GetFrameSize()
		m.list.SetSize(msg.Width / 2, msg.Height-y)
	case issuesLoadedMsg:
		var items []list.Item
		m.issues = msg

		for _, issue := range msg {
      item := li{
        title: issue.Title,
        desc: issue.Description,
        closed: issue.Closed,
      }
			items = append(items, item)
		}

		m.list, cmd = m.list.Update(items)

		if len(m.issues) > 0 {
			m.currentIssue = m.issues[m.list.Index()]
			m.list.SetItems(items)
			d := detail.New(m.currentIssue)
			m.detail = d
		}

		cmds = append(cmds, cmd)
		return m, cmds
	}

	m.list, cmd = m.list.Update(msg)
	if len(m.issues) > 0 {
    m.currentIssue = m.issues[m.list.Index()]
    d := detail.New(m.currentIssue)
    m.detail = d
  }
	cmds = append(cmds, cmd)

	return m, cmds
}

func handleFormViewMsg(m model, msg tea.Msg) (model, []tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
    case "esc":
      m.focusState = listView
    case "ctrl+c":
      cmds = append(cmds, tea.Quit)
		}
	case form.FormCompletedMsg:
    log.Info("form completed message")
		m.focusState = listView
		cmds = append(cmds, GetIssues)
		return m, cmds
	}

  log.Info("updating form")
	m.form, cmd = m.form.Update(msg)
	cmds = append(cmds, cmd)
	return m, cmds
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m.focusState {
	case listView:
		m, cmds = handleListViewMsg(m, msg)
	case formView:
		m, cmds = handleFormViewMsg(m, msg)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	list := baseStyle.Render(m.list.View())
	currentIssue := m.currentIssue

	if currentIssue == nil {
		currentIssue = &entity.Issue{Id: "none"}
	}

	var sidebarView string

	if m.focusState == listView {
		sidebarView = baseStyle.Render(m.detail.View())
	} else {
		sidebarView = baseStyle.Render(m.form.View())
	}

	view := lipgloss.JoinHorizontal(lipgloss.Top, list, sidebarView)

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
