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
	bl "github.com/winder/bubblelayout"
)

type checkPersistedMsg struct {
	Check      Check
	IsNewCheck bool
}

func persistCheck(check Check) tea.Cmd {
	return func() tea.Msg {
		var newCheck bool

		if check.Id == "" {
			newCheck = true
		} else {
			newCheck = false
		}

		if newCheck {
			id := uuid.NewString()
			check.Id = id
		}

		jsonData, err := json.Marshal(check)
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

		cmd = exec.Command("git", "update-ref", fmt.Sprintf("refs/ubik/checks/%s", check.Id), hash)
		err = cmd.Run()

		if err != nil {
			log.Fatalf("%#v", err.Error())
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
	issuesIndexPath               string = "/issues/index"
	issuesShowPath                string = "/issues/show"
	issuesCommentContentPath      string = "/issues/show/comments/new/content"
	issuesCommentConfirmationPath string = "/issues/show/comments/new/confirmation"
	issuesEditTitlePath           string = "/issues/edit/title"
	issuesEditDescriptionPath     string = "/issues/edit/description"
	issuesEditConfirmationPath    string = "/issues/edit/confirmation"
	issuesNewTitlePath            string = "/issues/new/title"
	issuesNewDescriptionPath      string = "/issues/new/description"
	issuesNewConfirmationPath     string = "/issues/new/confirmation"
	checksIndexPath               string = "/checks/index"
	checksShowPath                string = "/checks/show"
)

func matchRoute(currentRoute, route string) bool {
	return route == currentRoute
}

// func action(msg tea.Msg, m Model, currentPath string) {
//   // find controller
//   // call action, which returns a func
//   // the func returns model, cmd
// }
//
// type IssuesController struct {}
//
// func (IssuesController) Index() {
//
// }

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

type status int

const (
	todo       status = 1
	inProgress status = 2
	done       status = 3
	wontDo     status = 4
)

type keyMap struct {
	Path                  string
	Up                    key.Binding
	Down                  key.Binding
	Left                  key.Binding
	Right                 key.Binding
	Help                  key.Binding
	Quit                  key.Binding
	Suspend               key.Binding
	Back                  key.Binding
	IssueNewForm          key.Binding
	IssueEditForm         key.Binding
	IssueDetailFocus      key.Binding
	IssueStatusDone       key.Binding
	IssueStatusWontDo     key.Binding
	IssueStatusInProgress key.Binding
	IssueCommentFormFocus key.Binding
	IssueDelete           key.Binding
	CommitDetailFocus     key.Binding
	NextInput             key.Binding
	Submit                key.Binding
	NextPage              key.Binding
	PrevPage              key.Binding
	RunCheck              key.Binding
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
			{k.IssueNewForm, k.IssueDetailFocus},
			{k.IssueStatusDone, k.IssueStatusWontDo},
			{k.IssueStatusInProgress, k.IssueCommentFormFocus},
			{k.IssueDelete},
		}
	case matchRoute(k.Path, issuesShowPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.IssueEditForm, k.Back},
			{k.IssueNewForm, k.IssueDetailFocus},
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
			{k.RunCheck, k.CommitDetailFocus},
		}
	case matchRoute(k.Path, checksShowPath):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.RunCheck},
			{k.Back},
		}
	}

	return bindings
}

