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

type issuePersistedMsg struct {
	Issue    Issue
	NewIssue bool
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
			// shortcodeCache := make(map[string]bool)
			shortcode := StringToShortcode(id)
			issue.Id = id
			issue.Shortcode = shortcode
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
			Issue:    issue,
			NewIssue: newIssue,
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
	title := fmt.Sprintf("%s %s", status, titleFn(i.Shortcode, truncate(i.Title, 50)))

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
	loaded       bool
	path         string
	issueList    list.Model
	issueDetail  issueDetailModel
	issueForm    issueFormModel
	commitList   list.Model
	commitDetail commitDetailModel
	err          error
	help         help.Model
	styles       Styles
	tabs         []string
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
		path:       issuesIndexPath,
		help:       helpModel,
		styles:     DefaultStyles(),
		tabs:       []string{"Issues", "Checks"},
		Layout:     layout,
		issueList:  issueList,
		commitList: commitList,
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
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.loaded = true
		}

		return m, func() tea.Msg {
			return m.Layout.Resize(msg.Width, msg.Height)
		}
	case IssuesReadyMsg:
		var listItems []list.Item
		for _, issue := range msg {
			listItems = append(listItems, issue)
		}
		m.issueList.SetItems(listItems)
	case CommitListReadyMsg:
		var listItems []list.Item
		for _, commit := range msg {
			listItems = append(listItems, commit)
		}
		m.commitList.SetItems(listItems)
	case bl.BubbleLayoutMsg:
		m.LeftSize, _ = msg.Size(m.LeftID)
		m.RightSize, _ = msg.Size(m.RightID)
		m.HeaderSize, _ = msg.Size(m.HeaderID)
		m.FooterSize, _ = msg.Size(m.FooterID)

		m.issueList.SetSize(m.LeftSize.Width, m.LeftSize.Height)
		m.commitList.SetSize(m.LeftSize.Width, m.LeftSize.Height)
	case commentFormModel:
		currentIndex := m.issueList.Index()
		currentIssue := m.issueList.SelectedItem().(Issue)
		currentIssue.Comments = append(currentIssue.Comments, Comment{Author: "garrett@blvrd.co", Content: msg.contentInput.Value()})
		m.issueList.SetItem(currentIndex, currentIssue)
		m.issueDetail = issueDetailModel{issue: currentIssue}
		m.issueDetail.commentForm = NewCommentFormModel()
		m.issueDetail.Init(m)
		m.issueDetail.viewport.GotoBottom()

		m.path = issuesShowPath
	case issueFormModel:
		if msg.editing {
			currentIssue := m.issueList.SelectedItem().(Issue)
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
		if msg.NewIssue {
			m.issueList.InsertItem(0, msg.Issue)
			m.issueList.Select(0)
			m.issueDetail = issueDetailModel{issue: msg.Issue}
			m.issueDetail.commentForm = NewCommentFormModel()
			m.issueDetail.Init(m)
			m.path = issuesShowPath
		} else if !msg.Issue.DeletedAt.IsZero() {
			currentIndex := m.issueList.Index()
			m.issueList.RemoveItem(currentIndex)
			m.path = issuesIndexPath
		} else {
			currentIndex := m.issueList.Index()
			m.issueDetail = issueDetailModel{issue: msg.Issue}
			m.issueDetail.commentForm = NewCommentFormModel()
			m.issueDetail.Init(m)
			m.issueList.SetItem(currentIndex, msg.Issue)
			m.path = issuesShowPath
		}
		return m, nil
	}

	switch {
	case matchRoute(m.path, issuesIndexPath):
		if m.issueList.SettingFilter() {
			m.issueList, cmd = m.issueList.Update(msg)
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
				currentIndex := m.issueList.Index()
				currentIssue := m.issueList.SelectedItem().(Issue)
				if currentIssue.Status == todo {
					currentIssue.Status = done
				} else {
					currentIssue.Status = todo
				}
				cmd = m.issueList.SetItem(currentIndex, currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueStatusWontDo):
				currentIndex := m.issueList.Index()
				currentIssue := m.issueList.SelectedItem().(Issue)
				if currentIssue.Status == todo {
					currentIssue.Status = wontDo
				} else {
					currentIssue.Status = todo
				}
				cmd = m.issueList.SetItem(currentIndex, currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueStatusInProgress):
				currentIndex := m.issueList.Index()
				currentIssue := m.issueList.SelectedItem().(Issue)
				if currentIssue.Status == todo {
					currentIssue.Status = inProgress
				} else {
					currentIssue.Status = todo
				}
				cmd = m.issueList.SetItem(currentIndex, currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueCommentFormFocus):
				m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.commentForm.Init()
				m.issueDetail.Init(m)
				m.issueDetail.viewport.GotoBottom()
				m.path = issuesCommentContentPath
			case key.Matches(msg, keys.IssueDetailFocus):
				m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.commentForm.Init()
				m.issueDetail.Init(m)
				m.path = issuesShowPath
			case key.Matches(msg, keys.IssueNewForm):
				m.issueForm = issueFormModel{editing: false}
				m.issueForm.Init("", "")
				cmd = m.issueForm.titleInput.Focus()
				m.path = issuesEditTitlePath
			case key.Matches(msg, keys.IssueDelete):
				issue := m.issueList.SelectedItem().(Issue)
				issue.DeletedAt = time.Now().UTC()
				cmd = persistIssue(issue)
			case key.Matches(msg, keys.NextPage):
				m.path = checksIndexPath
			case key.Matches(msg, keys.PrevPage):
				return m, nil
			}
		}

		m.issueList, cmd = m.issueList.Update(msg)
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
				currentIndex := m.issueList.Index()
				currentIssue := m.issueList.SelectedItem().(Issue)
				if currentIssue.Status == todo {
					currentIssue.Status = done
				} else {
					currentIssue.Status = todo
				}
				m.issueDetail = issueDetailModel{issue: currentIssue}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.commentForm.Init()
				m.issueDetail.Init(m)
				cmd = m.issueList.SetItem(currentIndex, currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueStatusWontDo):
				currentIndex := m.issueList.Index()
				currentIssue := m.issueList.SelectedItem().(Issue)
				if currentIssue.Status == todo {
					currentIssue.Status = wontDo
				} else {
					currentIssue.Status = todo
				}
				m.issueDetail = issueDetailModel{issue: currentIssue}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.commentForm.Init()
				m.issueDetail.Init(m)
				cmd = m.issueList.SetItem(currentIndex, currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueStatusInProgress):
				currentIndex := m.issueList.Index()
				currentIssue := m.issueList.SelectedItem().(Issue)
				if currentIssue.Status == todo {
					currentIssue.Status = inProgress
				} else {
					currentIssue.Status = todo
				}
				m.issueDetail = issueDetailModel{issue: currentIssue}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.commentForm.Init()
				m.issueDetail.Init(m)
				cmd = m.issueList.SetItem(currentIndex, currentIssue)
				return m, cmd
			case key.Matches(msg, keys.IssueEditForm):
				selectedIssue := m.issueList.SelectedItem().(Issue)
				m.issueForm = issueFormModel{editing: true, identifier: selectedIssue.Shortcode}
				cmd = m.issueForm.Init(selectedIssue.Title, selectedIssue.Description)

				m.path = issuesEditTitlePath
				return m, cmd
			case key.Matches(msg, keys.Back):
				m.path = issuesIndexPath
			case key.Matches(msg, keys.IssueCommentFormFocus):
				m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.commentForm.Init()
				m.issueDetail.Init(m)
				m.issueDetail.viewport.GotoBottom()
				m.path = issuesCommentContentPath
			}
		}

		m.issueDetail.viewport, cmd = m.issueDetail.viewport.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesCommentContentPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				currentIssue := m.issueList.SelectedItem().(Issue)
				m.issueDetail = issueDetailModel{issue: currentIssue}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.Init(m)
				m.path = issuesShowPath
			case key.Matches(msg, keys.NextInput):
				m.issueDetail.commentForm.contentInput.Blur()
				m.path = issuesCommentConfirmationPath
			}
		}
		m.issueDetail.commentForm.contentInput, cmd = m.issueDetail.commentForm.contentInput.Update(msg)
		return m, cmd
	case matchRoute(m.path, issuesCommentConfirmationPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				currentIssue := m.issueList.SelectedItem().(Issue)
				m.issueDetail = issueDetailModel{issue: currentIssue}
				m.issueDetail.commentForm = NewCommentFormModel()
				m.issueDetail.Init(m)
				m.path = issuesShowPath
			case key.Matches(msg, keys.Submit):
				return m, m.issueDetail.commentForm.Submit
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
	case matchRoute(m.path, checksIndexPath):
		if m.commitList.SettingFilter() {
			m.commitList, cmd = m.commitList.Update(msg)
			return m, cmd
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Down):
				m.commitList, cmd = m.commitList.Update(msg)
				return m, cmd
			case key.Matches(msg, keys.Up):
				m.commitList, cmd = m.commitList.Update(msg)
				return m, cmd
			case key.Matches(msg, keys.RunCheck):
				commit := m.commitList.SelectedItem().(Commit)
				commit.latestCheck = Check{status: "running"}
				m.commitList.SetItem(m.commitList.Index(), commit)
				m.commitList, cmd = m.commitList.Update(msg)
				m.commitDetail = commitDetailModel{commit: commit}
				m.commitDetail.Init(m)
				cmd = RunCheck(commit.id)
				return m, cmd
			case key.Matches(msg, keys.CommitDetailFocus):
				m.commitDetail = commitDetailModel{commit: m.commitList.SelectedItem().(Commit)}
				m.commitDetail.Init(m)
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
		case checkResult:
			var commit Commit
			var commitIndex int
			for i, c := range m.commitList.Items() {
				if c.(Commit).id == msg.commitHash {
					commit = c.(Commit)
					commitIndex = i
					break
				}
			}
			commit.latestCheck = Check{status: msg.status, output: msg.output}
			m.commitList.SetItem(commitIndex, commit)
			m.commitList, cmd = m.commitList.Update(msg)
			m.commitDetail = commitDetailModel{commit: commit}
			m.commitDetail.Init(m)
		}

		m.commitList, cmd = m.commitList.Update(msg)
		return m, cmd
	case matchRoute(m.path, checksShowPath):
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Back):
				m.path = checksIndexPath
			case key.Matches(msg, keys.RunCheck):
				commit := m.commitList.SelectedItem().(Commit)
				commit.latestCheck = Check{status: "running"}
				m.commitList.SetItem(m.commitList.Index(), commit)
				m.commitList, cmd = m.commitList.Update(msg)
				m.commitDetail = commitDetailModel{commit: commit}
				m.commitDetail.Init(m)
				cmd = RunCheck(commit.id)
				return m, cmd
			}
		case checkResult:
			var commit Commit
			var commitIndex int
			for i, c := range m.commitList.Items() {
				if c.(Commit).id == msg.commitHash {
					commit = c.(Commit)
					commitIndex = i
					break
				}
			}
			commit.latestCheck = Check{status: msg.status, output: msg.output}
			m.commitList.SetItem(commitIndex, commit)
			m.commitList, cmd = m.commitList.Update(msg)
			m.commitDetail = commitDetailModel{commit: commit}
			m.commitDetail.Init(m)
		}

		m.commitDetail.viewport, cmd = m.commitDetail.viewport.Update(msg)
		return m, cmd
	}

	return m, cmd
}

