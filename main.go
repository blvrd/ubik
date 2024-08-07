package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	// "github.com/charmbracelet/bubbles/help"
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

type pathChangedMsg string

func NavigateTo(path string) tea.Cmd {
	return func() tea.Msg {
		return pathChangedMsg(path)
	}
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
	case strings.HasPrefix(k.Path, "/issues/index"):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.IssueNewForm, k.IssueDetailFocus},
			{k.IssueStatusDone, k.IssueStatusWontDo},
			{k.IssueStatusInProgress, k.IssueCommentFormFocus},
		}
	case strings.HasPrefix(k.Path, "/issues/show"):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.IssueEditForm, k.Back},
			{k.IssueNewForm, k.IssueDetailFocus},
			{k.IssueStatusDone, k.IssueStatusWontDo},
			{k.IssueStatusInProgress, k.IssueCommentFormFocus},
		}
	case strings.HasPrefix(k.Path, "/issues/edit"):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.NextInput, k.Back},
		}
	case strings.HasPrefix(k.Path, "/commits/index"):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.CommitDetailFocus},
		}
	case strings.HasPrefix(k.Path, "/commits/show"):
		bindings = [][]key.Binding{
			{k.Help, k.Quit},
			{k.Up, k.Down},
			{k.Back},
		}
	}

	return bindings
}

type Issue struct {
	id          string
	shortcode   string
	author      string
	title       string
	description string
	status      status
	comments    []Comment
	createdAt   time.Time
	updatedAt   time.Time
}

func (i Issue) FilterValue() string {
	return i.title
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

	switch i.status {
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
	title := fmt.Sprintf("%s %s", status, titleFn(i.shortcode, i.title))

	description := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("created by %s at %s", i.author, i.createdAt.Format(time.RFC822)))
	item := lipgloss.JoinVertical(lipgloss.Left, title, description)

	fmt.Fprintf(w, item)
}