type Issue struct {
	Id          string    `json:"id"`
	Shortcode   string    `json:"shortcode"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      status    `json:"status"`
	Comments    []Comment `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

func (i Issue) FilterValue() string {
	return i.Title
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

	var status string

	switch i.Status {
	case todo:
		status = "[·]"
	case inProgress:
		status = lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("[⋯]")
	case wontDo:
		status = lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("[×]")
	case done:
		status = lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("[✓]")
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
	title := fmt.Sprintf("%s %s", status, titleFn(fmt.Sprintf("#%s", i.Shortcode), truncate(i.Title, 50)))

	description := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("created by %s at %s", i.Author, i.CreatedAt.Format(time.RFC822)))
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

type Layout struct {
	bl.BubbleLayout

	HeaderID bl.ID
	RightID  bl.ID
	LeftID   bl.ID
	FooterID bl.ID

	LeftSize   bl.Size
	RightSize  bl.Size
	HeaderSize bl.Size
	FooterSize bl.Size
}

type Model struct {
	loaded      bool
	path        string
	issueIndex  list.Model
	issueShow   issueShowModel
	issueForm   issueFormModel
	commitIndex list.Model
	commitShow  commitShowModel
	err         error
	help        help.Model
	styles      Styles
	tabs        []string
	Layout
}

var (
	inactiveTabBorder = lipgloss.NormalBorder()
	activeTabBorder   = lipgloss.NormalBorder()
	docStyle          = lipgloss.NewStyle().Padding(0)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(styles.Theme.FaintBorder).Padding(0, 0)
	activeTabStyle    = lipgloss.NewStyle().Border(activeTabBorder, true).BorderForeground(styles.Theme.PrimaryBorder).Padding(0, 0)
	windowStyle       = lipgloss.NewStyle().Padding(0)
	helpStyle         = lipgloss.NewStyle().Padding(0, 0)
	footerHeight      = helpStyle.GetVerticalFrameSize() + 1 // 1 row for the context
)

func InitialModel() Model {
	blLayout := bl.New()
	headerId := blLayout.Add("dock north 4!")
	leftId := blLayout.Add("width 80")
	rightId := blLayout.Add("grow")
	footerId := blLayout.Add("dock south 2!")

	layout := Layout{
		BubbleLayout: blLayout,
		HeaderID:     headerId,
		LeftID:       leftId,
		RightID:      rightId,
		FooterID:     footerId,
	}

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

	return Model{
		path:        issuesIndexPath,
		help:        helpModel,
		styles:      DefaultStyles(),
		tabs:        []string{"Issues", "Checks"},
		Layout:      layout,
		issueIndex:  issueList,
		commitIndex: commitList,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(getIssues, getCommits)
}

type layoutMsg Layout

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	keys := m.HelpKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Suspend):
			return m, tea.Suspend
		}
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.loaded = true
		}

		return m, func() tea.Msg {
			return m.Layout.Resize(msg.Width, msg.Height)
		}
	case tea.FocusMsg:
		return m, tea.Batch(getIssues, getCommits)
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
	case bl.BubbleLayoutMsg:
		m.LeftSize, _ = msg.Size(m.LeftID)
		m.RightSize, _ = msg.Size(m.RightID)
		m.HeaderSize, _ = msg.Size(m.HeaderID)
		m.FooterSize, _ = msg.Size(m.FooterID)

		m.issueIndex.SetSize(m.LeftSize.Width, m.LeftSize.Height)
		m.commitIndex.SetSize(m.LeftSize.Width, m.LeftSize.Height)
	case commentFormModel:
		currentIssue := m.issueIndex.SelectedItem().(Issue)
		currentIssue.Comments = append(currentIssue.Comments, Comment{
			Author:  "garrett@blvrd.co",
			Content: msg.contentInput.Value(),
		})

		cmd = persistIssue(currentIssue)
		return m, cmd
	case issueFormModel:
		if msg.editing {
			currentIssue := m.issueIndex.SelectedItem().(Issue)
			currentIssue.Title = msg.titleInput.Value()
			currentIssue.Description = msg.descriptionInput.Value()
			cmd = persistIssue(currentIssue)
		} else {
			newIssue := Issue{
				Shortcode:   "xxxxxx",
				Title:       msg.titleInput.Value(),
				Description: msg.descriptionInput.Value(),
				Status:      todo,
				Author:      "garrett@blvrd.co",
			}
			cmd = persistIssue(newIssue)
		}

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
			m.issueShow = issueShowModel{issue: msg.Issue}
			m.issueShow.commentForm = NewCommentFormModel()
			m.issueShow.Init(m)
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
	case matchRoute(m.path, issuesIndexPath):
		if m.issueIndex.SettingFilter() {
			m.issueIndex, cmd = m.issueIndex.Update(msg)
			return m, cmd
		}

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
				m.issueShow = issueShowModel{issue: m.issueIndex.SelectedItem().(Issue)}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.commentForm.Init()
				m.issueShow.Init(m)
				m.issueShow.viewport.GotoBottom()
				m.path = issuesCommentContentPath
			case key.Matches(msg, keys.IssueDetailFocus):
				m.issueShow = issueShowModel{issue: m.issueIndex.SelectedItem().(Issue)}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.commentForm.Init()
				m.issueShow.Init(m)
				m.path = issuesShowPath
			case key.Matches(msg, keys.IssueNewForm):
				m.issueForm = issueFormModel{editing: false}
				m.issueForm.Init("", "")
				cmd = m.issueForm.titleInput.Focus()
				m.path = issuesNewTitlePath
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
			case key.Matches(msg, keys.PrevPage):
				return m, nil
			}
		}

		m.issueIndex, cmd = m.issueIndex.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesShowPath):
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
				m.issueShow = issueShowModel{issue: currentIssue}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.commentForm.Init()
				m.issueShow.Init(m)
				cmd = persistIssue(currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueStatusWontDo):
				currentIssue := m.issueIndex.SelectedItem().(Issue)
				if currentIssue.Status == wontDo {
					currentIssue.Status = todo
				} else {
					currentIssue.Status = wontDo
				}
				m.issueShow = issueShowModel{issue: currentIssue}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.commentForm.Init()
				m.issueShow.Init(m)
				cmd = persistIssue(currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueStatusInProgress):
				currentIssue := m.issueIndex.SelectedItem().(Issue)
				if currentIssue.Status == inProgress {
					currentIssue.Status = todo
				} else {
					currentIssue.Status = inProgress
				}
				m.issueShow = issueShowModel{issue: currentIssue}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.commentForm.Init()
				m.issueShow.Init(m)
				cmd = persistIssue(currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueEditForm):
				selectedIssue := m.issueIndex.SelectedItem().(Issue)
				m.issueForm = issueFormModel{editing: true, identifier: selectedIssue.Shortcode}
				cmd = m.issueForm.Init(selectedIssue.Title, selectedIssue.Description)

				m.path = issuesEditTitlePath
				return m, cmd
			case key.Matches(msg, keys.Back):
				m.path = issuesIndexPath
			case key.Matches(msg, keys.IssueCommentFormFocus):
				m.issueShow = issueShowModel{issue: m.issueIndex.SelectedItem().(Issue)}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.commentForm.Init()
				m.issueShow.Init(m)
				m.issueShow.viewport.GotoBottom()
				m.path = issuesCommentContentPath
			}
		}

		m.issueShow.viewport, cmd = m.issueShow.viewport.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesCommentContentPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				currentIssue := m.issueIndex.SelectedItem().(Issue)
				m.issueShow = issueShowModel{issue: currentIssue}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.Init(m)
				m.path = issuesShowPath
			case key.Matches(msg, keys.NextInput):
				m.issueShow.commentForm.contentInput.Blur()
				m.path = issuesCommentConfirmationPath
			}
		}
		m.issueShow.commentForm.contentInput, cmd = m.issueShow.commentForm.contentInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesCommentConfirmationPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				currentIssue := m.issueIndex.SelectedItem().(Issue)
				m.issueShow = issueShowModel{issue: currentIssue}
				m.issueShow.commentForm = NewCommentFormModel()
				m.issueShow.Init(m)
				m.path = issuesShowPath
			case key.Matches(msg, keys.Submit):
				return m, m.issueShow.commentForm.Submit
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
				m.path = issuesEditDescriptionPath
				m.issueForm.titleInput.Blur()
				cmd = m.issueForm.descriptionInput.Focus()
				return m, cmd
			}
		}

		m.issueForm.titleInput, cmd = m.issueForm.titleInput.Update(msg)
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
				return m, m.issueForm.Submit
			}
		}

		m.issueForm, cmd = m.issueForm.Update(msg)
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
				m.path = issuesNewDescriptionPath
				m.issueForm.titleInput.Blur()
				cmd = m.issueForm.descriptionInput.Focus()
				return m, cmd
			}
		}

		m.issueForm.titleInput, cmd = m.issueForm.titleInput.Update(msg)
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
				return m, m.issueForm.Submit
			}
		}

		m.issueForm, cmd = m.issueForm.Update(msg)
		return m, cmd
	case matchRoute(m.path, checksIndexPath):
		if m.commitIndex.SettingFilter() {
			m.commitIndex, cmd = m.commitIndex.Update(msg)
			return m, cmd
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Down):
				m.commitIndex, cmd = m.commitIndex.Update(msg)
				return m, cmd
			case key.Matches(msg, keys.Up):
				m.commitIndex, cmd = m.commitIndex.Update(msg)
				return m, cmd
			case key.Matches(msg, keys.RunCheck):
				// commit := m.commitIndex.SelectedItem().(Commit)
				// commit.LatestChecks = []Check{}
				// m.commitIndex.SetItem(m.commitIndex.Index(), commit)
				// m.commitIndex, cmd = m.commitIndex.Update(msg)
				// m.commitShow = commitShowModel{commit: commit}
				// m.commitShow.Init(m)
				// checkCommands := []*exec.Cmd{
				// 	exec.Command("go", "test"),
				// 	exec.Command("./check.sh"),
				// }
				// var cmds []tea.Cmd
				// for _, command := range checkCommands {
				// 	commit.LatestChecks = append(commit.LatestChecks, Check{Status: running, CommitId: commit.Id, Name: command.String()})
				// 	cmds = append(cmds, RunCheck(commit.Id, command))
				// }
				// return m, tea.Batch(cmds...)
			case key.Matches(msg, keys.CommitDetailFocus):
				m.commitShow = commitShowModel{commit: m.commitIndex.SelectedItem().(Commit)}
				m.commitShow.Init(m)
				m.path = checksShowPath
				return m, cmd
			case key.Matches(msg, keys.NextPage):
				return m, nil
			case key.Matches(msg, keys.PrevPage):
				m.path = issuesIndexPath
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
			// case checkPersistedMsg:
			// 	var commit Commit
			// 	var commitIndex int
			// 	for i, c := range m.commitIndex.Items() {
			// 		if c.(Commit).Id == msg.Check.CommitId {
			// 			commit = c.(Commit)
			// 			commitIndex = i
			// 			break
			// 		}
			// 	}
			// 	commit.LatestChecks = append(commit.LatestChecks, Check{Status: msg.Check.Status, Output: msg.Check.Output, CommitId: commit.Id})
			// 	m.commitIndex.SetItem(commitIndex, commit)
			// 	m.commitIndex, cmd = m.commitIndex.Update(msg)
			// 	m.commitShow = commitShowModel{commit: commit}
			// 	m.commitShow.Init(m)
			// case checkResult:
			// 	check := Check{
			// 		Status:   msg.status,
			// 		CommitId: msg.commitHash,
			// 		Output:   msg.output,
			// 	}
			//
			// 	return m, persistCheck(check)
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
				checks := []Check{
					Check{
						Id:       uuid.NewString(),
						Status:   running,
						CommitId: commit.Id,
						Command:  exec.Command("go", "test"),
						Name:     "tests",
					},
					Check{
						Id:       uuid.NewString(),
						Status:   running,
						CommitId: commit.Id,
						Command:  exec.Command("./check.sh"),
						Name:     "another check",
					},
				}

				commit.LatestChecks = checks

				for _, check := range commit.LatestChecks {
					cmds = append(cmds, RunCheck(check))
				}
				m.commitIndex.SetItem(m.commitIndex.Index(), commit)
				m.commitIndex, cmd = m.commitIndex.Update(msg)
				m.commitShow = commitShowModel{commit: commit}
				m.commitShow.Init(m)
				return m, tea.Batch(cmds...)
			}
		case checkResult:
			var commit Commit
			var commitIndex int
			for i, c := range m.commitIndex.Items() {
				if c.(Commit).Id == msg.CommitId {
					commit = c.(Commit)
					commitIndex = i
					break
				}
			}
			updatedChecks := make([]Check, len(commit.LatestChecks))
			for i, c := range commit.LatestChecks {
				if msg.Id == c.Id {
					updatedChecks[i] = Check(msg)
				} else {
					updatedChecks[i] = c
				}
			}
			commit.LatestChecks = updatedChecks
			m.commitIndex.SetItem(commitIndex, commit)
			m.commitIndex, cmd = m.commitIndex.Update(msg)
			m.commitShow = commitShowModel{commit: commit}
			m.commitShow.Init(m)
		}

		m.commitShow.viewport, cmd = m.commitShow.viewport.Update(msg)
		return m, cmd
	}

	return m, cmd
}