type checkResult struct {
	status     string
	commitHash string
	output     string
}

func RunCheck(commitId string) tea.Cmd {
	return func() tea.Msg {
		randomString := uuid.NewString()
		path := "tmp/ci-" + randomString
		command := exec.Command("git", "worktree", "add", "--detach", path, commitId)
		removeWorktree := exec.Command("git", "worktree", "remove", path)
		defer removeWorktree.Run()

		_, err := command.Output()
		if err != nil {
			log.Fatal(err)
		}
		command = exec.Command("go", "test")
		command.Dir = path
		output, err := command.Output()
		if err != nil {
			log.Debugf("%#v", err)
			return checkResult{commitHash: commitId, status: "failed", output: string(output)}
		}
		return checkResult{commitHash: commitId, status: "succeeded", output: string(output)}
	}
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
		Width(size.Width).
		Height(size.Height)

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
					boxStyle(m.LeftSize).Render(m.issueList.View()),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesShowPath):
			sidebarView = lipgloss.NewStyle().
				Render(m.issueDetail.View())

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueList.View()),
					boxStyle(m.RightSize).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesCommentContentPath):
			sidebarView = lipgloss.NewStyle().
				Render(lipgloss.JoinVertical(
					lipgloss.Left,
					m.issueDetail.View(),
					m.issueDetail.commentForm.View("content"),
				))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueList.View()),
					boxStyle(m.RightSize).Render(sidebarView),
				),
				boxStyle(m.FooterSize).Render(help),
			)
		case matchRoute(m.path, issuesCommentConfirmationPath):
			sidebarView = lipgloss.NewStyle().
				Render(lipgloss.JoinVertical(
					lipgloss.Left,
					m.issueDetail.View(),
					m.issueDetail.commentForm.View("confirmation"),
				))

			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					boxStyle(m.LeftSize).Render(m.issueList.View()),
					boxStyle(m.RightSize).Render(sidebarView),
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
					boxStyle(m.LeftSize).Render(m.issueList.View()),
					boxStyle(m.RightSize).Render(sidebarView),
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
					boxStyle(m.LeftSize).Render(m.issueList.View()),
					boxStyle(m.RightSize).Render(sidebarView),
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
					boxStyle(m.LeftSize).Render(m.issueList.View()),
					boxStyle(m.RightSize).Render(sidebarView),
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
			Render(m.commitList.View())

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
				Render(m.commitDetail.View())
			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				lipgloss.JoinHorizontal(lipgloss.Top, commitListView, commitDetailView),
				help,
			)
		}
	}

	doc.WriteString(view)
	return docStyle.Render(doc.String())
}

