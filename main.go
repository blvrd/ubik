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
	"path/filepath"
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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	// "github.com/go-git/go-git/v5/storage"
	"github.com/google/uuid"
	// "github.com/go-git/go-git/v5/plumbing/object"
	// "github.com/go-git/go-git/v5/config"
	"github.com/muesli/reflow/truncate"
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

type actionPersistedMsg struct {
	Action      Action
	IsNewAction bool
}

func persistAction(action Action, repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		jsonData, err := json.Marshal(action)
		if err != nil {
			debug("%#v", err.Error())
			return err
		}

		obj := repo.Storer.NewEncodedObject()
		obj.SetType(plumbing.BlobObject)
		obj.SetSize(int64(len(jsonData)))
		writer, err := obj.Writer()
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		_, err = writer.Write(jsonData)
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		err = writer.Close()
		if err != nil {
			debug("%#v", err.Error())
			return err
		}

		hash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		ref := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/ubik/actions/%s", action.Id), hash.String())
		err = repo.Storer.SetReference(ref)

		if err != nil {
			debug("%#v", err.Error())
			panic(err)
		}
		return actionPersistedMsg{Action: action}
	}
}

type issuePersistedMsg struct {
	Issue          Issue
	IsNewIssue     bool
	ScrollToBottom bool
}