type checkResult Check

func RunCheck(check Check) tea.Cmd {
	return func() tea.Msg {
		result, err := executeCheckInWorktree(check.CommitId, check.Command)
		check.Output = result
		if err != nil {
			log.Debugf("Check failed: %v", err)
			check.Status = failed
			return checkResult(check)
		}
		check.Status = succeeded
		return checkResult(check)
	}
}

func executeCheckInWorktree(commitId string, command *exec.Cmd) (string, error) {
	path, cleanup, err := setupWorktree(commitId)
	if err != nil {
		return "", fmt.Errorf("failed to setup worktree: %w", err)
	}
	defer cleanup()

	command.Dir = path
	output, err := runCommandWithOutput(command)
	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

func setupWorktree(commitId string) (string, func(), error) {
	path := fmt.Sprintf("tmp/ci-%s", uuid.NewString())
	worktreeCommand := exec.Command("git", "worktree", "add", "--detach", path, commitId)

	if _, err := worktreeCommand.Output(); err != nil {
		return "", nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	cleanup := func() {
		removeWorktree := exec.Command("git", "worktree", "remove", path)
		if err := removeWorktree.Run(); err != nil {
			log.Debugf("Failed to remove worktree: %v", err)
		}
	}

	return path, cleanup, nil
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
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
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
		IssueDetailFocus: key.NewBinding(
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
		CommitDetailFocus: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "more info"),
		),
	}

	switch {
	case matchRoute(m.path, issuesIndexPath):
	case strings.HasSuffix(m.path, "/show"):
		keys.Up = key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		)

		keys.Down = key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		)

		// Disable with this:
		// keys.IssueCommentFormFocus.SetEnabled(false)
	}

	keys.Path = m.path

	return keys
}

