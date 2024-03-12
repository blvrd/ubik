package tui

import (
	"github.com/blvrd/ubik/detail"
	"github.com/blvrd/ubik/entity"
	"github.com/blvrd/ubik/form"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
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
	table        table.Model
	list         list.Model
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
		table:      t,
		list:       list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
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
			if m.currentIssue.Closed == "true" {
				m.currentIssue.Closed = "false"
			} else {
				m.currentIssue.Closed = "true"
			}

			entity.Update(m.currentIssue)
			cmds = append(cmds, GetIssues)
			return m, cmds
		case "d", "backspace":
			if m.list.FilterState() != list.Filtering {
				m.currentIssue.Delete()
				cmds = append(cmds, GetIssues)
				return m, cmds
			}
		case "q", "ctrl+c":
			cmds = append(cmds, tea.Quit)
			return m, cmds
		}
	case tea.WindowSizeMsg:
		h, v := baseStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case issuesLoadedMsg:
		var items []list.Item
		m.issues = msg

		for _, issue := range msg {
			items = append(items, issue)
		}

		if len(items) == 0 {
			items = []list.Item{}
		}

		m.loading = false

		m.list, cmd = m.list.Update(msg)

		if len(m.issues) > 0 {
			m.currentIssue = m.issues[m.list.Cursor()]
			m.list.SetItems(items)
			d := detail.New(m.currentIssue)
			m.detail = d
		}

		cmds = append(cmds, cmd)
		return m, cmds
	case form.FormCompletedMsg:
		m.focusState = listView
		cmds = append(cmds, GetIssues)
		return m, cmds
	}

	m.list, cmd = m.list.Update(msg)
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
	}

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