func persistIssue(issue Issue, repo *git.Repository) tea.Cmd {
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

		obj := repo.Storer.NewEncodedObject()
		obj.SetType(plumbing.BlobObject)
		obj.SetSize(int64(len(jsonData)))
		writer, err := obj.Writer()
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		_, err = writer.Write(jsonData)
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		err = writer.Close()
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		hash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			debug("%#v", err.Error())
			return err
		}
		ref := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/ubik/issues/%s", issue.Id), hash.String())
		err = repo.Storer.SetReference(ref)

		if err != nil {
			debug("%#v", err.Error())
			return err
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
	issuesDeleteConfirmationPath
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
	actionsIndexPath
	actionsShowPath
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
	Path                      int
	Up                        key.Binding
	Down                      key.Binding
	Left                      key.Binding
	Right                     key.Binding
	Help                      key.Binding
	Quit                      key.Binding
	ForceQuit                 key.Binding
	Suspend                   key.Binding
	Back                      key.Binding
	IssueNewForm              key.Binding
	IssueEditForm             key.Binding
	IssueShowFocus            key.Binding
	IssueStatusDone           key.Binding
	IssueStatusWontDo         key.Binding
	IssueStatusInProgress     key.Binding
	IssueCommentFormFocus     key.Binding
	IssueDelete               key.Binding
	IssueConfirmDelete        key.Binding
	CommitShowFocus           key.Binding
	CommitExpandActionDetails key.Binding
	NextInput                 key.Binding
	Submit                    key.Binding
	NextPage                  key.Binding
	PrevPage                  key.Binding
	RunAction                 key.Binding
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
	case matchRoute(k.Path, actionsIndexPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.RunAction, k.CommitShowFocus},
		}
	case matchRoute(k.Path, actionsShowPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.RunAction, k.CommitExpandActionDetails},
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
	return fmt.Sprintf("%s\n%s\n%s", i.Title, labels, i.Status)
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
	title := fmt.Sprintf("%s %s", i.Status.Icon(), titleFn(truncate.StringWithTail(i.Title, 50, "...")))
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
		actionsShowPath:
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
	loaded       bool
	path         int
	underlayPath int // determines what view to display under the overlay
	issueIndex   list.Model
	issueShow    issueShow
	issueForm    issueForm
	commentForm  commentForm
	commitIndex  list.Model
	commitShow   commitShow
	err          error
	help         help.Model
	styles       Styles
	tabs         []string
	msgDump      io.Writer
	layout       Layout
	router       *Router
	gitConfig    *config.Config
	repo         *git.Repository
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
		cmd = persistIssue(currentIssue, m.repo)
	} else {
		description := form.descriptionInput.Value()

		newIssue := Issue{
			Shortcode:   "xxxxxx",
			Title:       title,
			Description: description,
			Labels:      labels,
			Status:      todo,
			Author:      m.gitConfig.User.Email,
		}
		cmd = persistIssue(newIssue, m.repo)
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
	router.AddRoute(issuesDeleteConfirmationPath, issuesDeleteHandler)
	router.AddRoute(issuesCommentContentPath, issuesCommentContentHandler)
	router.AddRoute(issuesCommentConfirmationPath, issuesCommentConfirmationHandler)
	router.AddRoute(issuesEditTitlePath, issuesEditTitleHandler)
	router.AddRoute(issuesEditLabelsPath, issuesEditLabelsHandler)
	router.AddRoute(issuesEditDescriptionPath, issuesEditDescriptionHandler)
	router.AddRoute(issuesEditConfirmationPath, issuesEditConfirmationHandler)
	router.AddRoute(issuesNewTitlePath, issuesNewTitleHandler)
	router.AddRoute(issuesNewLabelsPath, issuesNewLabelsHandler)
	router.AddRoute(issuesNewDescriptionPath, issuesNewDescriptionHandler)
	router.AddRoute(issuesNewConfirmationPath, issuesNewConfirmationHandler)
	router.AddRoute(actionsIndexPath, actionsIndexHandler)
	router.AddRoute(actionsShowPath, actionsShowHandler)

	return Model{
		path:        issuesIndexPath,
		help:        helpModel,
		styles:      DefaultStyles(),
		tabs:        []string{"Issues", "Actions"},
		layout:      layout,
		issueIndex:  issueList,
		commitIndex: commitList,
		commentForm: newCommentForm(),
		issueForm:   newIssueForm("", "", "", []string{}, false),
		issueShow:   newIssueShow(Issue{}, layout),
		router:      router,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Sequence(getGitRepo)
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

func CustomFilter(term string, targets []string) []list.Rank {
	terms := strings.Fields(term)
	filters := map[string]func(string, []string) []list.Rank{
		"label:":  LabelFilter,
		"status:": StatusFilter,
	}

	// Create a map to store all matching ranks
	allRanks := make(map[int]list.Rank)

	for _, t := range terms {
		var ranks []list.Rank
		for prefix, filterFunc := range filters {
			if strings.HasPrefix(t, prefix) {
				ranks = filterFunc(t, targets)
				break
			}
		}
		if len(ranks) == 0 {
			ranks = list.DefaultFilter(t, targets)
		}

		// Intersect the new ranks with existing ones
		if len(allRanks) == 0 {
			for _, r := range ranks {
				allRanks[r.Index] = r
			}
		} else {
			for i := range allRanks {
				if !containsRank(ranks, i) {
					delete(allRanks, i)
				}
			}
		}

		if len(allRanks) == 0 {
			break // No matches, no need to continue
		}
	}

	// Convert map to slice
	result := make([]list.Rank, 0, len(allRanks))
	for _, r := range allRanks {
		result = append(result, r)
	}

	return result
}

func containsRank(ranks []list.Rank, index int) bool {
	for _, r := range ranks {
		if r.Index == index {
			return true
		}
	}
	return false
}

func LabelFilter(term string, targets []string) []list.Rank {
	var labelTargets []string
	labelTerm := strings.TrimPrefix(term, "label:")

	for _, t := range targets {
		labelsPart := strings.Split(t, "\n")[1]
		labelTargets = append(labelTargets, labelsPart)
	}

	return list.DefaultFilter(labelTerm, labelTargets)
}

func StatusFilter(term string, targets []string) []list.Rank {
	var statusTargets []string
	statusTerm := strings.TrimPrefix(term, "status:")

	for _, t := range targets {
		statusPart := strings.Split(t, "\n")[2]
		statusTargets = append(statusTargets, statusPart)
	}

	return list.DefaultFilter(statusTerm, statusTargets)
}

func issuesIndexHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.issueIndex.SettingFilter() {
		m.issueIndex.Filter = CustomFilter
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
			cmd = persistIssue(currentIssue, m.repo)
			return m, cmd
		case key.Matches(msg, keys.IssueStatusWontDo):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == wontDo {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = wontDo
			}
			cmd = persistIssue(currentIssue, m.repo)
			return m, cmd
		case key.Matches(msg, keys.IssueStatusInProgress):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			if currentIssue.Status == inProgress {
				currentIssue.Status = todo
			} else {
				currentIssue.Status = inProgress
			}
			cmd = persistIssue(currentIssue, m.repo)
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
			m.underlayPath = m.path
			m.path = issuesDeleteConfirmationPath
			m.UpdateLayout(m.layout.TerminalSize)
			return m, cmd
		case key.Matches(msg, keys.NextPage):
			m.path = actionsIndexPath
			return m, nil
		case key.Matches(msg, keys.PrevPage):
			m.path = actionsIndexPath
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
			cmd = persistIssue(currentIssue, m.repo)
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
			cmd = persistIssue(currentIssue, m.repo)
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
			cmd = persistIssue(currentIssue, m.repo)
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
			m.UpdateLayout(m.layout.TerminalSize)
			return m, nil
		case key.Matches(msg, keys.IssueCommentFormFocus):
			m.commentForm = newCommentForm()
			cmd = m.commentForm.Init()
			m.path = issuesCommentContentPath
			m.UpdateLayout(m.layout.TerminalSize)
			m.issueShow = newIssueShow(m.issueIndex.SelectedItem().(Issue), m.layout)
			m.issueShow.viewport.GotoBottom()
			return m, cmd
		case key.Matches(msg, keys.IssueDelete):
			selectedItem := m.issueIndex.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}
			m.underlayPath = m.path
			m.path = issuesDeleteConfirmationPath
			// m.UpdateLayout(m.layout.TerminalSize)
			return m, cmd
		}
	}

	m.issueShow.viewport, cmd = m.issueShow.viewport.Update(msg)
	return m, cmd
}

func issuesDeleteHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.IssueConfirmDelete):
			selectedItem := m.issueIndex.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}
			issue := selectedItem.(Issue)
			issue.DeletedAt = time.Now().UTC()
			cmd = persistIssue(issue, m.repo)
			m.path = issuesIndexPath
			m.underlayPath = 0
			m.UpdateLayout(m.layout.TerminalSize)
			return m, cmd
		case key.Matches(msg, keys.Back):
			m.path = issuesIndexPath
			m.UpdateLayout(m.layout.TerminalSize)
			return m, cmd
		}
	}

	return m, cmd
}

func issuesCommentContentHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesCommentConfirmationHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			m.commentForm = newCommentForm()
			m.issueShow = newIssueShow(currentIssue, m.layout)
			m.path = issuesShowPath
		case key.Matches(msg, keys.NextInput):
			cmd = m.commentForm.contentInput.Focus()
			m.commentForm.confirming = false
			m.path = issuesCommentContentPath
		case key.Matches(msg, keys.Submit):
			cmd = m.commentForm.Submit
		}
	}

	return m, cmd
}

func issuesEditTitleHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesEditLabelsHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesEditDescriptionHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesEditConfirmationHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesNewTitleHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesNewLabelsHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

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
}

func issuesNewDescriptionHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m.path = issuesIndexPath
			return m, cmd
		case key.Matches(msg, keys.NextInput):
			m.path = issuesNewConfirmationPath
			m.issueForm.descriptionInput.Blur()
			return m, cmd
		}
	}

	m.issueForm.descriptionInput, cmd = m.issueForm.descriptionInput.Update(msg)
	return m, cmd
}

func issuesNewConfirmationHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m.path = issuesIndexPath
			return m, cmd
		case key.Matches(msg, keys.NextInput):
			m.path = issuesNewTitlePath
			cmd = m.issueForm.titleInput.Focus()
			return m, cmd
		case key.Matches(msg, keys.Submit):
			return m, m.submitIssueForm()
		}
	}

	return m, cmd
}

func actionsIndexHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case actionResult:
		return m, persistAction(Action(msg), m.repo)
	case actionPersistedMsg:
		action := msg.Action
		var commit Commit
		var commitIndex int
		for i, c := range m.commitIndex.Items() {
			if c.(Commit).Hash == action.CommitId {
				commit = c.(Commit)
				commitIndex = i
				break
			}
		}
		updatedActions := make([]Action, len(commit.LatestActions))
		for i, c := range commit.LatestActions {
			if action.Id == c.Id {
				updatedActions[i] = action
			} else {
				updatedActions[i] = c
			}
		}
		commit.LatestActions = updatedActions
		m.commitIndex.SetItem(commitIndex, commit)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, keys.RunAction):
			commit := m.commitIndex.SelectedItem().(Commit)
			var cmds []tea.Cmd
			actions := NewActions(commit)
			cmds = append(cmds, commit.DeleteExistingActions())
			commit.LatestActions = actions

			for i, action := range commit.LatestActions {
				action.ExecutionPosition = i
				cmds = append(cmds, RunAction(action))
			}
			m.commitIndex.SetItem(m.commitIndex.Index(), commit)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, keys.CommitShowFocus):
			m.path = actionsShowPath
			m.UpdateLayout(m.layout.TerminalSize)
			m.commitShow = newCommitShow(m.commitIndex.SelectedItem().(Commit), m.layout, false)
			return m, cmd
		case key.Matches(msg, keys.NextPage):
			m.path = issuesIndexPath
			return m, nil
		case key.Matches(msg, keys.PrevPage):
			m.path = issuesIndexPath
			return m, nil
		}
	}

	m.commitIndex, cmd = m.commitIndex.Update(msg)
	return m, cmd
}

func actionsShowHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m.path = actionsIndexPath
		case key.Matches(msg, keys.RunAction):
			commit := m.commitIndex.SelectedItem().(Commit)
			var cmds []tea.Cmd
			actions := NewActions(commit)
			cmds = append(cmds, commit.DeleteExistingActions())
			commit.LatestActions = actions

			for i, action := range commit.LatestActions {
				action.ExecutionPosition = i
				cmds = append(cmds, RunAction(action))
			}
			m.commitIndex.SetItem(m.commitIndex.Index(), commit)
			m.UpdateLayout(m.layout.TerminalSize)
			m.commitShow = newCommitShow(commit, m.layout, false)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, keys.CommitExpandActionDetails):
			commit := m.commitIndex.SelectedItem().(Commit)
			expand := !m.commitShow.expandActionDetails
			m.UpdateLayout(m.layout.TerminalSize)
			m.commitShow = newCommitShow(commit, m.layout, expand)
		}
	case actionResult:
		return m, persistAction(Action(msg), m.repo)
	case actionPersistedMsg:
		action := msg.Action
		var commit Commit
		var commitIndex int
		for i, c := range m.commitIndex.Items() {
			if c.(Commit).Hash == action.CommitId {
				commit = c.(Commit)
				commitIndex = i
				break
			}
		}
		updatedActions := make([]Action, len(commit.LatestActions))
		for i, c := range commit.LatestActions {
			if action.Id == c.Id {
				updatedActions[i] = action
			} else {
				updatedActions[i] = c
			}
		}
		commit.LatestActions = updatedActions
		m.commitIndex.SetItem(commitIndex, commit)
		m.UpdateLayout(m.layout.TerminalSize)
		m.commitShow = newCommitShow(commit, m.layout, m.commitShow.expandActionDetails)
	}

	m.commitShow.viewport, cmd = m.commitShow.viewport.Update(msg)
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
		if m.repo == nil {
			return m, nil
		}

		return m, tea.Sequence(getCommits(m.repo))
	case tea.BlurMsg:
		return m, nil
	case GitRepoReadyMsg:
		m.repo = msg.repo
		m.gitConfig = msg.cfg
		return m, tea.Sequence(getIssues(m.repo), getCommits(m.repo))
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
			Author:  m.gitConfig.User.Email,
			Content: msg.contentInput.Value(),
		})

		cmd = persistIssue(currentIssue, m.repo)
		return m, cmd
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

	return m.router.Route(m, msg)
}

type actionResult Action

func RunAction(action Action) tea.Cmd {
	return func() tea.Msg {
		result, err := executeActionUsingArchive(action)
		action.Output = result
		action.FinishedAt = time.Now().UTC()
		if err != nil {
			debug("Action failed: %v", err)
			action.Status = failed
			return actionResult(action)
		}
		action.Status = succeeded
		return actionResult(action)
	}
}

