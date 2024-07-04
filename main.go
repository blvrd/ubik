package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
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

type status int

const (
	todo       status = 1
	inProgress status = 2
	done       status = 3
	wontDo     status = 4
)

type focusState int

const (
	issueListFocused   focusState = 1
	issueDetailFocused focusState = 2
	issueFormFocused   focusState = 3
)

type pageState int

const (
	issues pageState = 1
	checks pageState = 2
)

type issueListKeyMap struct {
	Up                    key.Binding
	Down                  key.Binding
	Left                  key.Binding
	Right                 key.Binding
	Help                  key.Binding
	Quit                  key.Binding
	IssueNewForm          key.Binding
	IssueDetailFocus      key.Binding
	IssueStatusDone       key.Binding
	IssueStatusWontDo     key.Binding
	IssueStatusInProgress key.Binding
	IssueCommentFormFocus key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k issueListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k issueListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.IssueStatusDone, k.IssueStatusWontDo, k.IssueStatusInProgress, k.IssueCommentFormFocus},
		{k.IssueNewForm, k.IssueDetailFocus, k.Help, k.Quit},
	}
}

type issueDetailKeyMap struct {
	Up                    key.Binding
	Down                  key.Binding
	Left                  key.Binding
	Right                 key.Binding
	Help                  key.Binding
	Quit                  key.Binding
	Back                  key.Binding
	IssueEditForm         key.Binding
	IssueStatusDone       key.Binding
	IssueStatusWontDo     key.Binding
	IssueStatusInProgress key.Binding
	IssueCommentForm      key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k issueDetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k issueDetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.IssueStatusDone, k.IssueStatusWontDo, k.IssueStatusInProgress, k.IssueCommentForm},
		{k.IssueEditForm, k.Help, k.Back, k.Quit},
	}
}

type issueFormKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Help      key.Binding
	Quit      key.Binding
	Back      key.Binding
	NextInput key.Binding
	Submit    key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k issueFormKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k issueFormKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextInput, k.Up, k.Down},
		{k.Help, k.Back, k.Quit},
	}
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

func (i Issue) Title() string {
	var status string

	switch i.status {
	case todo:
		status = "[·]"
	case inProgress:
		status = "[⋯]"
	case wontDo:
		status = "[×]"
	case done:
		status = "[✓]"
	}
	return fmt.Sprintf("%s #%s %s", status, i.shortcode, i.title)
}

func (i Issue) Description() string {
	return fmt.Sprintf("created by %s at %s", i.author, i.createdAt.Format(time.RFC822))
}

type Comment struct {
	author    string
	content   string
	createdAt time.Time
	updatedAt time.Time
}

/* MAIN MODEL */

type Model struct {
	loaded      bool
	page        pageState
	focusState  focusState
	issueList   list.Model
	issueDetail issueDetailModel
	issueForm   issueFormModel
	err         error
	totalWidth  int
	totalHeight int
	help        help.Model
	keys        issueListKeyMap
}

func (m Model) percentageToWidth(percentage float32) int {
	return int(float32(m.totalWidth) * percentage)
}