type Commit struct {
	id            string
	abbreviatedId string
	author        string
	description   string
	timestamp     time.Time
	latestCheck   Check
}

type Check struct {
	id         string
	commitId   string
	status     string
	checker    string
	output     string
	startedAt  time.Time
	finishedAt time.Time
}

func (c Commit) FilterValue() string {
	return c.id
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

	if c.author == "" {
		author = "unknown"
	} else {
		author = c.author
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

	title := fmt.Sprintf("%s", titleFn(c.abbreviatedId, truncate(c.description, 50)))
	if c.latestCheck.status == "running" {
		title = fmt.Sprintf("%s %s", title, lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("[⋯]"))
	}
	if c.latestCheck.status == "failed" {
		title = fmt.Sprintf("%s %s", title, lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("[×]"))
	}
	if c.latestCheck.status == "succeeded" {
		title = fmt.Sprintf("%s %s", title, lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("[✓]"))
	}

	description := fmt.Sprintf("committed at %s by %s", c.timestamp.Format(time.RFC822), author)
	item := lipgloss.JoinVertical(lipgloss.Left, title, description)

	fmt.Fprintf(w, item)
}

type CommitListReadyMsg []Commit

func getCommits() tea.Msg {
	var commits []Commit

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
		commits = append(commits, Commit{
			id:            parts[0],
			abbreviatedId: parts[1],
			author:        parts[2],
			timestamp:     timestamp,
			description:   parts[4],
		})
	}

	return CommitListReadyMsg(commits)
}