func boxStyle(size bl.Size) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(size.Width - 2).
		Height(size.Height - 2)

	return style
}

func (m Model) View() string {
	if !m.loaded {
		return "Loading..."
	}

	doc := strings.Builder{}

	var renderedTabs []string

	var view string

	help := helpStyle.Render(m.help.View(m.HelpKeys()))

	switch {
	case strings.HasPrefix(m.path, "/issues"):
		var sidebarView string

		for _, t := range m.tabs {
			var style lipgloss.Style
			isActive := t == "Issues"
			if isActive {
				style = activeTabStyle
			} else {
				style = inactiveTabStyle
			}
			renderedTabs = append(renderedTabs, style.Render(t))
		}

		switch {
		case matchRoute(m.path, issuesIndexPath):

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesShowPath):
			sidebarView = lipgloss.NewStyle().
				Render(m.issueShow.View())

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesCommentContentPath):
			sidebarView = lipgloss.NewStyle().
				Render(lipgloss.JoinVertical(
					lipgloss.Left,
					m.issueShow.View(),
					m.issueShow.commentForm.View("content"),
				))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesCommentConfirmationPath):
			sidebarView = lipgloss.NewStyle().
				Render(lipgloss.JoinVertical(
					lipgloss.Left,
					m.issueShow.View(),
					m.issueShow.commentForm.View("confirmation"),
				))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesEditTitlePath):
			style := lipgloss.NewStyle()

			sidebarView = style.
				Render(m.issueForm.View("title", true))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesEditDescriptionPath):
			style := lipgloss.NewStyle()

			sidebarView = style.
				Render(m.issueForm.View("description", true))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesEditConfirmationPath):
			style := lipgloss.NewStyle()

			sidebarView = style.
				Render(m.issueForm.View("confirmation", true))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesNewTitlePath):
			style := lipgloss.NewStyle()

			sidebarView = style.
				Render(m.issueForm.View("title", false))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesNewDescriptionPath):
			style := lipgloss.NewStyle()

			sidebarView = style.
				Render(m.issueForm.View("description", false))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesNewConfirmationPath):
			style := lipgloss.NewStyle()

			sidebarView = style.
				Render(m.issueForm.View("confirmation", false))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueIndex.View()),
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		}
	case strings.HasPrefix(m.path, "/checks"):
		for _, t := range m.tabs {
			var style lipgloss.Style
			isActive := t == "Checks"
			if isActive {
				style = activeTabStyle
			} else {
				style = inactiveTabStyle
			}
			renderedTabs = append(renderedTabs, style.Render(t))
		}
		commitListView := lipgloss.NewStyle().
			Render(m.commitIndex.View())

		switch {
		case matchRoute(m.path, checksIndexPath):
			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				boxStyle(m.LeftSize).Render(commitListView),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, checksShowPath):
			commitDetailView := lipgloss.NewStyle().
				Render(m.commitShow.View())
			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					commitListView,
					boxStyle(m.RightSize).Border(lipgloss.NormalBorder(), true).Render(commitDetailView),
				),
				help,
			)
		}
	}

	doc.WriteString(view)
	return docStyle.Render(doc.String())
}