type Comment struct {
	author    string
	content   string
	createdAt time.Time
	updatedAt time.Time
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
		path:       "/issues/index",
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
	componentUpdateMsg := updateMsg{originalMsg: msg, keys: keys}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.loaded = true
		}

		return m, func() tea.Msg {
			return m.Layout.Resize(msg.Width, msg.Height)
		}
	case pathChangedMsg:
		m.path = string(msg)
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
		currentIssue.comments = append(currentIssue.comments, Comment{author: "garrett@blvrd.co", content: msg.contentInput.Value()})
		m.issueList.SetItem(currentIndex, currentIssue)
		m.issueDetail = issueDetailModel{issue: currentIssue}
		m.issueDetail.commentForm = NewCommentFormModel()
		m.issueDetail.Init(m)
		m.issueDetail.viewport.GotoBottom()

		m.path = "/issues/show"
		return m, NavigateTo("/issues/show")
	case issueFormModel:
		if msg.editing {
			currentIndex := m.issueList.Index()
			currentIssue := m.issueList.SelectedItem().(Issue)
			currentIssue.title = msg.titleInput.Value()
			currentIssue.description = msg.descriptionInput.Value()
			m.issueDetail = issueDetailModel{issue: currentIssue}
			m.issueDetail.commentForm = NewCommentFormModel()
			m.issueDetail.Init(m)
			m.issueList.SetItem(currentIndex, currentIssue)
		} else {
			id := uuid.NewString()
			newIssue := Issue{
				id:          id,
				shortcode:   StringToShortcode(id),
				title:       msg.titleInput.Value(),
				description: msg.descriptionInput.Value(),
				status:      todo,
				author:      "garrett@blvrd.co",
			}
			m.issueList.InsertItem(0, newIssue)
			m.issueList.Select(0)
			m.issueDetail = issueDetailModel{issue: newIssue}
			m.issueDetail.commentForm = NewCommentFormModel()
			m.issueDetail.Init(m)
		}

		m.path = "/issues/show"
		return m, NavigateTo("/issues/show")
	}

	// switch path {
	// case "/issues":
	// case "/issues/:id":
	// case "/issues/:id/edit/title":
	// // route keystrokes to issue form model's title field
	//   // MSG = TAB KEY
	// // WRAP IT (TAB KEY, extra info for the specific input)
	// case "/issues/:id/edit/description":
	// // route keystrokes to issue form model's description field
	// case "/issues/:id/edit/confirm":
	// // route keystrokes to issue form model's confirmation field
	// }

	switch {
	case strings.HasPrefix(m.path, "/issues"):
		switch {
		case strings.HasSuffix(m.path, "/index"):
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
					if currentIssue.status == todo {
						currentIssue.status = done
					} else {
						currentIssue.status = todo
					}
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case key.Matches(msg, keys.IssueStatusWontDo):
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = wontDo
					} else {
						currentIssue.status = todo
					}
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case key.Matches(msg, keys.IssueStatusInProgress):
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = inProgress
					} else {
						currentIssue.status = todo
					}
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case key.Matches(msg, keys.IssueCommentFormFocus):
					m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
					m.issueDetail.commentForm = NewCommentFormModel()
					m.issueDetail.commentForm.Init()
					m.issueDetail.Init(m)
					m.issueDetail.viewport.GotoBottom()
					m.path = "/issues/show/comments/new/content"
					return m, NavigateTo("/issues/show/comments/new/content")
				case key.Matches(msg, keys.IssueDetailFocus):
					m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
					m.issueDetail.commentForm = NewCommentFormModel()
					m.issueDetail.commentForm.Init()
					m.issueDetail.Init(m)
					m.path = "/issues/show"
					return m, NavigateTo("/issues/show")
				case key.Matches(msg, keys.IssueNewForm):
					m.issueForm = issueFormModel{editing: false}
					m.issueForm.Init("", "")
					cmd = m.issueForm.titleInput.Focus()
				case key.Matches(msg, keys.NextPage):
					m.path = "/checks/index"
					return m, NavigateTo("/checks/index")
				case key.Matches(msg, keys.PrevPage):
					return m, nil
				}
			}

			m.issueList, cmd = m.issueList.Update(msg)
		case strings.HasSuffix(m.path, "/show"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Help):
					if m.issueDetail.focus == issueDetailCommentFocused {
						break
					}

					if m.help.ShowAll {
						m.help.ShowAll = false
					} else {
						m.help.ShowAll = true
					}
					return m, nil
				case key.Matches(msg, keys.IssueStatusDone):
					if m.issueDetail.focus == issueDetailCommentFocused {
						break
					}
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = done
					} else {
						currentIssue.status = todo
					}
					m.issueDetail = issueDetailModel{issue: currentIssue}
					m.issueDetail.Init(m)
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case key.Matches(msg, keys.IssueStatusWontDo):
					if m.issueDetail.focus == issueDetailCommentFocused {
						break
					}
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = wontDo
					} else {
						currentIssue.status = todo
					}
					m.issueDetail = issueDetailModel{issue: currentIssue}
					m.issueDetail.Init(m)
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case key.Matches(msg, keys.IssueStatusInProgress):
					if m.issueDetail.focus == issueDetailCommentFocused {
						break
					}
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = inProgress
					} else {
						currentIssue.status = todo
					}
					m.issueDetail = issueDetailModel{issue: currentIssue}
					m.issueDetail.Init(m)
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case key.Matches(msg, keys.IssueEditForm):
					if m.issueDetail.focus == issueDetailCommentFocused {
						m.issueDetail, cmd = m.issueDetail.Update(componentUpdateMsg)
					} else {
						selectedIssue := m.issueList.SelectedItem().(Issue)
						m.issueForm = issueFormModel{editing: true, identifier: selectedIssue.shortcode}
						cmd = m.issueForm.Init(selectedIssue.title, selectedIssue.description)
						// m.issueForm.SetTitle(selectedIssue.title)
						// m.issueForm.SetDescription(selectedIssue.description)
						// m.issueForm.titleInput.Focus()
					}

					m.path = "/issues/edit/title"
					return m, cmd
				case key.Matches(msg, keys.Back):
					if m.issueDetail.focus == issueDetailViewportFocused {
						m.path = "/issues/index"
						return m, NavigateTo("/issues/index")
					} else {
						m.issueDetail, cmd = m.issueDetail.Update(componentUpdateMsg)
						return m, cmd
					}
				case key.Matches(msg, keys.IssueCommentFormFocus):
					m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
					m.issueDetail.commentForm = NewCommentFormModel()
					m.issueDetail.commentForm.Init()
					m.issueDetail.Init(m)
					m.issueDetail.viewport.GotoBottom()
					m.path = "/issues/show/comments/new/content"
					return m, NavigateTo("/issues/show/comments/new/content")
				}
			}

			m.issueDetail, cmd = m.issueDetail.Update(componentUpdateMsg)
		case strings.HasSuffix(m.path, "/show/comments/new/content"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Back):
					currentIssue := m.issueList.SelectedItem().(Issue)
					m.issueDetail = issueDetailModel{issue: currentIssue}
					m.issueDetail.commentForm = NewCommentFormModel()
					m.issueDetail.Init(m)
					m.path = "/issues/show"
					return m, NavigateTo("/issues/show")
				case key.Matches(msg, keys.NextInput):
					m.issueDetail.commentForm.contentInput.Blur()
					m.issueDetail.commentForm.focusState = commentConfirmationFocused
					m.path = "/issues/show/comments/new/confirmation"
					return m, NavigateTo("/issues/show/comments/new/confirmation")
				}
			}
			m.issueDetail.commentForm.contentInput, cmd = m.issueDetail.commentForm.contentInput.Update(componentUpdateMsg.originalMsg)
			return m, cmd
		case strings.HasSuffix(m.path, "/show/comments/new/confirmation"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Back):
					currentIssue := m.issueList.SelectedItem().(Issue)
					m.issueDetail = issueDetailModel{issue: currentIssue}
					m.issueDetail.commentForm = NewCommentFormModel()
					m.issueDetail.Init(m)
					m.path = "/issues/show"
					return m, NavigateTo("/issues/show")
				case key.Matches(msg, keys.Submit):
					return m, m.issueDetail.commentForm.Submit
				}
			}
		case strings.HasSuffix(m.path, "/edit/title"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Back):
					if m.issueForm.editing {
						m.path = "/issues/show"
						return m, cmd
					} else {
						m.path = "/issues/index"
						return m, cmd
					}
				case key.Matches(msg, keys.NextInput):
					m.path = "/issues/edit/description"
					m.issueForm.titleInput.Blur()
					cmd = m.issueForm.descriptionInput.Focus()
					return m, cmd
				}
			}

			m.issueForm.titleInput, cmd = m.issueForm.titleInput.Update(msg)
			return m, cmd
		case strings.HasSuffix(m.path, "/edit/description"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Back):
					if m.issueForm.editing {
						m.path = "/issues/show"
						return m, cmd
					} else {
						m.path = "/issues/index"
						return m, cmd
					}
				case key.Matches(msg, keys.NextInput):
					m.path = "/issues/edit/confirmation"
					m.issueForm.descriptionInput.Blur()
					return m, cmd
				}
			}

			m.issueForm.descriptionInput, cmd = m.issueForm.descriptionInput.Update(msg)
			return m, cmd
		case strings.HasSuffix(m.path, "/edit/confirmation"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Back):
					if m.issueForm.editing {
						m.path = "/issues/show"
						return m, cmd
					} else {
						m.path = "/issues/index"
						return m, cmd
					}
				case key.Matches(msg, keys.NextInput):
					m.path = "issues/edit/title"
					cmd = m.issueForm.titleInput.Focus()
					return m, cmd
				case key.Matches(msg, keys.Submit):
					return m, m.issueForm.Submit
				}
			}

			m.issueForm, cmd = m.issueForm.Update(componentUpdateMsg)
			return m, cmd
		}
	case strings.HasPrefix(m.path, "/checks"):
		switch {
		case strings.HasSuffix(m.path, "/index"):
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
					m.path = "/checks/show"
					return m, cmd
				case key.Matches(msg, keys.NextPage):
					return m, nil
				case key.Matches(msg, keys.PrevPage):
					m.path = "/issues/index"
					return m, NavigateTo("/issues/index")
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
		case strings.HasSuffix(m.path, "/show"):
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, keys.Back):
					m.path = "/checks/index"
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
		}
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
	case strings.HasPrefix(m.path, "/issues/index"):
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
		// Background(lipgloss.Color(fmt.Sprintf("%d", rand.Intn(255)))).
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
		case strings.HasSuffix(m.path, "/index"):

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
		case strings.HasSuffix(m.path, "/show"):
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
		case strings.HasSuffix(m.path, "/show/comments/new/content"):
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
		case strings.HasSuffix(m.path, "/show/comments/new/confirmation"):
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
		case strings.HasSuffix(m.path, "/edit/title"):
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
		case strings.HasSuffix(m.path, "/edit/description"):
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
		case strings.HasSuffix(m.path, "/edit/confirmation"):
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
		case strings.HasSuffix(m.path, "/index"):
			view = lipgloss.JoinVertical(
				lipgloss.Left,
				boxStyle(m.HeaderSize).Render(
					lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...),
				),
				boxStyle(m.LeftSize).Render(commitListView),
				boxStyle(m.FooterSize).Render(help),
			)
		case strings.HasSuffix(m.path, "/show"):
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
	return IssuesReadyMsg(seedIssues)
}

