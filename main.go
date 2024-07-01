package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
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

var (
	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)

type Issue struct {
	id          string
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
	return i.title
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
	focusState  focusState
	issueList   list.Model
	issueDetail issueDetailModel
	issueForm   issueFormModel
	err         error
	totalWidth  int
	totalHeight int
}

func (m Model) percentageToWidth(percentage float32) int {
	return int(float32(m.totalWidth) * percentage)
}

func InitialModel() *Model {
	return &Model{
		focusState: issueListFocused,
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
	}

	if m.issueList.SettingFilter() {
		m.issueList, cmd = m.issueList.Update(msg)
		return m, cmd
	}

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
			case "enter":
				m.focusState = issueDetailFocused
				m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue)}
				m.issueDetail.Init(m)
				return m, cmd
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
			case "enter":
				if m.issueDetail.focus == issueDetailCommentFocused {
					m.issueDetail, cmd = m.issueDetail.Update(msg)
				} else {
					m.focusState = issueFormFocused
					m.issueForm = issueFormModel{}
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
		case issueFormModel:
			m.focusState = issueDetailFocused
			currentIndex := m.issueList.Index()
			currentIssue := m.issueList.SelectedItem().(Issue)
			currentIssue.title = msg.titleInput.Value()
			currentIssue.description = msg.descriptionInput.Value()
			m.issueList.SetItem(currentIndex, currentIssue)
		}

		m.issueForm, cmd = m.issueForm.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if !m.loaded {
		return "Loading..."
	}

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

	return lipgloss.JoinHorizontal(lipgloss.Top, issueListView, sidebarView)
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
	m.viewport = viewport.New(ctx.percentageToWidth(0.4), ctx.totalHeight-4)
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
	s.WriteString(lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).PaddingTop(1).Render(m.issue.title))
	s.WriteString("\n")
	s.WriteString(m.viewport.View())
	s.WriteString("\n")
	percentage := fmt.Sprintf("%f", m.viewport.ScrollPercent()*100)
	s.WriteString(percentage)

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
