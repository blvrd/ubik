package tui

import (
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/blvrd/ubik/detail"
	"github.com/blvrd/ubik/entity"
	"github.com/blvrd/ubik/form"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var baseStyle = lipgloss.NewStyle().Margin(2, 2)

var borderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#333333")).
	BorderRight(true).
	MarginRight(3)

type focusedView int

const (
	issuesListView   focusedView = 1
	issuesFormView   focusedView = 2
	memosListView    focusedView = 3
	projectsListView focusedView = 4
)

type model struct {
	focusState    focusedView
	issuesList    list.Model
	issues        map[string]*entity.Issue
	currentIssue  *entity.Issue
	memosList     list.Model
	memos         []*entity.Memo
	projects      []*entity.Project
	checks        []*entity.Check
	details       map[string]*detail.Model
	currentDetail *detail.Model
	form          form.Model
	loading       bool
	totalWidth    int
	totalHeight   int
}

func (m model) WidthByPercentage(percentage float64) int {
	return int(float64(m.totalWidth) * percentage)
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
	d.Init()
	details := make(map[string]*detail.Model)
	details[issue.Id] = &d
	formMode := form.FormMode{Mode: "new"}
	f := form.New(&issue, formMode)
	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	l.Title = "Issues"

	return model{
		issuesList:    l,
		memosList:     list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		currentDetail: &d,
		details:       details,
		form:          f,
		focusState:    issuesListView,
	}
}

type issuesLoadedMsg struct {
	Issues         map[string]*entity.Issue
	FocusedIssueId *string
}

func LoadIssues(focusedIssueId *string, issues map[string]*entity.Issue) tea.Cmd {
	return func() tea.Msg {
		return issuesLoadedMsg{
			Issues:         issues,
			FocusedIssueId: focusedIssueId,
		}
	}
}

func GetIssues(focusedIssueId *string) tea.Cmd {
	return func() tea.Msg {
		var issues map[string]*entity.Issue
		// do IO
		log.Debug("🪚 getting issues")

		return issuesLoadedMsg{
			Issues:         issues,
			FocusedIssueId: focusedIssueId,
		}
	}
}

func CheckIssueClosuresFromCommits() tea.Msg {
	refPath := entity.IssuesPath
	notes, err := entity.GetNotes(refPath)

	if err != nil {
		log.Fatal(err)
	}

	issues := entity.IssuesFromGitNotes(notes)

	for _, issue := range issues {
		if issue.Shortcode() == "" || !issue.ClosedAt.IsZero() {
			continue
		}
		closes := fmt.Sprintf("closes #%s", issue.Shortcode())

		cmd := exec.Command(
			"git",
			"log",
			"--grep",
			closes,
			"-i",
			"--pretty=format:%h",
		)

		bytes, err := cmd.Output()
		if err != nil {
			panic(err)
		}

		if len(bytes) != 0 {
			issue.CloseWithComment(fmt.Sprintf("closed by: %s", string(bytes)))
		}
	}

	notes, err = entity.GetNotes(refPath)

	if err != nil {
		log.Fatal(err)
	}

	issues = entity.IssuesFromGitNotes(notes)
  issueMap := make(map[string]*entity.Issue)
  for _, issue := range issues {
    issueMap[issue.Id] = issue
  }
	return issuesLoadedMsg{
		Issues:         issueMap,
		FocusedIssueId: nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(GetIssues(nil), CheckIssueClosuresFromCommits)
}

func handleListViewMsg(m model, msg tea.Msg) (model, []tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	newIssue := entity.NewIssue()

	if !m.issuesList.SettingFilter() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+n":
				m.focusState = issuesFormView
				formMode := form.FormMode{Mode: "new"}
				m.form = form.New(&newIssue, formMode)
				m.form.Init()
			case "m":
				m.focusState = memosListView
				return m, cmds
			case "ctrl+d":
				m.focusState = issuesFormView
				newIssue.Title = m.currentIssue.Title
				newIssue.Description = m.currentIssue.Description
				currentShortcode := m.currentIssue.Shortcode()
				formMode := form.FormMode{Mode: "duplicating", Shortcode: &currentShortcode}
				m.form = form.New(&newIssue, formMode)
				m.form.Init()
			case "enter", "ctrl+e":
				m.focusState = issuesFormView
				formMode := form.FormMode{Mode: "editing"}
				m.form = form.New(m.currentIssue, formMode)
				m.form.Init()
				return m, cmds
			case " ":
				if m.currentIssue.ClosedAt.IsZero() {
					m.currentIssue.Close()
				} else {
					m.currentIssue.Open()
				}

				cmds = append(cmds, GetIssues(&m.currentIssue.Id))
				return m, cmds
			case "backspace":
				m.currentIssue.Delete()

				// deleting the last issue
				if len(m.issues) == 1 {
					cmds = append(cmds, GetIssues(nil))
					return m, cmds
				}

				prevIndex := float64((m.issuesList.Index() - 1))
				prevIssue := m.issuesList.Items()[int(math.Max(0, prevIndex))].(li).Id()
				cmds = append(cmds, GetIssues(&prevIssue))
				return m, cmds
			case "ctrl+q", "ctrl+c":
				cmds = append(cmds, tea.Quit)
				return m, cmds
			case "ctrl+r":
				m.currentIssue.Restore()
				cmds = append(cmds, GetIssues(&m.currentIssue.Id))
				return m, cmds
			}
		case tea.WindowSizeMsg:
			x, y := baseStyle.GetFrameSize()
			m.totalWidth = msg.Width
			m.totalHeight = msg.Width

			log.Debugf("🪚 framesize: %#v %#v", x, y)
			m.issuesList.SetHeight(msg.Height - y)
			// m.issuesList.SetSize(90, msg.Height-y)
			// m.memosList.SetSize(90, msg.Height-y)
		case issuesLoadedMsg:
			var items []list.Item
			m.issues = msg.Issues

			for _, issue := range m.issues {
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

			m.issuesList, cmd = m.issuesList.Update(items)

			if len(m.issues) > 0 {
        // var focusedIssueIndex int
        // for i, issue := range m.issues {
        //   if issue.Id == *msg.FocusedIssueId {
        //     focusedIssueIndex = i
        //   }
        // }

        if msg.FocusedIssueId == nil {
          m.currentIssue = m.issues[items[0].(li).Id()]
        } else {
          m.currentIssue = m.issues[*msg.FocusedIssueId]
        }
        var idx int
        for i, item := range m.issuesList.Items() {
          if item.(li).Id() == *msg.FocusedIssueId {
            idx = i
          }
        }
				m.issuesList.Select(idx)
				m.issuesList.SetItems(items)
				d := detail.New(m.currentIssue)
				d.Init()
				m.details[m.currentIssue.Id] = &d
				m.currentDetail = &d
			}

			cmds = append(cmds, cmd)
			return m, cmds
		}
	}

	m.issuesList, cmd = m.issuesList.Update(msg)
	if len(m.issues) > 0 && m.issuesList.SelectedItem() != nil {
		selectedItem := m.issuesList.SelectedItem().(li)
		currentIssue := entity.NewIssue()

		// This would be simpler/faster as a map access
		for _, issue := range m.issues {
			if issue.Id == selectedItem.Id() {
				currentIssue = *issue
				break
			}
		}

		m.currentIssue = &currentIssue
		d := m.details[m.currentIssue.Id]
		if d == nil {
			newDetail := detail.New(m.currentIssue)
			d = &newDetail
			d.Init()
			m.details[m.currentIssue.Id] = d
		}
		m.currentDetail = d
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
			m.focusState = issuesListView
		case "ctrl+c":
			cmds = append(cmds, tea.Quit)
		}
	case form.FormCompletedMsg:
		m.focusState = issuesListView
		var issues map[string]*entity.Issue
		issues = m.issues
		issues[msg.Id] = msg
		cmd := LoadIssues(&msg.Id, issues)
		cmds = append(cmds, cmd)
		return m, cmds
	case form.FormCancelledMsg:
		m.focusState = issuesListView
		return m, nil
	}

	m.form, cmd = m.form.Update(msg)
	cmds = append(cmds, cmd)
	return m, cmds
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd []tea.Cmd

	switch m.focusState {
	case issuesListView:
		m, cmds = handleListViewMsg(m, msg)
	case issuesFormView:
		m, cmds = handleFormViewMsg(m, msg)
	}

	*m.currentDetail, cmd = m.currentDetail.Update(msg)

	cmds = append(cmds, cmd...)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	list := borderStyle.Width(m.WidthByPercentage(0.5)).Render(m.issuesList.View())
	currentIssue := m.currentIssue

	if currentIssue == nil {
		currentIssue = &entity.Issue{Id: "none"}
	}

	var sidebarView string

	switch m.focusState {
	case issuesListView:
		sidebarView = lipgloss.NewStyle().Render(m.currentDetail.View())
	case issuesFormView:
		sidebarView = m.form.View()
	case memosListView:
		sidebarView = ""
		list = borderStyle.Width(m.memosList.Width()).Render(m.memosList.View())
	default:
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