type commitDetailFocusState int

const (
	commitDetailViewportFocused commitDetailFocusState = 1
)

type commitDetailModel struct {
	commit   Commit
	viewport viewport.Model
	focus    commitDetailFocusState
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
	m.focus = commitDetailViewportFocused
	s.WriteString(m.commit.description)
	s.WriteString(fmt.Sprintf("\n\n%s", m.commit.latestCheck.output))

	// for i, comment := range m.commit.comments {
	// 	commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.author, comment.createdAt))
	// 	if i == len(m.commit.comments)-1 { // last comment
	// 		content += commentStyle.MarginBottom(2).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content))
	// 	} else {
	// 		content += commentStyle.Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content))
	// 	}
	// }
	m.viewport.SetContent(s.String())
	return nil
}

func (m commitDetailModel) Update(msg tea.Msg) (commitDetailModel, tea.Cmd) {
	var cmd tea.Cmd
	// msgg := msg.(updateMsg)
	// keys := msgg.keys

	// switch m.focus {
	// case commitDetailViewportFocused:
	// 	switch msg := msgg.originalMsg.(type) {
	// 	case tea.KeyMsg:
	// 		switch {
	// 		case key.Matches(msg, keys.CommitDetailFocus):
	// 			m.viewport.Height = m.viewport.Height - 7
	// 			m.focus = commitDetailCommentFocused
	// 			m.commentForm = NewCommentFormModel()
	// 			m.commentForm.Init()
	// 		}
	// 	}
	// 	m.viewport, cmd = m.viewport.Update(msg)
	// case commitDetailCommentFocused:
	// 	switch msg := msgg.originalMsg.(type) {
	// 	case tea.KeyMsg:
	// 		switch {
	// 		case key.Matches(msg, keys.Back):
	// 			m.focus = commitDetailViewportFocused
	// 		}
	// 	}
	// 	m.commentForm, cmd = m.commentForm.Update(msgg)
	// }
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m commitDetailModel) View() string {
	var s strings.Builder
	s.WriteString(m.viewport.View())
	return s.String()
}