type Commit struct {
	Id            string    `json:"id"`
	AbbreviatedId string    `json:"abbreviatedId"`
	Author        string    `json:"author"`
	Description   string    `json:"description"`
	Timestamp     time.Time `json:"timestamp"`
	LatestChecks  []Check   `json:"latestCheck"`
}

type CheckStatus string

const (
	failed    CheckStatus = "failed"
	succeeded CheckStatus = "succeeded"
	running   CheckStatus = "running"
)

type Check struct {
	Command    *exec.Cmd
	Id         string      `json:"id"`
	CommitId   string      `json:"commitId"`
	Status     CheckStatus `json:"status"`
	Checker    string      `json:"checker"`
	Name       string      `json:"name"`
	Output     string      `json:"output"`
	StartedAt  time.Time   `json:"startedAt"`
	FinishedAt time.Time   `json:"finishedAt"`
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
		aggregateStatus := running
		anyStillRunning := false
		failing := false
		for _, check := range c.LatestChecks {
			switch check.Status {
			case running:
				anyStillRunning = true
			case failed:
				failing = true
			}
		}

		if !anyStillRunning {
			if failing {
				aggregateStatus = failed
			} else {
				aggregateStatus = succeeded
			}
		}

		if aggregateStatus == running {
			title = fmt.Sprintf("%s %s", title, lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("[⋯]"))
		}
		if aggregateStatus == failed {
			title = fmt.Sprintf("%s %s", title, lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("[×]"))
		}
		if aggregateStatus == succeeded {
			title = fmt.Sprintf("%s %s", title, lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("[✓]"))
		}
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
		cmd := exec.Command("git", "cat-file", "-p", refHash)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			continue
		}

		var check Check
		json.Unmarshal(out.Bytes(), &check)

		for _, commit := range commits {
			if check.CommitId == commit.Id {
				commit.LatestChecks = append(commit.LatestChecks, check)
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
		cmd := exec.Command("git", "cat-file", "-p", refHash)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			continue
		}

		var issue Issue
		json.Unmarshal(out.Bytes(), &issue)

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

		if issue.Status == done {
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

type commitShowModel struct {
	commit   Commit
	viewport viewport.Model
}

func (m *commitShowModel) Init(ctx Model) tea.Cmd {
	var s strings.Builder
	var status string
	var aggregateStatus CheckStatus

	for _, check := range m.commit.LatestChecks {
		aggregateStatus = check.Status
		if aggregateStatus == failed || aggregateStatus == running {
			break
		}
	}
	switch aggregateStatus {
	case running:
		status = lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("[⋯]")
	case failed:
		status = lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("[×]")
	case succeeded:
		status = lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("[✓]")
	}
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf("#%s", m.commit.AbbreviatedId))
	header := fmt.Sprintf("%s %s\nStatus: %s\n\n", identifier, m.commit.Description, status)
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString("\n")

	m.viewport = viewport.New(ctx.Layout.RightSize.Width-2, ctx.Layout.RightSize.Height-2)
	s.WriteString(m.commit.Description)
	for _, check := range m.commit.LatestChecks {
		s.WriteString(fmt.Sprintf("\n\n%s", check.Output))
		s.WriteString(fmt.Sprintf("\n\n%s", check.Status))
	}

	m.viewport.SetContent(s.String())
	return nil
}

func (m commitShowModel) Update(msg tea.Msg) (commitShowModel, tea.Cmd) {
	return m, nil
}

func (m commitShowModel) View() string {
	var s strings.Builder
	s.WriteString(m.viewport.View())
	return s.String()
}

type issueShowModel struct {
	layout      Layout
	issue       Issue
	viewport    viewport.Model
	commentForm commentFormModel
}

func (m *issueShowModel) Init(ctx Model) tea.Cmd {
	m.viewport = viewport.New(
		ctx.Layout.RightSize.Width-2,
		ctx.Layout.RightSize.Height-len(strings.Split(m.commentForm.View("content"), "\n"))-3,
	)
	m.layout = ctx.Layout
	var s strings.Builder
	var status string
	switch m.issue.Status {
	case todo:
		status = "todo"
	case inProgress:
		status = lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("in-progress")
	case wontDo:
		status = lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("wont-do")
	case done:
		status = lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("done")
	}
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("#%s", m.issue.Shortcode))
	header := fmt.Sprintf("%s %s\nStatus: %s\n\n", identifier, m.issue.Title, status)
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString(m.issue.Description + "\n")

	commentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.Theme.SecondaryBorder).
		Width(m.viewport.Width - 2).
		MarginTop(1)

	commentHeaderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(styles.Theme.SecondaryBorder).
		Width(m.viewport.Width - 2)

	for i, comment := range m.issue.Comments {
		commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.Author, comment.CreatedAt.Format(time.RFC822)))
		if i == len(m.issue.Comments)-1 { // last comment
			s.WriteString(commentStyle.MarginBottom(6).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.Content)))
		} else {
			s.WriteString(commentStyle.Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.Content)))
		}
	}
	m.viewport.SetContent(s.String())
	return nil
}