func executeActionUsingArchive(action Action) (string, error) {
	tempDir, err := os.MkdirTemp("", "action-archive-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// #nosec G204
	archiveCmd := exec.Command("git", "archive", "--format=tar", action.CommitId)
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

	action.Command.Dir = tempDir
	output, err := runCommandWithOutput(action.Command)
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
		IssueConfirmDelete: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm issue deletion"),
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
		RunAction: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "run action"),
		),
		CommitShowFocus: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "more info"),
		),
		CommitExpandActionDetails: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "expand action details"),
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
	layout := windowStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(m.layout.LeftSize.Width).Height(m.layout.LeftSize.Height).Render(left),
			right,
		),
		footer,
	))

	if m.path == issuesDeleteConfirmationPath {
		overlayBoxStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(m.styles.Theme.FaintBorder).Foreground(m.styles.Theme.PrimaryText).Width(40).Height(4).Padding(1)
		issue := m.issueIndex.SelectedItem().(Issue)
		overlayContent := overlayBoxStyle.Render(fmt.Sprintf("Delete issue #%s?", issue.Shortcode))
		return PlaceOverlay((m.layout.TerminalSize.Width/2 - 20), (m.layout.TerminalSize.Height/2 - 3), overlayContent, layout, false)
	} else {
		return layout
	}
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

	switch m.underlayPath {
	case issuesShowPath:
		right = m.issueShowView()
	case issuesCommentContentPath, issuesCommentConfirmationPath:
		right = lipgloss.JoinVertical(lipgloss.Left, m.issueShowView(), m.commentFormView())
	}

	return m.renderMainLayout(m.renderTabs("Issues"), left, right, m.help.View(m.HelpKeys()))
}

func (m Model) renderActionsView() string {
	left := m.commitIndex.View()
	var right string
	if m.path == actionsShowPath {
		right = m.commitShowView()
	}
	return m.renderMainLayout(m.renderTabs("Actions"), left, right, m.help.View(m.HelpKeys()))
}

func (m Model) View() string {
	if !m.loaded {
		return "Loading..."
	}

	var view string
	switch m.path {
	case issuesIndexPath, issuesShowPath, issuesDeleteConfirmationPath, issuesCommentContentPath, issuesCommentConfirmationPath,
		issuesEditTitlePath, issuesEditLabelsPath, issuesEditDescriptionPath, issuesEditConfirmationPath,
		issuesNewTitlePath, issuesNewLabelsPath, issuesNewDescriptionPath, issuesNewConfirmationPath:
		view = m.renderIssuesView()
	case actionsIndexPath, actionsShowPath:
		view = m.renderActionsView()
	}

	return docStyle.Render(view)
}

type Commit struct {
	Hash            string    `json:"id"`
	AbbreviatedHash string    `json:"abbreviatedId"`
	AuthorEmail     string    `json:"author_email"`
	AuthorName      string    `json:"author_name"`
	Message         string    `json:"message"`
	Timestamp       time.Time `json:"timestamp"`
	LatestActions   []Action  `json:"latestActions"`
	Repo            *git.Repository
}

func (c Commit) AggregateActionStatus() ActionStatus {
	if len(c.LatestActions) == 0 {
		return ""
	}

	hasFailedAction := false
	for _, action := range c.LatestActions {
		switch action.Status {
		case running:
			return running
		case failed:
			if action.Optional {
				continue
			}
			hasFailedAction = true
		case succeeded:
			// Continue actioning other actions
		default:
			return running
		}
	}

	if hasFailedAction {
		return failed
	}
	return succeeded
}

func (c Commit) DeleteExistingActions() tea.Cmd {
	var cmds []tea.Cmd

	for _, action := range c.LatestActions {
		cmds = append(cmds, action.Delete(c.Repo))
	}

	return tea.Batch(cmds...)
}

type ActionStatus string

func (c ActionStatus) Icon() string {
	icons := map[ActionStatus]string{
		running:   "[⋯]",
		failed:    "[×]",
		succeeded: "[✓]",
	}
	return lipgloss.NewStyle().Foreground(c.color()).Render(icons[c])
}

func (c ActionStatus) PrettyString() string {
	return lipgloss.NewStyle().Foreground(c.color()).Render(string(c))
}