type issueDetailFocusState int

const (
	issueDetailViewportFocused issueDetailFocusState = 1
	issueDetailCommentFocused  issueDetailFocusState = 2
)

type issueDetailModel struct {
	layout      Layout
	issue       Issue
	viewport    viewport.Model
	focus       issueDetailFocusState
	commentForm commentFormModel
}

func (m *issueDetailModel) Init(ctx Model) tea.Cmd {
	m.viewport = viewport.New(
		ctx.Layout.RightSize.Width,
		ctx.Layout.RightSize.Height-len(strings.Split(m.commentForm.View("content"), "\n")),
	)
	m.layout = ctx.Layout
	m.focus = issueDetailViewportFocused
	var s strings.Builder
	var status string
	switch m.issue.status {
	case todo:
		status = "todo"
	case inProgress:
		status = lipgloss.NewStyle().Foreground(styles.Theme.YellowText).Render("in-progress")
	case wontDo:
		status = lipgloss.NewStyle().Foreground(styles.Theme.RedText).Render("wont-do")
	case done:
		status = lipgloss.NewStyle().Foreground(styles.Theme.GreenText).Render("done")
	}
	identifier := lipgloss.NewStyle().Foreground(styles.Theme.SecondaryText).Render(fmt.Sprintf("#%s", m.issue.shortcode))
	header := fmt.Sprintf("%s %s\nStatus: %s", identifier, m.issue.title, status)
	s.WriteString(lipgloss.NewStyle().Render(header))
	s.WriteString(m.issue.description + "\n")

	commentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.Theme.SecondaryBorder).
		Width(m.viewport.Width - 2).
		MarginTop(1)

	commentHeaderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(styles.Theme.SecondaryBorder).
		Width(m.viewport.Width - 2)

	for i, comment := range m.issue.comments {
		commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.author, comment.createdAt))
		if i == len(m.issue.comments)-1 { // last comment
			s.WriteString(commentStyle.MarginBottom(0).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content)))
		} else {
			s.WriteString(commentStyle.Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content)))
		}
	}
	m.viewport.SetContent(s.String())
	return nil
}

