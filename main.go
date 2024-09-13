package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/blvrd/ubik/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

type Router struct {
	routes map[int]func(Model, tea.Msg) (Model, tea.Cmd)
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[int]func(Model, tea.Msg) (Model, tea.Cmd)),
	}
}

func (r *Router) Route(m Model, msg tea.Msg) (Model, tea.Cmd) {
	if handler, ok := r.routes[m.path]; ok {
		return handler(m, msg)
	}

	// Default handler if no route is found
	return m, nil
}

func (r *Router) AddRoute(path int, handler func(Model, tea.Msg) (Model, tea.Cmd)) {
	r.routes[path] = handler
}

type checkPersistedMsg struct {
	Check      Check
	IsNewCheck bool
}

func persistCheck(check Check) tea.Cmd {
	return func() tea.Msg {
		jsonData, err := json.Marshal(check)
		if err != nil {
			debug("%#v", err.Error())
			return err
		}

		cmd := exec.Command("git", "hash-object", "--stdin", "-w")
		cmd.Stdin = bytes.NewReader(jsonData)

		b, err := cmd.Output()
		if err != nil {
			debug("%#v", err.Error())
			return err
		}

		hash := strings.TrimSpace(string(b))

		// #nosec G204
		cmd = exec.Command("git", "update-ref", fmt.Sprintf("refs/ubik/checks/%s", check.Id), hash)
		err = cmd.Run()

		if err != nil {
			debug("%#v", err.Error())
			panic(err)
		}
		return checkPersistedMsg{Check: check}
	}
}

type issuePersistedMsg struct {
	Issue          Issue
	IsNewIssue     bool
	ScrollToBottom bool
}

func persistIssue(issue Issue) tea.Cmd {
	return func() tea.Msg {
		var newIssue bool

		if issue.Id == "" {
			newIssue = true
		} else {
			newIssue = false
		}

		if newIssue {
			id := uuid.NewString()
			shortcode := StringToShortcode(id)
			issue.Id = id
			issue.Shortcode = shortcode
			issue.CreatedAt = time.Now().UTC()
		}
		issue.UpdatedAt = time.Now().UTC()

		var issueHasNewComment bool

		for i, comment := range issue.Comments {
			if comment.CreatedAt.IsZero() {
				issueHasNewComment = true
				comment.CreatedAt = time.Now().UTC()
				comment.UpdatedAt = time.Now().UTC()
				issue.Comments[i] = comment
			}
		}

		var scrollToBottom bool
		if issueHasNewComment {
			scrollToBottom = true
		}

		jsonData, err := json.Marshal(issue)
		if err != nil {
			return err
		}

		cmd := exec.Command("git", "hash-object", "--stdin", "-w")
		cmd.Stdin = bytes.NewReader(jsonData)

		b, err := cmd.Output()
		if err != nil {
			return err
		}

		hash := strings.TrimSpace(string(b))

		// #nosec G204
		cmd = exec.Command("git", "update-ref", fmt.Sprintf("refs/ubik/issues/%s", issue.Id), hash)
		err = cmd.Run()

		if err != nil {
			log.Fatalf("%#v", err.Error())
			panic(err)
		}

		return issuePersistedMsg{
			Issue:          issue,
			IsNewIssue:     newIssue,
			ScrollToBottom: scrollToBottom,
		}
	}
}

const (
	issuesIndexPath = iota
	issuesShowPath
	issuesCommentContentPath
	issuesCommentConfirmationPath
	issuesEditTitlePath
	issuesEditLabelsPath
	issuesEditDescriptionPath
	issuesNewTitlePath
	issuesEditConfirmationPath
	issuesNewLabelsPath
	issuesNewDescriptionPath
	issuesNewConfirmationPath
	checksIndexPath
	checksShowPath
)

func matchRoute(currentRoute, route int) bool {
	return route == currentRoute
}

type Styles struct {
	Theme Theme
}

type Theme struct {
	SelectedBackground lipgloss.AdaptiveColor
	PrimaryBorder      lipgloss.AdaptiveColor
	FaintBorder        lipgloss.AdaptiveColor
	SecondaryBorder    lipgloss.AdaptiveColor
	FaintText          lipgloss.AdaptiveColor
	PrimaryText        lipgloss.AdaptiveColor
	SecondaryText      lipgloss.AdaptiveColor
	InvertedText       lipgloss.AdaptiveColor
	GreenText          lipgloss.AdaptiveColor
	YellowText         lipgloss.AdaptiveColor
	RedText            lipgloss.AdaptiveColor
}

func DefaultStyles() Styles {
	return Styles{
		Theme: Theme{
			PrimaryBorder:      lipgloss.AdaptiveColor{Light: "013", Dark: "008"},
			SecondaryBorder:    lipgloss.AdaptiveColor{Light: "008", Dark: "007"},
			SelectedBackground: lipgloss.AdaptiveColor{Light: "006", Dark: "008"},
			FaintBorder:        lipgloss.AdaptiveColor{Light: "254", Dark: "000"},
			PrimaryText:        lipgloss.AdaptiveColor{Light: "000", Dark: "015"},
			SecondaryText:      lipgloss.AdaptiveColor{Light: "244", Dark: "251"},
			FaintText:          lipgloss.AdaptiveColor{Light: "007", Dark: "245"},
			InvertedText:       lipgloss.AdaptiveColor{Light: "015", Dark: "236"},
			GreenText:          lipgloss.AdaptiveColor{Light: "#3B875E", Dark: "#3B875E"},
			YellowText:         lipgloss.AdaptiveColor{Light: "#FAAC26", Dark: "#FAAC26"},
			RedText:            lipgloss.AdaptiveColor{Light: "#b03f3c", Dark: "#b03f3c"},
		},
	}
}

var styles = DefaultStyles()

type issueStatus string

const (
	todo       issueStatus = "todo"
	inProgress issueStatus = "in-progress"
	done       issueStatus = "done"
	wontDo     issueStatus = "wont-do"
)

func (s issueStatus) Icon() string {
	icons := map[issueStatus]string{
		todo:       "[·]",
		inProgress: "[⋯]",
		wontDo:     "[×]",
		done:       "[✓]",
	}
	return lipgloss.NewStyle().Foreground(s.color()).Render(icons[s])
}

func (s issueStatus) PrettyString() string {
	return lipgloss.NewStyle().Foreground(s.color()).Render(string(s))
}

func (s issueStatus) color() lipgloss.AdaptiveColor {
	colors := map[issueStatus]lipgloss.AdaptiveColor{
		todo:       styles.Theme.SecondaryText,
		inProgress: styles.Theme.YellowText,
		wontDo:     styles.Theme.RedText,
		done:       styles.Theme.GreenText,
	}
	return colors[s]
}

