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
	"io"
	"strings"
	"time"
  "os/exec"
)

var baseStyle = lipgloss.NewStyle().Margin(2, 2)

var borderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#333333")).
	BorderRight(true).
	MarginRight(3)

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
	id        string
	author    string
	title     string
	desc      string
	shortcode string
	closedAt  time.Time
	createdAt time.Time
}

func (i li) Id() string { return i.id }
func (i li) Title() string {
	var closed string

	if i.closedAt.IsZero() {
		closed = lipgloss.NewStyle().Foreground(lipgloss.Color("#838383")).Render("[·]")
	} else {
		closed = lipgloss.NewStyle().Foreground(lipgloss.Color("#5db158")).Render("[✓]")
	}
	return fmt.Sprintf("%s %s", closed, i.title)
}
func (i li) Description() string {
	return fmt.Sprintf("#%s opened %s by %s", i.shortcode, i.createdAt.Format(time.RFC822), i.author)
}
func (i li) FilterValue() string { return i.title }

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().MarginLeft(2)
	selectedItemStyle = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 2 }
func (d itemDelegate) Spacing() int                            { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(li)
	if !ok {
		return
	}

	str := i.Title()
	str += "\n"
	str += i.Description()

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func NewModel() tea.Model {
	issue := entity.NewIssue()
	d := detail.New(&issue)
	formMode := form.FormMode{Mode: "new"}
	f := form.New(&issue, formMode)
	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
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
	notes, err := entity.GetNotes(refPath)

	if err != nil {
		log.Fatal(err)
	}
	issues := entity.IssuesFromGitNotes(notes)

	return issuesLoadedMsg(issues)
}

func CheckIssueClosuresFromCommits() tea.Msg {
	refPath := entity.IssuesPath
	notes, err := entity.GetNotes(refPath)

	if err != nil {
		log.Fatal(err)
	}
	issues := entity.IssuesFromGitNotes(notes)

  cmdArgs := []string{"log"}

  for _, issue := range issues {
    closes := fmt.Sprintf("closes %s", issue.Shortcode())
    cmdArgs = append(cmdArgs, "--grep", closes)
  }

  cmdArgs = append(cmdArgs, "-i", "--pretty=format:'%h'")

  cmd := exec.Command(
    "git",
    cmdArgs...,
  )
	bytes, err := cmd.Output()
	if err != nil {
		panic(err)
	}

  log.Debug(string(bytes))

	return issuesLoadedMsg(issues)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(GetIssues, CheckIssueClosuresFromCommits)
}

func handleListViewMsg(m model, msg tea.Msg) (model, []tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	newIssue := entity.NewIssue()

	if !m.list.SettingFilter() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+n":
				m.focusState = formView
        log.Debug("hi")
				formMode := form.FormMode{Mode: "new"}
				m.form = form.New(&newIssue, formMode)
				m.form.Init()
			case "ctrl+d":
				m.focusState = formView
				newIssue.Title = m.currentIssue.Title
				newIssue.Description = m.currentIssue.Description
				currentShortcode := m.currentIssue.Shortcode()
				formMode := form.FormMode{Mode: "duplicating", Shortcode: &currentShortcode}
				m.form = form.New(&newIssue, formMode)
				m.form.Init()
			case "enter", "ctrl+e":
				m.focusState = formView
				formMode := form.FormMode{Mode: "editing"}
				m.form = form.New(m.currentIssue, formMode)
				m.form.Init()
			case " ":
				if m.currentIssue.ClosedAt.IsZero() {
					m.currentIssue.Close()
				} else {
					m.currentIssue.Open()
				}

				cmds = append(cmds, GetIssues)
				return m, cmds
			case "backspace":
				m.currentIssue.Delete()
				cmds = append(cmds, GetIssues)
				return m, cmds
			case "ctrl+q", "ctrl+c":
				cmds = append(cmds, tea.Quit)
				return m, cmds
			case "ctrl+r":
				m.currentIssue.Restore()
				cmds = append(cmds, GetIssues)
				return m, cmds
			}
		case tea.WindowSizeMsg:
			_, y := baseStyle.GetFrameSize()
			m.list.SetSize(90, msg.Height-y)
		case issuesLoadedMsg:
			var items []list.Item
			m.issues = msg

			for _, issue := range msg {
				item := li{
					id:        issue.Id,
					author:    issue.Author,
					title:     issue.Title,
					desc:      issue.Description,
					closedAt:  issue.ClosedAt,
					shortcode: issue.Shortcode(),
					createdAt: issue.CreatedAt,
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
	}

	m.list, cmd = m.list.Update(msg)
	if len(m.issues) > 0 && m.list.SelectedItem() != nil {
		selectedItem := m.list.SelectedItem().(li)
		currentIssue := entity.NewIssue()

		// This would be simpler/faster as a map access
		for _, issue := range m.issues {
			if issue.Id == selectedItem.Id() {
				currentIssue = *issue
				break
			}
		}

		m.currentIssue = &currentIssue
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
		m.focusState = listView
		cmds = append(cmds, GetIssues)
		return m, cmds
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
	list := borderStyle.Width(m.list.Width()).Render(m.list.View())
	currentIssue := m.currentIssue

	if currentIssue == nil {
		currentIssue = &entity.Issue{Id: "none"}
	}

	var sidebarView string

	if m.focusState == listView {
		sidebarView = lipgloss.NewStyle().Width(60).Render(m.detail.View())
	} else {
		sidebarView = m.form.View()
	}

	view := baseStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, list, sidebarView))

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