type updateMsg struct {
	originalMsg tea.Msg
	keys        keyMap
}

func (m issueDetailModel) Update(msg tea.Msg) (issueDetailModel, tea.Cmd) {
	var cmd tea.Cmd
	msgg := msg.(updateMsg)

	switch m.focus {
	case issueDetailViewportFocused:
		switch msg := msgg.originalMsg.(type) {
		case tea.KeyMsg:
			switch {
			default:
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
		}
	case issueDetailCommentFocused:
		m.commentForm, cmd = m.commentForm.Update(msgg)
	}

	return m, cmd
}

func (m issueDetailModel) View() string {
	return m.viewport.View()
}

type issueFormFocusState int

type issueFormModel struct {
	titleInput       textinput.Model
	descriptionInput textarea.Model
	focusState       issueFormFocusState
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

type commentFormFocusState int

const (
	commentContentFocused      commentFormFocusState = 1
	commentConfirmationFocused commentFormFocusState = 2
)

type commentFormModel struct {
	contentInput textarea.Model
	focusState   commentFormFocusState
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
		focusState:   commentContentFocused,
	}
}

func (m commentFormModel) Submit() tea.Msg {
	return m
}

func (m commentFormModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m commentFormModel) Update(msg tea.Msg) (commentFormModel, tea.Cmd) {
	var cmd tea.Cmd
	msgg := msg.(updateMsg)
	keys := msgg.keys

	switch m.focusState {
	case commentContentFocused:
		switch msg := msgg.originalMsg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.NextInput):
				m.focusState = commentConfirmationFocused
				m.contentInput.Blur()
			}
		}

		m.contentInput, cmd = m.contentInput.Update(msgg.originalMsg)
	case commentConfirmationFocused:
		switch msg := msgg.originalMsg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.NextInput):
				m.focusState = commentContentFocused
				m.contentInput.Focus()
			case key.Matches(msg, keys.Submit):
				cmd = m.Submit
			}
		}
	}

	return m, cmd
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