func InitialModel() *Model {
	var keys = issueListKeyMap{
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
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
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
	}

	return &Model{
		focusState: issueListFocused,
		keys:       keys,
		help:       help.New(),
    page:       checks,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.loaded = true
		}
		m.totalWidth = msg.Width
		m.totalHeight = msg.Height
		m.initIssueList(msg.Width, msg.Height-4)
		m.issueList, cmd = m.issueList.Update(msg)
	case commentFormModel:
		currentIndex := m.issueList.Index()
		currentIssue := m.issueList.SelectedItem().(Issue)
		currentIssue.comments = append(currentIssue.comments, Comment{author: "garrett@blvrd.co", content: msg.contentInput.Value()})
		m.issueList.SetItem(currentIndex, currentIssue)
		m.issueDetail = issueDetailModel{issue: currentIssue}
		m.issueDetail.Init(m)
		m.issueDetail.viewport.GotoBottom()

		return m, tea.Batch(cmds...)
	case issueFormModel:
		if msg.editing {
			m.focusState = issueDetailFocused
			currentIndex := m.issueList.Index()
			currentIssue := m.issueList.SelectedItem().(Issue)
			currentIssue.title = msg.titleInput.Value()
			currentIssue.description = msg.descriptionInput.Value()
			m.issueDetail = issueDetailModel{issue: currentIssue}
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
			m.focusState = issueDetailFocused
			m.issueDetail = issueDetailModel{issue: newIssue}
			m.issueDetail.Init(m)
		}
	case tea.KeyMsg:
		if m.focusState == issueFormFocused || m.issueDetail.focus == issueDetailCommentFocused {
			break
		}

		switch {
		case key.Matches(msg, m.keys.Help):
			if m.help.ShowAll {
				m.help.ShowAll = false
				m.issueList.SetHeight(m.issueList.Height() + 4)
			} else {
				m.help.ShowAll = true
				m.issueList.SetHeight(m.issueList.Height() - 4)
			}
		}
	}

	if m.issueList.SettingFilter() {
		m.issueList, cmd = m.issueList.Update(msg)
		return m, cmd
	}

	switch m.page {
	case issues:
		if m.focusState == issueListFocused {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "j":
					m.issueList, cmd = m.issueList.Update(msg)
					return m, cmd
				case "k":
					m.issueList, cmd = m.issueList.Update(msg)
					return m, cmd
				case " ":
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = done
					} else {
						currentIssue.status = todo
					}
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case "w":
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = wontDo
					} else {
						currentIssue.status = todo
					}
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case "p":
					currentIndex := m.issueList.Index()
					currentIssue := m.issueList.SelectedItem().(Issue)
					if currentIssue.status == todo {
						currentIssue.status = inProgress
					} else {
						currentIssue.status = todo
					}
					cmd = m.issueList.SetItem(currentIndex, currentIssue)
					return m, cmd
				case "c":
					m.focusState = issueDetailFocused
					m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
					m.issueDetail.Init(m)
					m.issueDetail, cmd = m.issueDetail.Update(msg)
					return m, cmd
				case "enter":
					m.focusState = issueDetailFocused
					m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
					m.issueDetail.Init(m)
					return m, cmd
				case "n":
					m.focusState = issueFormFocused
					m.issueForm = issueFormModel{editing: false}
					m.issueForm.SetTitle("")
					m.issueForm.SetDescription("")
					m.issueForm.focusState = issueTitleFocused
					cmd = m.issueForm.titleInput.Focus()
				}
			}

			m.issueList, cmd = m.issueList.Update(msg)
		} else if m.focusState == issueDetailFocused {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "j":
					m.issueDetail, cmd = m.issueDetail.Update(msg)
					return m, cmd
				case "k":
					m.issueDetail, cmd = m.issueDetail.Update(msg)
					return m, cmd
				case " ":
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
				case "w":
					if m.focusState == issueFormFocused || m.issueDetail.focus == issueDetailCommentFocused {
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
				case "p":
					if m.focusState == issueFormFocused || m.issueDetail.focus == issueDetailCommentFocused {
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
				case "enter":
					if m.issueDetail.focus == issueDetailCommentFocused {
						m.issueDetail, cmd = m.issueDetail.Update(msg)
					} else {
						m.focusState = issueFormFocused
						m.issueForm = issueFormModel{editing: true}
						selectedIssue := m.issueList.SelectedItem().(Issue)
						m.issueForm.SetTitle(selectedIssue.title)
						m.issueForm.SetDescription(selectedIssue.description)
						m.issueForm.focusState = issueTitleFocused
						cmd = m.issueForm.titleInput.Focus()
					}

					return m, cmd
				case "c":
					m.issueDetail, cmd = m.issueDetail.Update(msg)
					return m, cmd
				case "esc":
					if m.issueDetail.focus == issueDetailViewportFocused {
						m.focusState = issueListFocused
					} else {
						m.issueDetail, cmd = m.issueDetail.Update(msg)
					}
					return m, cmd
				}
			}

			m.issueDetail, cmd = m.issueDetail.Update(msg)
		} else if m.focusState == issueFormFocused {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "esc":
					m.focusState = issueDetailFocused
					return m, cmd
				}
			}

			m.issueForm, cmd = m.issueForm.Update(msg)
		}
	case checks:

	}

	return m, cmd
}