func (m issueShowModel) Update(msg tea.Msg) (issueShowModel, tea.Cmd) {
	return m, nil
}

func (m issueShowModel) View() string {
	return m.viewport.View()
}

type issueFormModel struct {
	titleInput       textinput.Model
	descriptionInput textarea.Model
	identifier       string
	editing          bool
}

func (m issueFormModel) Submit() tea.Msg {
	return m
}

func (m *issueFormModel) Init(title, description string) tea.Cmd {
	m.SetTitle(title)
	m.SetDescription(description)
	return m.titleInput.Focus()
}

func (m issueFormModel) Update(msg tea.Msg) (issueFormModel, tea.Cmd) {
	return m, nil
}

func (m issueFormModel) View(focus string, editing bool) string {
	var s strings.Builder

	identifier := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("#%s", m.identifier))

	if editing {
		s.WriteString(fmt.Sprintf("Editing issue %s\n\n", identifier))
	} else {
		s.WriteString("New issue\n\n")
	}

	s.WriteString(m.titleInput.View())
	s.WriteString("\n")
	s.WriteString(m.descriptionInput.View())
	s.WriteString("\n")
	var style lipgloss.Style
	if focus == "confirmation" {
		style = lipgloss.NewStyle().Foreground(styles.Theme.PrimaryText).Background(styles.Theme.SelectedBackground)
	} else {
		style = lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText)
	}

	s.WriteString(style.Render("Save"))

	return s.String()
}