type keyMap struct {
	Path                     int
	Up                       key.Binding
	Down                     key.Binding
	Left                     key.Binding
	Right                    key.Binding
	Help                     key.Binding
	Quit                     key.Binding
	ForceQuit                key.Binding
	Suspend                  key.Binding
	Back                     key.Binding
	IssueNewForm             key.Binding
	IssueEditForm            key.Binding
	IssueShowFocus           key.Binding
	IssueStatusDone          key.Binding
	IssueStatusWontDo        key.Binding
	IssueStatusInProgress    key.Binding
	IssueCommentFormFocus    key.Binding
	IssueDelete              key.Binding
	CommitShowFocus          key.Binding
	CommitExpandCheckDetails key.Binding
	NextInput                key.Binding
	Submit                   key.Binding
	NextPage                 key.Binding
	PrevPage                 key.Binding
	RunCheck                 key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	var bindings [][]key.Binding

	switch {
	case matchRoute(k.Path, issuesIndexPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.IssueNewForm, k.IssueShowFocus},
			{k.IssueStatusDone, k.IssueStatusWontDo},
			{k.IssueStatusInProgress, k.IssueCommentFormFocus},
			{k.IssueDelete},
		}
	case matchRoute(k.Path, issuesShowPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.IssueEditForm, k.Back},
			{k.IssueNewForm, k.IssueShowFocus},
			{k.IssueStatusDone, k.IssueStatusWontDo},
			{k.IssueStatusInProgress, k.IssueCommentFormFocus},
		}
	case matchRoute(k.Path, issuesEditConfirmationPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.NextInput, k.Back},
		}
	case matchRoute(k.Path, checksIndexPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.RunCheck, k.CommitShowFocus},
		}
	case matchRoute(k.Path, checksShowPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.RunCheck, k.CommitExpandCheckDetails},
			{k.Back},
		}
	}

	return bindings
}

type Issue struct {
	Id          string      `json:"id"`
	Shortcode   string      `json:"shortcode"`
	Author      string      `json:"author"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      issueStatus `json:"status"`
	Labels      []string    `jaon:"labels"`
	Comments    []Comment   `json:"comments"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   time.Time   `json:"deleted_at"`
}

func (i Issue) FilterValue() string {
	labels := strings.Join(i.Labels, " ")
	return fmt.Sprintf("%s %s", i.Title, labels)
}

func (i Issue) Height() int                             { return 2 }
func (i Issue) Spacing() int                            { return 1 }
func (i Issue) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (i Issue) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Issue)

	if !ok {
		return
	}

	defaultItemStyles := list.NewDefaultItemStyles()

	titleFn := defaultItemStyles.NormalTitle.Padding(0).Render
	if index == m.Index() {
		titleFn = func(s ...string) string {
			return defaultItemStyles.SelectedTitle.
				Border(lipgloss.NormalBorder(), false, false, false, false).
				Padding(0).
				Render(strings.Join(s, " "))
		}
	}
	title := fmt.Sprintf("%s %s", i.Status.Icon(), titleFn(truncate(i.Title, 50)))
	labels := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf(strings.Join(i.Labels, ",")))
	title = fmt.Sprintf("%s %s", title, labels)

	description := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf(
		"#%s opened by %s on %s",
		i.Shortcode,
		i.Author,
		i.CreatedAt.Format(time.DateOnly),
	))
	item := lipgloss.JoinVertical(lipgloss.Left, title, description)

	fmt.Fprintf(w, item)
}

type Comment struct {
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

/* MAIN MODEL */

type Size struct {
	Width  int
	Height int
}

type Layout struct {
	TerminalSize    Size
	AvailableSize   Size
	HeaderSize      Size
	LeftSize        Size
	RightSize       Size
	CommentFormSize Size
	FooterSize      Size
}

func (m Model) IsRightSidebarOpen() bool {
	switch m.path {
	case issuesCommentContentPath, issuesCommentConfirmationPath,
		issuesEditTitlePath, issuesEditDescriptionPath, issuesEditLabelsPath, issuesEditConfirmationPath,
		issuesNewTitlePath, issuesNewDescriptionPath, issuesNewLabelsPath, issuesNewConfirmationPath, issuesShowPath,
		checksShowPath:
		return true
	default:
		return false
	}
}

func (m *Model) UpdateLayout(terminalSize Size) {
	layout := m.layout
	layout.TerminalSize = terminalSize

	windowFrameWidth, windowFrameHeight := windowStyle.GetFrameSize()
	layout.AvailableSize = Size{
		Width:  layout.TerminalSize.Width - windowFrameWidth,
		Height: layout.TerminalSize.Height - windowFrameHeight,
	}
	available := layout.AvailableSize

	layout.HeaderSize = Size{Width: available.Width, Height: lipgloss.Height(m.renderTabs("Issues"))}
	layout.FooterSize = Size{Width: available.Width, Height: lipgloss.Height(m.help.View(m.HelpKeys()))}
	contentHeight := available.Height - layout.HeaderSize.Height - layout.FooterSize.Height
	if m.IsRightSidebarOpen() {
		layout.LeftSize = Size{Width: 70, Height: contentHeight}
		layout.CommentFormSize = Size{
			Width:  clamp(available.Width-layout.LeftSize.Width, 50, 80),
			Height: len(strings.Split(m.commentFormView(), "\n")),
		}
	} else {
		layout.LeftSize = Size{Width: available.Width, Height: contentHeight}
		layout.CommentFormSize = Size{
			Width:  0,
			Height: 0,
		}
	}

	layout.RightSize = Size{Width: available.Width - layout.LeftSize.Width, Height: contentHeight}

	m.layout = layout

	// update component sizes based on layout
	m.issueIndex.SetSize(m.layout.LeftSize.Width, m.layout.LeftSize.Height)
	m.commitIndex.SetSize(m.layout.LeftSize.Width, m.layout.LeftSize.Height)
	m.commentForm.contentInput.SetWidth(m.layout.CommentFormSize.Width)
	m.issueForm.titleInput.Width = clamp(layout.RightSize.Width, 50, 80)
	m.issueForm.labelsInput.Width = clamp(layout.RightSize.Width, 50, 80)
	m.issueForm.descriptionInput.SetWidth(clamp(layout.RightSize.Width, 50, 80))
	m.issueForm.descriptionInput.SetHeight(layout.RightSize.Height / 3)
}

type Model struct {
	loaded             bool
	path               int
	issueIndex         list.Model
	issueShow          issueShow
	issueForm          issueForm
	commentForm        commentForm
	commitIndex        list.Model
	commitShow         commitShow
	err                error
	help               help.Model
	styles             Styles
	tabs               []string
	previousSearchTerm string
	msgDump            io.Writer
	layout             Layout
	router             *Router
	author             string
}

type SetSearchTermMsg string

func SetSearchTerm(term string) tea.Cmd {
	return func() tea.Msg {
		return SetSearchTermMsg(term)
	}
}

func (m Model) submitIssueForm() tea.Cmd {
	var cmd tea.Cmd
	form := m.issueForm
	description := form.descriptionInput.Value()
	title := form.titleInput.Value()
	labels := strings.Split(form.labelsInput.Value(), " ")

	if m.issueForm.editing {
		currentIssue := m.issueIndex.SelectedItem().(Issue)
		currentIssue.Title = title
		currentIssue.Description = description
		currentIssue.Labels = labels
		cmd = persistIssue(currentIssue)
	} else {
		description := form.descriptionInput.Value()

		newIssue := Issue{
			Shortcode:   "xxxxxx",
			Title:       title,
			Description: description,
			Labels:      labels,
			Status:      todo,
			Author:      m.author,
		}
		cmd = persistIssue(newIssue)
	}

	return cmd
}

type issueForm struct {
	titleInput       textinput.Model
	labelsInput      textinput.Model
	descriptionInput textarea.Model
	identifier       string
	editing          bool
}

var (
	inactiveTabBorder   = lipgloss.NormalBorder()
	activeTabBorder     = lipgloss.NormalBorder()
	docStyle            = lipgloss.NewStyle().Padding(0)
	highlightColor      = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle    = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(styles.Theme.FaintBorder).Padding(0, 0)
	activeTabStyle      = lipgloss.NewStyle().Border(activeTabBorder, true).BorderForeground(styles.Theme.PrimaryBorder).Padding(0, 0)
	windowStyle         = lipgloss.NewStyle().Padding(2)
	contentAreaStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).BorderForeground(styles.Theme.FaintText)
	helpStyle           = lipgloss.NewStyle().Padding(0, 0)
	commentContentStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(styles.Theme.SecondaryBorder).MarginTop(1)
	commentHeaderStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(styles.Theme.SecondaryBorder)
)