func (m Model) HelpKeys() help.KeyMap {
	var keys help.KeyMap
	switch m.focusState {
	case issueListFocused:
		keys = issueListKeyMap{
			Up: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "move up"),
			),
			Down: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "move down"),
			),
			IssueDetailFocus: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "more info"),
			),
			Help: key.NewBinding(
				key.WithKeys("?"),
				key.WithHelp("?", "toggle help"),
			),
			Quit: key.NewBinding(
				key.WithKeys("q", "esc", "ctrl+c"),
				key.WithHelp("q", "quit"),
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
		}
	case issueDetailFocused:
		keys = issueDetailKeyMap{
			Up: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "scroll up"),
			),
			Down: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "scroll down"),
			),
			Help: key.NewBinding(
				key.WithKeys("?"),
				key.WithHelp("?", "toggle help"),
			),
			IssueEditForm: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "edit issue"),
			),
			Quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
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
			IssueCommentForm: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "toggle issue comment form"),
			),
		}
	case issueFormFocused:
		keys = issueFormKeyMap{
			Up: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "scroll up"),
			),
			Down: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "scroll down"),
			),
			Help: key.NewBinding(
				key.WithKeys("?"),
				key.WithHelp("?", "toggle help"),
			),
			Quit: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "quit"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
			NextInput: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next input"),
			),
		}
	}

	return keys
}

func (m Model) View() string {
	if !m.loaded {
		return "Loading..."
	}

	var view string

	switch m.page {
	case issues:
		issueListView := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Width(m.percentageToWidth(0.5)).
			// MarginRight(2).
			Render(m.issueList.View())
		var sidebarView string

		switch m.focusState {
		case issueDetailFocused:
			sidebarView = lipgloss.NewStyle().
				// Border(lipgloss.NormalBorder()).
				// BorderForeground(lipgloss.Color("238")).
				Width(m.percentageToWidth(0.4)).
				// MarginLeft(2).
				Render(m.issueDetail.View())
		case issueFormFocused:
			style := lipgloss.NewStyle().
				// Border(lipgloss.NormalBorder()).
				// BorderForeground(lipgloss.Color("238")).
				Width(m.percentageToWidth(0.4))
			// MarginLeft(2)

			sidebarView = style.
				Render(m.issueForm.View())

		}

		help := m.help.View(m.HelpKeys())
		view = lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Top, issueListView, sidebarView), help)
	case checks:
		view = "hey"
	}

	return view
}

func (m *Model) initIssueList(width, height int) {
	m.issueList = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	m.issueList.SetShowHelp(false)
	m.issueList.Title = "Issues"
	var listItems []list.Item
	for _, issue := range seedIssues {
		listItems = append(listItems, issue)
	}
	m.issueList.SetItems(listItems)
}

type issueDetailFocusState int

const (
	issueDetailViewportFocused issueDetailFocusState = 1
	issueDetailCommentFocused  issueDetailFocusState = 2
)

type issueDetailModel struct {
	issue       Issue
	viewport    viewport.Model
	focus       issueDetailFocusState
	commentForm commentFormModel
}

func (m *issueDetailModel) Init(ctx Model) tea.Cmd {
	// m.viewport = viewport.New(ctx.percentageToWidth(0.4), ctx.totalHeight-4)
	m.viewport = viewport.New(ctx.percentageToWidth(0.4), 50)
	m.focus = issueDetailViewportFocused
	content := m.issue.description

	commentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Width(m.viewport.Width - 2).
		MarginTop(1)

	commentHeaderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("238")).
		Width(40)

	for i, comment := range m.issue.comments {
		commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.author, comment.createdAt))
		if i == len(m.issue.comments)-1 { // last comment
			content += commentStyle.MarginBottom(2).Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content))
		} else {
			content += commentStyle.Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content))
		}
	}
	m.viewport.SetContent(content)
	return nil
}

func (m issueDetailModel) Update(msg tea.Msg) (issueDetailModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.focus == issueDetailViewportFocused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "c":
				m.viewport.Height = m.viewport.Height - 7
				m.focus = issueDetailCommentFocused
				m.commentForm = NewCommentFormModel()
				m.commentForm.Init()
			}
		}
		m.viewport, cmd = m.viewport.Update(msg)
	} else if m.focus == issueDetailCommentFocused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.focus = issueDetailViewportFocused
			}
		}
		m.commentForm, cmd = m.commentForm.Update(msg)
	}
	return m, cmd
}