func (c ActionStatus) color() lipgloss.AdaptiveColor {
	colors := map[ActionStatus]lipgloss.AdaptiveColor{
		running:   styles.Theme.YellowText,
		failed:    styles.Theme.RedText,
		succeeded: styles.Theme.GreenText,
	}
	return colors[c]
}

const (
	failed    ActionStatus = "failed"
	succeeded ActionStatus = "succeeded"
	running   ActionStatus = "running"
)

type Action struct {
	Command           *exec.Cmd    `json:"-"`
	Id                string       `json:"id"`
	CommitId          string       `json:"commitId"`
	Status            ActionStatus `json:"status"`
	Actioner          string       `json:"actioner"`
	Name              string       `json:"name"`
	Output            string       `json:"output"`
	StartedAt         time.Time    `json:"startedAt"`
	FinishedAt        time.Time    `json:"finishedAt"`
	Optional          bool         `json:"optional"`
	ExecutionPosition int          `json:"executionPosition"`
}

func NewActions(commit Commit) []Action {
	return []Action{
		Action{
			Id:        uuid.NewString(),
			Status:    running,
			CommitId:  commit.Hash,
			Command:   exec.Command("go", "test"),
			Name:      "Tests ('go test')",
			StartedAt: time.Now().UTC(),
		},
		Action{
			Id:        uuid.NewString(),
			Status:    running,
			CommitId:  commit.Hash,
			Command:   exec.Command("gosec", "./"),
			Name:      "Security ('gosec')",
			StartedAt: time.Now().UTC(),
			Optional:  true,
		},
	}
}

func (c Action) Delete(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		refName := plumbing.ReferenceName(fmt.Sprintf("refs/ubik/actions/%s", c.Id))
		err := repo.Storer.RemoveReference(refName)

		if err != nil {
			debug("%#v", err)
			return err
		}

		return nil
	}
}

func (c Action) ElapsedTime() time.Duration {
	return c.FinishedAt.Sub(c.StartedAt)
}

func (c Commit) FilterValue() string {
	return c.Hash
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

	if c.AuthorEmail == "" {
		author = "unknown"
	} else {
		author = c.AuthorEmail
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

	title := fmt.Sprintf("%s", titleFn(c.AbbreviatedHash, truncate.StringWithTail(c.Message, 50, "...")))

	if len(c.LatestActions) > 0 {
		title = fmt.Sprintf("%s %s", title, c.AggregateActionStatus().Icon())
	}

	description := fmt.Sprintf("committed at %s by %s", c.Timestamp.Format(time.RFC822), author)
	item := lipgloss.JoinVertical(lipgloss.Left, title, description)

	fmt.Fprintf(w, item)
}

type GitRepoReadyMsg struct {
	repo *git.Repository
	cfg  *config.Config
}

func getGitRepo() tea.Msg {
	repo, _ := git.PlainOpen(".")
	cfg, err := repo.ConfigScoped(config.GlobalScope)
	if err != nil {
		panic(err)
	}
	return GitRepoReadyMsg{repo, cfg}
}

type CommitListReadyMsg []Commit

func getCommits(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		var commits []Commit
		actions := make(map[string][]Action)

		refs, err := repo.References()

		if err != nil {
			panic(err)
		}

		refPath := "refs/ubik/actions"

		err = refs.ForEach(func(ref *plumbing.Reference) error {
			if !strings.HasPrefix(ref.Name().String(), refPath) {
				return nil
			}
			obj, err := repo.Object(plumbing.AnyObject, ref.Hash())
			if err != nil {
				return nil
			}

			if blob, ok := obj.(*object.Blob); ok {
				// Read the contents of the blob
				blobReader, _ := blob.Reader()
				b, err := io.ReadAll(blobReader)
				if err != nil {
					fmt.Printf("Error reading blob content: %v\n", err)
					return nil
				}

				var action Action
				err = json.Unmarshal(b, &action)
				if err != nil {
					panic(err)
				}

				actions[action.CommitId] = append(actions[action.CommitId], action)
			}
			return nil
		})

		if err != nil {
			panic(err)
		}

		logOptions := git.LogOptions{
			Order: git.LogOrderCommitterTime,
		}

		gitCommits, err := repo.Log(&logOptions)

		if err != nil {
			panic(err)
		}

		err = gitCommits.ForEach(func(c *object.Commit) error {
			id := c.Hash.String()
			commitActions := actions[id]
			slices.SortFunc(commitActions, func(a, b Action) int {
				return a.ExecutionPosition - b.ExecutionPosition
			})
			commits = append(commits, Commit{
				Hash:            id,
				AbbreviatedHash: id[:8],
				AuthorEmail:     c.Author.Email,
				Timestamp:       c.Author.When,
				Message:         strings.TrimSuffix(c.Message, "\n"),
				LatestActions:   actions[id],
				Repo:            repo,
			})
			return nil
		})

		if err != nil {
			panic(err)
		}

		return CommitListReadyMsg(commits)
	}
}