func InitialModel() Model {
	author, err := getGitAuthorEmail()
	if err != nil {
		log.Fatal(err)
	}
	layout := Layout{}

	issueList := list.New([]list.Item{}, Issue{}, 0, 0)
	issueList.SetShowHelp(false)
	issueList.SetShowTitle(false)
	issueList.SetShowStatusBar(false)
	issueList.Styles.TitleBar = lipgloss.NewStyle().Padding(0)
	issueList.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0)
	issueList.FilterInput.Prompt = "search: "
	issueList.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText)
	issueList.Title = "Issues"
	commitList := list.New([]list.Item{}, Commit{}, 0, 0)
	commitList.SetShowHelp(false)
	commitList.SetShowTitle(false)
	commitList.SetShowStatusBar(false)
	commitList.Styles.TitleBar = lipgloss.NewStyle().Padding(0)
	commitList.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0)
	commitList.FilterInput.Prompt = "search: "
	commitList.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText)
	commitList.Title = "Commits"

	helpModel := help.New()
	helpModel.FullSeparator = "    "

	router := NewRouter()
	router.AddRoute(issuesIndexPath, issuesIndexHandler)
	router.AddRoute(issuesShowPath, issuesShowHandler)

	return Model{
		path:        issuesIndexPath,
		help:        helpModel,
		styles:      DefaultStyles(),
		tabs:        []string{"Issues", "Checks"},
		layout:      layout,
		issueIndex:  issueList,
		commitIndex: commitList,
		commentForm: newCommentForm(),
		issueForm:   newIssueForm("", "", "", []string{}, false),
		issueShow:   newIssueShow(Issue{}, layout),
		router:      router,
		author:      author,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(getIssues, getCommits)
}

type layoutMsg Layout

func (m Model) isUserTyping() bool {
	paths := []int{
		issuesCommentContentPath,
		issuesEditTitlePath,
		issuesEditLabelsPath,
		issuesEditDescriptionPath,
		issuesEditConfirmationPath,
		issuesNewTitlePath,
		issuesNewLabelsPath,
		issuesNewDescriptionPath,
	}

	return slices.Contains(paths, m.path)
}

func issuesIndexHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.issueIndex.SettingFilter() {
		m.issueIndex, cmd = m.issueIndex.Update(msg)
		return m, cmd
	}
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Help):
			if m.help.ShowAll {
				m.help.ShowAll = false
				return m, nil
			} else {
				var maxHelpHeight int
				for _, column := range keys.FullHelp() {
					if len(column) > maxHelpHeight {
						maxHelpHeight = len(column)
					}
				}
				m.help.ShowAll = true
				return m, nil
			}
		case key.Matches(msg, keys.IssueStatusDone):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == done {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = done
			}
			cmd = persistIssue(currentIssue)
			return m, cmd
		case key.Matches(msg, keys.IssueStatusWontDo):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == wontDo {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = wontDo
			}
			cmd = persistIssue(currentIssue)
			return m, cmd
		case key.Matches(msg, keys.IssueStatusInProgress):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == inProgress {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = inProgress
			}
			cmd = persistIssue(currentIssue)
			return m, cmd
		case key.Matches(msg, keys.IssueCommentFormFocus):
			m.commentForm = newCommentForm()
			cmd = m.commentForm.Init()
			m.path = issuesCommentContentPath
			m.UpdateLayout(m.layout.TerminalSize)
			m.issueShow = newIssueShow(m.issueIndex.SelectedItem().(Issue), m.layout)
			m.issueShow.viewport.GotoBottom()
			return m, cmd
		case key.Matches(msg, keys.IssueShowFocus):
			m.commentForm = newCommentForm()
			m.path = issuesShowPath
			m.UpdateLayout(m.layout.TerminalSize)
			m.issueShow = newIssueShow(m.issueIndex.SelectedItem().(Issue), m.layout)
		case key.Matches(msg, keys.IssueNewForm):
			m.path = issuesNewTitlePath
			m.issueForm = newIssueForm("", "", "", []string{}, false)
			cmd = m.issueForm.titleInput.Focus()
			m.UpdateLayout(m.layout.TerminalSize)
			return m, cmd
		case key.Matches(msg, keys.IssueDelete):
			selectedItem := m.issueIndex.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}
			issue := selectedItem.(Issue)
			issue.DeletedAt = time.Now().UTC()
			cmd = persistIssue(issue)
			return m, cmd
		case key.Matches(msg, keys.NextPage):
			m.path = checksIndexPath
			return m, nil
		case key.Matches(msg, keys.PrevPage):
			return m, nil
		}
	}

	m.issueIndex, cmd = m.issueIndex.Update(msg)
	return m, cmd
}

func issuesShowHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Help):
			if m.help.ShowAll {
				m.help.ShowAll = false
			} else {
				m.help.ShowAll = true
			}
			return m, nil
		case key.Matches(msg, keys.IssueStatusDone):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == todo {
				currentIssue.Status = done
			} else {
				currentIssue.Status = todo
			}
			m.commentForm = newCommentForm()
			m.issueShow = newIssueShow(currentIssue, m.layout)
			cmd = persistIssue(currentIssue)
			return m, cmd
		case key.Matches(msg, keys.IssueStatusWontDo):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == wontDo {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = wontDo
			}
			m.commentForm = newCommentForm()
			m.issueShow = newIssueShow(currentIssue, m.layout)
			cmd = persistIssue(currentIssue)
			return m, cmd
		case key.Matches(msg, keys.IssueStatusInProgress):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == inProgress {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = inProgress
			}
			m.commentForm = newCommentForm()
			m.issueShow = newIssueShow(currentIssue, m.layout)
			cmd = persistIssue(currentIssue)
			return m, cmd
		case key.Matches(msg, keys.IssueEditForm):
			selectedIssue := m.issueIndex.SelectedItem().(Issue)
			m.issueForm = newIssueForm(
				selectedIssue.Shortcode,
				selectedIssue.Title,
				selectedIssue.Description,
				selectedIssue.Labels,
				true,
			)
			cmd = m.issueForm.titleInput.Focus()

			m.path = issuesEditTitlePath
			m.UpdateLayout(m.layout.TerminalSize)
			return m, cmd
		case key.Matches(msg, keys.Back):
			m.path = issuesIndexPath
		case key.Matches(msg, keys.IssueCommentFormFocus):
			m.commentForm = newCommentForm()
			cmd = m.commentForm.Init()
			m.path = issuesCommentContentPath
			m.UpdateLayout(m.layout.TerminalSize)
			m.issueShow = newIssueShow(m.issueIndex.SelectedItem().(Issue), m.layout)
			m.issueShow.viewport.GotoBottom()
			return m, cmd
		}
	}

	m.issueShow.viewport, cmd = m.issueShow.viewport.Update(msg)
	return m, cmd
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.msgDump != nil {
		fmt.Fprintf(m.msgDump, "%T\n", msg)
	}
	var cmd tea.Cmd

	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			if !m.isUserTyping() {
				return m, tea.Quit
			}
		case key.Matches(msg, keys.ForceQuit):
			return m, tea.Quit
		case key.Matches(msg, keys.Suspend):
			return m, tea.Suspend
		}
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.loaded = true
		}

		m.UpdateLayout(Size{Width: msg.Width, Height: msg.Height})
		return m, nil
	case tea.FocusMsg:
		return m, tea.Sequence(getIssues, getCommits, SetSearchTerm(m.previousSearchTerm))
	case tea.BlurMsg:
		if m.issueIndex.FilterValue() != "" {
			m.previousSearchTerm = m.issueIndex.FilterValue()
		}
		return m, nil
	case IssuesReadyMsg:
		var listItems []list.Item
		for _, issue := range msg {
			listItems = append(listItems, issue)
		}
		m.issueIndex.SetItems(listItems)
	case CommitListReadyMsg:
		var listItems []list.Item
		for _, commit := range msg {
			listItems = append(listItems, commit)
		}
		m.commitIndex.SetItems(listItems)
	case commentForm:
		currentIssue := m.issueIndex.SelectedItem().(Issue)
		currentIssue.Comments = append(currentIssue.Comments, Comment{
			Author:  m.author,
			Content: msg.contentInput.Value(),
		})

		cmd = persistIssue(currentIssue)
		return m, cmd
	case SetSearchTermMsg:
		if msg != "" {
			m.issueIndex.SetFilterText(string(msg))
		}

		m.previousSearchTerm = ""

		return m, nil
	case issuePersistedMsg:
		if !msg.Issue.DeletedAt.IsZero() {
			currentIndex := m.issueIndex.Index()
			m.issueIndex.RemoveItem(currentIndex)
			m.issueIndex.Select(clamp(currentIndex-1, 0, len(m.issueIndex.Items())))
			m.issueIndex, cmd = m.issueIndex.Update(msg)
		} else {
			issues := convertSlice(m.issueIndex.Items(), func(item list.Item) Issue {
				return item.(Issue)
			})
			for i, issue := range issues {
				if issue.Id == msg.Issue.Id {
					issues[i] = msg.Issue
				}
			}

			if msg.IsNewIssue {
				issues = append(issues, msg.Issue)
			}

			sortedIssues := SortIssues(issues)

			var listIndexToFocus int
			for i, issue := range sortedIssues {
				if issue.Id == msg.Issue.Id {
					listIndexToFocus = i
				}
			}
			items := convertSlice(sortedIssues, func(issue Issue) list.Item {
				return list.Item(issue)
			})
			m.issueIndex.SetItems(items)
			m.issueIndex.Select(listIndexToFocus)
			m.commentForm = newCommentForm()
			m.issueShow = newIssueShow(msg.Issue, m.layout)
			if msg.ScrollToBottom {
				m.issueShow.viewport.GotoBottom()
			}
		}

		if m.path != issuesIndexPath {
			m.path = issuesShowPath
		}
		return m, cmd
	}

	switch {
	case matchRoute(m.path, issuesCommentContentPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				currentIssue := m.issueIndex.SelectedItem().(Issue)
				m.commentForm = newCommentForm()
				m.issueShow = newIssueShow(currentIssue, m.layout)
				m.path = issuesShowPath
			case key.Matches(msg, keys.NextInput):
				m.commentForm.contentInput.Blur()
				m.commentForm.confirming = true
				m.path = issuesCommentConfirmationPath
			}
		}
		m.commentForm.contentInput, cmd = m.commentForm.contentInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesCommentConfirmationPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				currentIssue := m.issueIndex.SelectedItem().(Issue)
				m.commentForm = newCommentForm()
				m.issueShow = newIssueShow(currentIssue, m.layout)
				m.path = issuesShowPath
			case key.Matches(msg, keys.Submit):
				return m, m.commentForm.Submit
			}
		}
	case matchRoute(m.path, issuesEditTitlePath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesEditLabelsPath
				m.issueForm.titleInput.Blur()
				cmd = m.issueForm.labelsInput.Focus()
				return m, cmd
			}
		}

		m.issueForm.titleInput, cmd = m.issueForm.titleInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesEditLabelsPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesEditDescriptionPath
				m.issueForm.labelsInput.Blur()
				m.issueForm.descriptionInput.Focus()
				return m, cmd
			}
		}

		m.issueForm.labelsInput, cmd = m.issueForm.labelsInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesEditDescriptionPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesEditConfirmationPath
				m.issueForm.descriptionInput.Blur()
				return m, cmd
			}
		}

		m.issueForm.descriptionInput, cmd = m.issueForm.descriptionInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesEditConfirmationPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesEditTitlePath
				cmd = m.issueForm.titleInput.Focus()
				return m, cmd
			case key.Matches(msg, keys.Submit):
				return m, m.submitIssueForm()
			}
		}

		return m, cmd
	case matchRoute(m.path, issuesNewTitlePath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesNewLabelsPath
				m.issueForm.titleInput.Blur()
				cmd = m.issueForm.labelsInput.Focus()
				return m, cmd
			}
		}

		m.issueForm.titleInput, cmd = m.issueForm.titleInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesNewLabelsPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesNewDescriptionPath
				m.issueForm.labelsInput.Blur()
				m.issueForm.descriptionInput.Focus()
				return m, cmd
			}
		}

		m.issueForm.labelsInput, cmd = m.issueForm.labelsInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesNewDescriptionPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesNewConfirmationPath
				m.issueForm.descriptionInput.Blur()
				return m, cmd
			}
		}

		m.issueForm.descriptionInput, cmd = m.issueForm.descriptionInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesNewConfirmationPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				if m.issueForm.editing {
					m.path = issuesShowPath
					return m, cmd
				} else {
					m.path = issuesIndexPath
					return m, cmd
				}
			case key.Matches(msg, keys.NextInput):
				m.path = issuesNewTitlePath
				cmd = m.issueForm.titleInput.Focus()
				return m, cmd
			case key.Matches(msg, keys.Submit):
				return m, m.submitIssueForm()
			}
		}

		return m, cmd
	case matchRoute(m.path, checksIndexPath):
		if m.commitIndex.SettingFilter() {
			m.commitIndex, cmd = m.commitIndex.Update(msg)
			return m, cmd
		}

		switch msg := msg.(type) {
		case checkResult:
			return m, persistCheck(Check(msg))
		case checkPersistedMsg:
			check := msg.Check
			var commit Commit
			var commitIndex int
			for i, c := range m.commitIndex.Items() {
				if c.(Commit).Id == check.CommitId {
					commit = c.(Commit)
					commitIndex = i
					break
				}
			}
			updatedChecks := make([]Check, len(commit.LatestChecks))
			for i, c := range commit.LatestChecks {
				if check.Id == c.Id {
					updatedChecks[i] = check
				} else {
					updatedChecks[i] = c
				}
			}
			commit.LatestChecks = updatedChecks
			m.commitIndex.SetItem(commitIndex, commit)
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Down):
				m.commitIndex, cmd = m.commitIndex.Update(msg)
				return m, cmd
			case key.Matches(msg, keys.Up):
				m.commitIndex, cmd = m.commitIndex.Update(msg)
				return m, cmd
			case key.Matches(msg, keys.RunCheck):
				commit := m.commitIndex.SelectedItem().(Commit)
				var cmds []tea.Cmd
				checks := NewChecks(commit)
				cmds = append(cmds, commit.DeleteExistingChecks())
				commit.LatestChecks = checks

				for i, check := range commit.LatestChecks {
					check.ExecutionPosition = i
					cmds = append(cmds, RunCheck(check))
				}
				m.commitIndex.SetItem(m.commitIndex.Index(), commit)
				return m, tea.Batch(cmds...)
			case key.Matches(msg, keys.CommitShowFocus):
				m.path = checksShowPath
				m.UpdateLayout(m.layout.TerminalSize)
				m.commitShow = newCommitShow(m.commitIndex.SelectedItem().(Commit), m.layout, false)
				return m, cmd
			case key.Matches(msg, keys.NextPage):
				return m, nil
			case key.Matches(msg, keys.PrevPage):
				m.path = issuesIndexPath
				return m, nil
			case key.Matches(msg, keys.Help):
				if m.help.ShowAll {
					m.help.ShowAll = false
					return m, nil
				} else {
					var maxHelpHeight int
					for _, column := range keys.FullHelp() {
						if len(column) > maxHelpHeight {
							maxHelpHeight = len(column)
						}
					}
					m.help.ShowAll = true
					return m, nil
				}
			}
		}

		m.commitIndex, cmd = m.commitIndex.Update(msg)
		return m, cmd
	case matchRoute(m.path, checksShowPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				m.path = checksIndexPath
			case key.Matches(msg, keys.RunCheck):
				commit := m.commitIndex.SelectedItem().(Commit)
				var cmds []tea.Cmd
				checks := NewChecks(commit)
				cmds = append(cmds, commit.DeleteExistingChecks())
				commit.LatestChecks = checks

				for i, check := range commit.LatestChecks {
					check.ExecutionPosition = i
					cmds = append(cmds, RunCheck(check))
				}
				m.commitIndex.SetItem(m.commitIndex.Index(), commit)
				m.UpdateLayout(m.layout.TerminalSize)
				m.commitShow = newCommitShow(commit, m.layout, false)
				return m, tea.Batch(cmds...)
			case key.Matches(msg, keys.CommitExpandCheckDetails):
				commit := m.commitIndex.SelectedItem().(Commit)
				var expand bool
				if m.commitShow.expandCheckDetails {
					expand = false
				} else {
					expand = true
				}
				m.UpdateLayout(m.layout.TerminalSize)
				m.commitShow = newCommitShow(commit, m.layout, expand)
			}
		case checkResult:
			return m, persistCheck(Check(msg))
		case checkPersistedMsg:
			check := msg.Check
			var commit Commit
			var commitIndex int
			for i, c := range m.commitIndex.Items() {
				if c.(Commit).Id == check.CommitId {
					commit = c.(Commit)
					commitIndex = i
					break
				}
			}
			updatedChecks := make([]Check, len(commit.LatestChecks))
			for i, c := range commit.LatestChecks {
				if check.Id == c.Id {
					updatedChecks[i] = check
				} else {
					updatedChecks[i] = c
				}
			}
			commit.LatestChecks = updatedChecks
			m.commitIndex.SetItem(commitIndex, commit)
			m.UpdateLayout(m.layout.TerminalSize)
			m.commitShow = newCommitShow(commit, m.layout, false)
		}

		m.commitShow.viewport, cmd = m.commitShow.viewport.Update(msg)
		return m, cmd
	}

	return m.router.Route(m, msg)
}