type IssuesReadyMsg []Issue

func getIssues() tea.Msg {
	issues := seedIssues

	cmd := exec.Command("git", "for-each-ref", "--format=%(objectname)", "refs/ubik")
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

		issues = append(issues, issue)
	}

	return IssuesReadyMsg(issues)
}

type commitDetailModel struct {
	commit   Commit
	viewport viewport.Model
}

func (m *commitDetailModel) Init(ctx Model) tea.Cmd {
	var s strings.Builder
	var status string
	switch m.commit.latestCheck.status {
	case "running":
		status = lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("[⋯]")
	case "failed":
		status = lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("[×]")
	case "succeeded":
		status = lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("[✓]")
	}
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.FaintText).Render(fmt.Sprintf("#%s", m.commit.abbreviatedId))
	header := fmt.Sprintf("%s %s\nStatus: %s", identifier, m.commit.description, status)
	s.WriteString(lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).PaddingTop(0).Render(header))
	s.WriteString("\n")

	m.viewport = viewport.New(ctx.Layout.RightSize.Width, ctx.Layout.RightSize.Height)
	s.WriteString(m.commit.description)
	s.WriteString(fmt.Sprintf("\n\n%s", m.commit.latestCheck.output))

	m.viewport.SetContent(s.String())
	return nil
}

func (m commitDetailModel) Update(msg tea.Msg) (commitDetailModel, tea.Cmd) {
	return m, nil
}

func (m commitDetailModel) View() string {
	var s strings.Builder
	s.WriteString(m.viewport.View())
	return s.String()
}

type issueDetailModel struct {
	layout      Layout
	issue       Issue
	viewport    viewport.Model
	commentForm commentFormModel
}

func (m *issueDetailModel) Init(ctx Model) tea.Cmd {
	m.viewport = viewport.New(
		ctx.Layout.RightSize.Width,
		ctx.Layout.RightSize.Height-len(strings.Split(m.commentForm.View("content"), "\n")),
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
	header := fmt.Sprintf("%s %s\nStatus: %s", identifier, m.issue.Title, status)
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
		commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.Author, comment.CreatedAt))
		if i == len(m.issue.Comments)-1 { // last comment
			s.WriteString(commentStyle.MarginBottom(6).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.Content)))
		} else {
			s.WriteString(commentStyle.Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.Content)))
		}
	}
	m.viewport.SetContent(s.String())
	return nil
}

func (m issueDetailModel) Update(msg tea.Msg) (issueDetailModel, tea.Cmd) {
	return m, nil
}

func (m issueDetailModel) View() string {
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
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gomnd
	if err != nil {
		fmt.Printf("error opening file for logging: %s", err)
		os.Exit(1)
	}
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	if err != nil {
		log.Print("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	_, err = p.Run()
	if err != nil {
		log.Debug(err)
		os.Exit(1)
	}
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