type IssuesReadyMsg []Issue

func getIssues(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		var issues []Issue

		refs, err := repo.References()
		if err != nil {
			panic(err)
		}

		refPath := "refs/ubik/issues"

		err = refs.ForEach(func(ref *plumbing.Reference) error {
			if !strings.HasPrefix(ref.Name().String(), refPath) {
				return nil
			}

			obj, err := repo.Object(plumbing.AnyObject, ref.Hash())
			if err != nil {
				return nil
			}

			if blob, ok := obj.(*object.Blob); ok {
				// Read the contents of the blob
				blobReader, _ := blob.Reader()
				b, err := io.ReadAll(blobReader)
				if err != nil {
					fmt.Printf("Error reading blob content: %v\n", err)
					return nil
				}

				var issue Issue
				err = json.Unmarshal(b, &issue)
				if err != nil {
					panic(err)
				}

				if issue.DeletedAt.IsZero() {
					issues = append(issues, issue)
				}
			}
			return nil
		})

		if err != nil {
			panic(err)
		}

		sortedIssues := SortIssues(issues)

		return IssuesReadyMsg(sortedIssues)
	}
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
	commit              Commit
	viewport            viewport.Model
	expandActionDetails bool
}

func newCommitShow(commit Commit, layout Layout, expandActionDetails bool) commitShow {
	var s strings.Builder

	viewport := viewport.New(layout.RightSize.Width, layout.RightSize.Height)
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf("%s", commit.AbbreviatedHash))
	var header string
	if len(commit.LatestActions) > 0 {
		header = fmt.Sprintf("%s %s\nStatus: %s\n\n", identifier, commit.Message, commit.AggregateActionStatus().PrettyString())
	} else {
		header = fmt.Sprintf("%s %s\n\nNo actions yet.", identifier, commit.Message)
	}
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString("\n")

	for _, action := range commit.LatestActions {
		s.WriteString(fmt.Sprintf("\n%s %s", action.Status.Icon(), action.Name))
		if action.Optional {
			s.WriteString(lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(" (optional)"))
		}
		if expandActionDetails {
			s.WriteString(
				lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(
					fmt.Sprintf(" finished in %s\n\n", action.ElapsedTime()),
				),
			)
			s.WriteString(fmt.Sprintf("\n%s\n\n", action.Output))
		}
	}

	viewport.SetContent(s.String())

	return commitShow{
		commit:              commit,
		viewport:            viewport,
		expandActionDetails: expandActionDetails,
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

	if !insideGitRepository() {
		fmt.Print("Error: ubik must be run inside a git repository\n")
		os.Exit(1)
	}

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

func insideGitRepository() bool {
	gitDir := filepath.Join(".", ".git")
	_, err := os.Stat(gitDir)
	return !os.IsNotExist(err)
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

func UUIDToShortcode(id uuid.UUID) string {
	// Take the first 6 bytes of the UUID
	shortBytes := id[:6]

	// Encode to base64
	encoded := base64.RawURLEncoding.EncodeToString(shortBytes)

	// Return the first 6 characters
	return encoded[:6]
}

func StringToShortcode(input string) string {
	// Hash the input string
	hash := sha256.Sum256([]byte(input))

	// Encode the first 6 bytes of the hash to base64
	encoded := base64.RawURLEncoding.EncodeToString(hash[:6])

	// Return the first 6 characters
	return encoded[:6]
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