type checkResult Check

func RunCheck(check Check) tea.Cmd {
	return func() tea.Msg {
		result, err := executeCheckUsingArchive(check)
		check.Output = result
		check.FinishedAt = time.Now().UTC()
		if err != nil {
			debug("Check failed: %v", err)
			check.Status = failed
			return checkResult(check)
		}
		check.Status = succeeded
		return checkResult(check)
	}
}

func executeCheckUsingArchive(check Check) (string, error) {
	tempDir, err := os.MkdirTemp("", "check-archive-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// #nosec G204
	archiveCmd := exec.Command("git", "archive", "--format=tar", check.CommitId)
	archive, err := archiveCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}

	extractCmd := exec.Command("tar", "-xf", "-")
	extractCmd.Dir = tempDir
	extractCmd.Stdin = bytes.NewReader(archive)
	if err := extractCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}

	check.Command.Dir = tempDir
	output, err := runCommandWithOutput(check.Command)
	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

func runCommandWithOutput(command *exec.Cmd) (string, error) {
	var outputBuffer bytes.Buffer
	command.Stdout = &outputBuffer
	command.Stderr = &outputBuffer

	if err := command.Run(); err != nil {
		return outputBuffer.String(), err
	}

	return outputBuffer.String(), nil
}

func (m Model) HelpKeys() keyMap {
	keys := keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
		),
		Suspend: key.NewBinding(
			key.WithKeys("ctrl+z"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		IssueEditForm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit issue"),
		),
		IssueStatusDone: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle done"),
		),
		IssueStatusWontDo: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "toggle wont-do"),
		),
		IssueStatusInProgress: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "toggle in-progress"),
		),
		IssueCommentFormFocus: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "toggle issue comment form"),
		),
		IssueShowFocus: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "more info"),
		),
		IssueNewForm: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new issue"),
		),
		IssueDelete: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "delete issue"),
		),
		NextInput: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next input"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("right", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("left", "previous page"),
		),
		RunCheck: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "run check"),
		),
		CommitShowFocus: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "more info"),
		),
		CommitExpandCheckDetails: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "expand check details"),
		),
	}

	keys.Path = m.path

	return keys
}