func (m issueDetailModel) View() string {
	var s strings.Builder
	var status string
	switch m.issue.status {
	case todo:
		status = "todo"
	case inProgress:
		status = "in-progress"
	case wontDo:
		status = "wont-do"
	case done:
		status = "done"
	}
	header := fmt.Sprintf("%s\nStatus: %s", m.issue.title, status)
	s.WriteString(lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).PaddingTop(1).Render(header))
	s.WriteString("\n")
	s.WriteString(m.viewport.View())
	s.WriteString("\n")
	// percentage := fmt.Sprintf("%f", m.viewport.ScrollPercent()*100)
	// s.WriteString(percentage)

	if m.focus == issueDetailCommentFocused {
		s.WriteString("\n")
		s.WriteString(m.commentForm.View())
	}

	return s.String()
}

type issueFormFocusState int

const (
	issueTitleFocused        issueFormFocusState = 1
	issueDescriptionFocused  issueFormFocusState = 2
	issueConfirmationFocused issueFormFocusState = 3
)

type issueFormModel struct {
	titleInput       textinput.Model
	descriptionInput textarea.Model
	focusState       issueFormFocusState
	editing          bool
}

func (m issueFormModel) Submit() tea.Msg {
	return m
}

func (m issueFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m issueFormModel) Update(msg tea.Msg) (issueFormModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			switch m.focusState {
			case issueTitleFocused:
				m.focusState = issueDescriptionFocused
				m.titleInput.Blur()
				cmd = m.descriptionInput.Focus()
			case issueDescriptionFocused:
				m.focusState = issueConfirmationFocused
				m.descriptionInput.Blur()
			}

			return m, cmd
		case "enter":
			if m.focusState == issueConfirmationFocused {
				return m, m.Submit
			}
		}
	}

	switch m.focusState {
	case issueTitleFocused:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case issueDescriptionFocused:
		m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	}

	return m, cmd
}

func (m issueFormModel) View() string {
	var s strings.Builder

	s.WriteString(m.titleInput.View())
	s.WriteString("\n")
	s.WriteString(m.descriptionInput.View())
	s.WriteString("\n")
	if m.focusState == issueConfirmationFocused {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF"))
		s.WriteString(style.Render("Save"))
	} else {
		s.WriteString("Save")
	}

	return s.String()
}

func (m *issueFormModel) SetTitle(title string) {
	m.titleInput = textinput.New()
	m.titleInput.CharLimit = 120
	m.titleInput.SetValue(title)
}

func (m *issueFormModel) SetDescription(description string) {
	m.descriptionInput = textarea.New()
	m.descriptionInput.CharLimit = 0 // unlimited
	m.descriptionInput.MaxHeight = 0 // unlimited
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
	return commentFormModel{
		contentInput: textarea.New(),
		focusState:   commentContentFocused,
	}
}

func (m commentFormModel) Submit() tea.Msg {
	return m
}

func (m *commentFormModel) Init() tea.Cmd {
	m.focusState = commentContentFocused
	m.contentInput.Focus()
	return textinput.Blink
}

func (m commentFormModel) Update(msg tea.Msg) (commentFormModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			switch m.focusState {
			case commentContentFocused:
				m.focusState = commentConfirmationFocused
				m.contentInput.Blur()
			case commentConfirmationFocused:
				m.focusState = commentContentFocused
			}

			return m, cmd
		case "enter":
			if m.focusState == commentConfirmationFocused {
				return m, m.Submit
			}
		}
	}

	switch m.focusState {
	case commentContentFocused:
		m.contentInput, cmd = m.contentInput.Update(msg)
	}

	return m, cmd
}

func (m commentFormModel) View() string {
	var s strings.Builder
	s.WriteString(m.contentInput.View())
	s.WriteString("\n")
	if m.focusState == commentContentFocused {
		s.WriteString("Save")
	} else {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("200")).Render("Save"))
	}
	return s.String()
}

func main() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gomnd
	if err != nil {
		fmt.Printf("error opening file for logging: %s", err)
		os.Exit(1)
	}
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)

	if err != nil {
		log.Print("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	if _, err := p.Run(); err != nil {
		log.Print(err)
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