func (m *issueFormModel) SetTitle(title string) {
	m.titleInput = textinput.New()
	m.titleInput.CharLimit = 120
	m.titleInput.Width = 80
	m.titleInput.SetValue(title)
}

func (m *issueFormModel) SetDescription(description string) {
	m.descriptionInput = textarea.New()
	m.descriptionInput.CharLimit = 0 // unlimited
	m.descriptionInput.MaxHeight = 0 // unlimited
	m.descriptionInput.ShowLineNumbers = false
	m.descriptionInput.SetHeight(30)
	m.descriptionInput.SetWidth(80)
	m.descriptionInput.SetValue(description)
}

type commentFormModel struct {
	contentInput textarea.Model
}

func NewCommentFormModel() commentFormModel {
	t := textarea.New()
	t.ShowLineNumbers = false
	t.Prompt = "┃"
	t.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("transparent"))
	t.SetCursor(0)
	t.SetWidth(75)
	t.Focus()

	return commentFormModel{
		contentInput: t,
	}
}

func (m commentFormModel) Submit() tea.Msg {
	return m
}

func (m commentFormModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m commentFormModel) Update(msg tea.Msg) (commentFormModel, tea.Cmd) {
	return m, nil
}

func (m commentFormModel) View(focus string) string {
	var s strings.Builder
	s.WriteString(m.contentInput.View())
	s.WriteString("\n")
	if focus == "content" {
		s.WriteString(lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render("Save"))
	} else {
		s.WriteString(lipgloss.NewStyle().Foreground(styles.Theme.PrimaryText).Background(styles.Theme.SelectedBackground).Render("Save"))
	}
	return s.String()
}

func main() {
	_ = lipgloss.HasDarkBackground()
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen(), tea.WithReportFocus())

	var logFile *os.File
	if isDebugEnabled() {
		var err error
		logFile, err = setupLogging()
		if err != nil {
			fmt.Printf("Error setting up logging: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()
	}

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