func (m Model) renderTabs(activeTab string) string {
	var renderedTabs []string
	for _, t := range m.tabs {
		style := inactiveTabStyle
		if t == activeTab {
			style = activeTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func (m Model) renderMainLayout(header, left, right, footer string) string {
	if right == "" {
		right = ""
	} else {
		right = lipgloss.NewStyle().Width(m.layout.RightSize.Width).Render(right)
	}
	return windowStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(m.layout.LeftSize.Width).Height(m.layout.LeftSize.Height).Render(left),
			right,
		),
		footer,
	))
}

func (m Model) renderIssuesView() string {
	left := m.issueIndex.View()
	var right string
	switch m.path {
	case issuesShowPath:
		right = m.issueShowView()
	case issuesCommentContentPath, issuesCommentConfirmationPath:
		right = lipgloss.JoinVertical(lipgloss.Left, m.issueShowView(), m.commentFormView())
	case issuesEditTitlePath, issuesEditLabelsPath, issuesEditDescriptionPath, issuesEditConfirmationPath,
		issuesNewTitlePath, issuesNewLabelsPath, issuesNewDescriptionPath, issuesNewConfirmationPath:
		right = m.issueFormView()
	}
	return m.renderMainLayout(m.renderTabs("Issues"), left, right, m.help.View(m.HelpKeys()))
}

func (m Model) renderChecksView() string {
	left := m.commitIndex.View()
	var right string
	if m.path == checksShowPath {
		right = m.commitShowView()
	}
	return m.renderMainLayout(m.renderTabs("Checks"), left, right, m.help.View(m.HelpKeys()))
}

func (m Model) View() string {
	if !m.loaded {
		return "Loading..."
	}

	var view string
	switch m.path {
	case issuesIndexPath, issuesShowPath, issuesCommentContentPath, issuesCommentConfirmationPath,
		issuesEditTitlePath, issuesEditLabelsPath, issuesEditDescriptionPath, issuesEditConfirmationPath,
		issuesNewTitlePath, issuesNewLabelsPath, issuesNewDescriptionPath, issuesNewConfirmationPath:
		view = m.renderIssuesView()
	case checksIndexPath, checksShowPath:
		view = m.renderChecksView()
	}

	return docStyle.Render(view)
}

type Commit struct {
	Id            string    `json:"id"`
	AbbreviatedId string    `json:"abbreviatedId"`
	Author        string    `json:"author"`
	Description   string    `json:"description"`
	Timestamp     time.Time `json:"timestamp"`
	LatestChecks  []Check   `json:"latestCheck"`
}

func (c Commit) AggregateCheckStatus() CheckStatus {
	if len(c.LatestChecks) == 0 {
		return ""
	}

	hasFailedCheck := false
	for _, check := range c.LatestChecks {
		switch check.Status {
		case running:
			return running
		case failed:
			if check.Optional {
				continue
			}
			hasFailedCheck = true
		case succeeded:
			// Continue checking other checks
		default:
			return running
		}
	}

	if hasFailedCheck {
		return failed
	}
	return succeeded
}

func (c Commit) DeleteExistingChecks() tea.Cmd {
	var cmds []tea.Cmd

	for _, check := range c.LatestChecks {
		cmds = append(cmds, check.Delete)
	}

	return tea.Batch(cmds...)
}

type CheckStatus string

func (c CheckStatus) Icon() string {
	icons := map[CheckStatus]string{
		running:   "[⋯]",
		failed:    "[×]",
		succeeded: "[✓]",
	}
	return lipgloss.NewStyle().Foreground(c.color()).Render(icons[c])
}

func (c CheckStatus) PrettyString() string {
	return lipgloss.NewStyle().Foreground(c.color()).Render(string(c))
}

func (c CheckStatus) color() lipgloss.AdaptiveColor {
	colors := map[CheckStatus]lipgloss.AdaptiveColor{
		running:   styles.Theme.YellowText,
		failed:    styles.Theme.RedText,
		succeeded: styles.Theme.GreenText,
	}
	return colors[c]
}

const (
	failed    CheckStatus = "failed"
	succeeded CheckStatus = "succeeded"
	running   CheckStatus = "running"
)

type Check struct {
	Command           *exec.Cmd   `json:"-"`
	Id                string      `json:"id"`
	CommitId          string      `json:"commitId"`
	Status            CheckStatus `json:"status"`
	Checker           string      `json:"checker"`
	Name              string      `json:"name"`
	Output            string      `json:"output"`
	StartedAt         time.Time   `json:"startedAt"`
	FinishedAt        time.Time   `json:"finishedAt"`
	Optional          bool        `json:"optional"`
	ExecutionPosition int         `json:"executionPosition"`
}

func NewChecks(commit Commit) []Check {
	return []Check{
		Check{
			Id:        uuid.NewString(),
			Status:    running,
			CommitId:  commit.Id,
			Command:   exec.Command("go", "test"),
			Name:      "Tests ('go test')",
			StartedAt: time.Now().UTC(),
		},
		Check{
			Id:        uuid.NewString(),
			Status:    running,
			CommitId:  commit.Id,
			Command:   exec.Command("gosec", "./"),
			Name:      "Security ('gosec')",
			StartedAt: time.Now().UTC(),
			Optional:  true,
		},
	}
}

func (c Check) Delete() tea.Msg {
	// #nosec G204
	cmd := exec.Command("git", "update-ref", "-d", fmt.Sprintf("refs/ubik/checks/%s", c.Id))
	err := cmd.Run()

	if err != nil {
		debug("%#v", err)
		panic(err)
	}

	return nil
}

func (c Check) ElapsedTime() time.Duration {
	return c.FinishedAt.Sub(c.StartedAt)
}

func (c Commit) FilterValue() string {
	return c.Id
}

func (c Commit) Height() int  { return 2 }
func (c Commit) Spacing() int { return 1 }
func (c Commit) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (c Commit) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	c, ok := listItem.(Commit)

	if !ok {
		return
	}

	defaultItemStyles := list.NewDefaultItemStyles()

	var author string

	if c.Author == "" {
		author = "unknown"
	} else {
		author = c.Author
	}

	titleFn := defaultItemStyles.NormalTitle.Padding(0).Render
	if index == m.Index() {
		titleFn = func(s ...string) string {
			return defaultItemStyles.SelectedTitle.
				Border(lipgloss.NormalBorder(), false, false, false, false).
				Padding(0).
				Render(strings.Join(s, " "))
		}
	}

	title := fmt.Sprintf("%s", titleFn(c.AbbreviatedId, truncate(c.Description, 50)))

	if len(c.LatestChecks) > 0 {
		title = fmt.Sprintf("%s %s", title, c.AggregateCheckStatus().Icon())
	}

	description := fmt.Sprintf("committed at %s by %s", c.Timestamp.Format(time.RFC822), author)
	item := lipgloss.JoinVertical(lipgloss.Left, title, description)

	fmt.Fprintf(w, item)
}

type CommitListReadyMsg []Commit

func getCommits() tea.Msg {
	var commits []*Commit

	cmd := exec.Command(
		"git",
		"log",
		"--pretty=format:%H|%h|%ae|%aI|%s",
		"--date=format:%d %b %y %H:%M %z",
	)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running command:", err)
	}

	for _, commitHash := range strings.Split(string(output), "\n") {
		parts := strings.Split(commitHash, "|")

		timestamp, err := time.Parse(time.RFC3339, parts[3])
		if err != nil {
			log.Errorf("Error parsing timestamp: %v", err)
		}
		commits = append(commits, &Commit{
			Id:            parts[0],
			AbbreviatedId: parts[1],
			Author:        parts[2],
			Timestamp:     timestamp,
			Description:   parts[4],
		})
	}

	cmd = exec.Command("git", "for-each-ref", "--format=%(objectname)", "refs/ubik/checks")
	b, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var refHashes []string
	str := string(b)
	for _, s := range strings.Split(str, "\n") {
		noteId := strings.Split(s, " ")[0]
		if noteId != "" {
			refHashes = append(refHashes, noteId)
		}
	}

	var readyCommits []Commit
	for _, refHash := range refHashes {
		// #nosec G204
		cmd := exec.Command("git", "cat-file", "-p", refHash)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			continue
		}

		var check Check
		err = json.Unmarshal(out.Bytes(), &check)
		if err != nil {
			panic(err)
		}

		for _, commit := range commits {
			if check.CommitId == commit.Id {
				commit.LatestChecks = append(commit.LatestChecks, check)
				slices.SortFunc(commit.LatestChecks, func(a, b Check) int {
					return a.ExecutionPosition - b.ExecutionPosition
				})
			}
		}

	}

	for _, commit := range commits {
		readyCommits = append(readyCommits, *commit)
	}

	return CommitListReadyMsg(readyCommits)
}

type IssuesReadyMsg []Issue

func getIssues() tea.Msg {
	var issues []Issue

	cmd := exec.Command("git", "for-each-ref", "--format=%(objectname)", "refs/ubik/issues")
	b, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var refHashes []string
	str := string(b)
	for _, s := range strings.Split(str, "\n") {
		noteId := strings.Split(s, " ")[0]
		if noteId != "" {
			refHashes = append(refHashes, noteId)
		}
	}

	for _, refHash := range refHashes {
		// #nosec G204
		cmd := exec.Command("git", "cat-file", "-p", refHash)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			continue
		}

		var issue Issue
		err = json.Unmarshal(out.Bytes(), &issue)
		if err != nil {
			panic(err)
		}

		if issue.DeletedAt.IsZero() {
			issues = append(issues, issue)
		}
	}

	sortedIssues := SortIssues(issues)

	return IssuesReadyMsg(sortedIssues)
}

func SortIssues(issues []Issue) []Issue {
	var openIssues []Issue
	var closedIssues []Issue
	for _, issue := range issues {
		if !issue.DeletedAt.IsZero() {
			continue
		}

		if issue.Status == done || issue.Status == wontDo {
			closedIssues = append(closedIssues, issue)
			continue
		}

		openIssues = append(openIssues, issue)
	}

	slices.SortFunc(openIssues, func(a, b Issue) int {
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})

	slices.SortFunc(closedIssues, func(a, b Issue) int {
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})

	return append(openIssues, closedIssues...)
}

type commitShow struct {
	commit             Commit
	viewport           viewport.Model
	expandCheckDetails bool
}

func newCommitShow(commit Commit, layout Layout, expandCheckDetails bool) commitShow {
	var s strings.Builder

	viewport := viewport.New(layout.RightSize.Width, layout.RightSize.Height)
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf("%s", commit.AbbreviatedId))
	var header string
	if len(commit.LatestChecks) > 0 {
		header = fmt.Sprintf("%s %s\nStatus: %s\n\n", identifier, commit.Description, commit.AggregateCheckStatus().PrettyString())
	} else {
		header = fmt.Sprintf("%s %s\n\nNo checks yet.", identifier, commit.Description)
	}
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString("\n")

	for _, check := range commit.LatestChecks {
		s.WriteString(fmt.Sprintf("\n%s %s", check.Status.Icon(), check.Name))
		if check.Optional {
			s.WriteString(lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(" (optional)"))
		}
		if expandCheckDetails {
			s.WriteString(
				lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(
					fmt.Sprintf(" finished in %s\n\n", check.ElapsedTime()),
				),
			)
			s.WriteString(fmt.Sprintf("\n%s\n\n", check.Output))
		}
	}

	viewport.SetContent(s.String())

	return commitShow{
		commit:             commit,
		viewport:           viewport,
		expandCheckDetails: expandCheckDetails,
	}
}

func (m Model) commitShowView() string {
	var s strings.Builder
	s.WriteString(m.commitShow.viewport.View())
	return s.String()
}

type issueShow struct {
	issue    Issue
	viewport viewport.Model
}

func (m *Model) InitIssueShow() {
	var s strings.Builder
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("#%s", m.issueShow.issue.Shortcode))
	labels := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf("%s", strings.Join(m.issueShow.issue.Labels, ",")))
	header := fmt.Sprintf("%s %s %s\nStatus: %s\n\n", identifier, m.issueShow.issue.Title, labels, m.issueShow.issue.Status.PrettyString())
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString(m.issueShow.issue.Description + "\n")

	commentFrameX, _ := commentContentStyle.GetFrameSize()
	w := lipgloss.NewStyle().Width(m.issueShow.viewport.Width - commentFrameX)

	for _, comment := range m.issueShow.issue.Comments {
		commentHeader := commentHeaderStyle.Inherit(w).Render(fmt.Sprintf("%s commented at %s", comment.Author, comment.CreatedAt.Format(time.RFC822)))
		s.WriteString(commentContentStyle.Inherit(w).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.Content)))
	}
	m.issueShow.viewport.SetContent(s.String())
}

func newIssueShow(issue Issue, layout Layout) issueShow {
	var s strings.Builder
	viewport := viewport.New(layout.RightSize.Width, layout.RightSize.Height-layout.CommentFormSize.Height)
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("#%s", issue.Shortcode))
	labels := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf("%s", strings.Join(issue.Labels, ",")))
	header := fmt.Sprintf("%s %s %s\nStatus: %s\n\n", identifier, issue.Title, labels, issue.Status.PrettyString())
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString(issue.Description + "\n")

	commentFrameX, _ := commentContentStyle.GetFrameSize()
	w := lipgloss.NewStyle().Width(viewport.Width - commentFrameX)

	for _, comment := range issue.Comments {
		commentHeader := commentHeaderStyle.Inherit(w).Render(fmt.Sprintf("%s commented at %s", comment.Author, comment.CreatedAt.Format(time.RFC822)))
		s.WriteString(commentContentStyle.Inherit(w).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.Content)))
	}
	viewport.SetContent(s.String())

	return issueShow{
		issue:    issue,
		viewport: viewport,
	}
}

func (m Model) issueShowView() string {
	return m.issueShow.viewport.View()
}

func newIssueForm(identifier, title, description string, labels []string, editing bool) issueForm {
	form := issueForm{
		identifier:       identifier,
		titleInput:       textinput.New(),
		labelsInput:      textinput.New(),
		descriptionInput: textarea.New(),
		editing:          editing,
	}

	form.titleInput.CharLimit = 120
	form.titleInput.SetValue(title)

	form.labelsInput.CharLimit = 100
	form.labelsInput.SetValue(strings.Join(labels, " "))

	form.descriptionInput.CharLimit = 0 // unlimited
	form.descriptionInput.MaxHeight = 0 // unlimited
	form.descriptionInput.ShowLineNumbers = false
	form.descriptionInput.SetValue(description)

	return form
}

func (m issueForm) Update(msg tea.Msg) (issueForm, tea.Cmd) {
	return m, nil
}

func (m Model) issueFormView() string {
	var s strings.Builder

	form := m.issueForm

	identifier := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("#%s", form.identifier))
	labelStyle := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render
	fieldStyle := lipgloss.NewStyle().Foreground(styles.Theme.PrimaryText).Render

	if m.issueForm.editing {
		s.WriteString(fmt.Sprintf("Editing issue %s\n\n", identifier))
	} else {
		s.WriteString("New issue\n\n")
	}

	s.WriteString(labelStyle("Title"))
	s.WriteString("\n")
	s.WriteString(fieldStyle(form.titleInput.View()))
	s.WriteString("\n\n")
	s.WriteString(labelStyle("Labels"))
	s.WriteString("\n")
	s.WriteString(fieldStyle(form.labelsInput.View()))
	s.WriteString("\n\n")
	s.WriteString(labelStyle("Description"))
	s.WriteString("\n")
	s.WriteString(fieldStyle(form.descriptionInput.View()))
	s.WriteString("\n\n")
	var confirmationStyle lipgloss.Style
	if matchRoute(m.path, issuesEditConfirmationPath) || matchRoute(m.path, issuesNewConfirmationPath) {
		confirmationStyle = lipgloss.NewStyle().Foreground(styles.Theme.PrimaryText).Background(styles.Theme.SelectedBackground)
	} else {
		confirmationStyle = lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText)
	}

	s.WriteString(confirmationStyle.Render("Save"))

	return s.String()
}

func (m *issueForm) SetDescription(description string) {
}

type commentForm struct {
	contentInput textarea.Model
	confirming   bool
}

func (m Model) newCommentForm() commentForm {
	t := textarea.New()
	t.ShowLineNumbers = false
	t.Prompt = "┃"
	t.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("transparent"))
	t.SetCursor(0)
	t.SetWidth(m.layout.CommentFormSize.Width)
	t.Focus()

	return commentForm{
		contentInput: t,
	}
}

func newCommentForm() commentForm {
	t := textarea.New()
	t.ShowLineNumbers = false
	t.Prompt = "┃"
	t.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("transparent"))
	t.SetCursor(0)
	t.Focus()

	return commentForm{
		contentInput: t,
	}
}

func (m commentForm) Submit() tea.Msg {
	return m
}

func (m commentForm) Init() tea.Cmd {
	return textarea.Blink
}

func (m commentForm) Update(msg tea.Msg) (commentForm, tea.Cmd) {
	return m, nil
}

func (m Model) commentFormView() string {
	var s strings.Builder
	s.WriteString(m.commentForm.contentInput.View())
	s.WriteString("\n")
	if m.commentForm.confirming {
		s.WriteString(lipgloss.NewStyle().Foreground(styles.Theme.PrimaryText).Background(styles.Theme.SelectedBackground).Render("Save"))
	} else {
		s.WriteString(lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render("Save"))
	}
	return s.String()
}

func main() {
	_ = lipgloss.HasDarkBackground()
	m := InitialModel()

	var logFile *os.File
	var dump *os.File
	if isDebugEnabled() {
		if _, ok := os.LookupEnv("DEBUG"); ok {
			var err error
			dump, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				os.Exit(1)
			}
		}
		m.msgDump = dump
		var err error
		logFile, err = setupLogging()
		if err != nil {
			fmt.Printf("Error setting up logging: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithReportFocus())
	_, err := p.Run()
	if err != nil {
		if isDebugEnabled() {
			log.Debug(err)
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func isDebugEnabled() bool {
	return os.Getenv("DEBUG") != ""
}

func setupLogging() (*os.File, error) {
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("error opening file for logging: %w", err)
	}
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)
	return f, nil
}

// UTILS

func StringToShortcode(input string) string {
	// Hash the input string
	hash := sha256.Sum256([]byte(input))

	// Encode the first 6 bytes of the hash to base64
	encoded := base64.RawURLEncoding.EncodeToString(hash[:6])

	// Return the first 6 characters
	return encoded[:6]
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func truncate(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

func convertSlice[T, U any](input []T, convert func(T) U) []U {
	result := make([]U, len(input))
	for i, v := range input {
		result[i] = convert(v)
	}
	return result
}

func debug(format string, args ...any) {
	if isDebugEnabled() {
		log.Helper()
		log.Debugf(format, args...)
	}
}

func getGitAuthorEmail() (string, error) {
	cmd := exec.Command("git", "config", "user.email")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
