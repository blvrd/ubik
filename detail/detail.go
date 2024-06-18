package detail

import (
	"fmt"
	"strings"
	"time"

	"github.com/blvrd/ubik/entity"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	// "github.com/charmbracelet/log"
)

const (
	viewportHeight int = 30
	spacebar           = " "
)

// DefaultKeyMap returns a set of pager-like default keybindings.
func DefaultKeyMap() viewport.KeyMap {
	return viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f"),
			key.WithHelp("f/pgdn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("b/pgup", "page up"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "½ page down"),
		),
		Up: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("J", "scroll down"),
		),
	}
}

type Model struct {
	ent      *entity.Issue
	viewport viewport.Model
}

var commentStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#838383"))

var commentHeaderStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(lipgloss.Color("#838383"))

func New(ent *entity.Issue) Model {
	m := Model{
		ent:      ent,
		viewport: viewport.New(60, viewportHeight),
	}
	entMap := ent.ToMap()
	content := []string{entMap["description"].(string)}
	content = append(content, "\nComments:\n")
	for _, comment := range entMap["comments"].([]entity.Comment) {
		commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.Author, comment.CreatedAt.Format(time.RFC822)))
		content = append(content, commentStyle.Render(fmt.Sprintf("%s\n\n %s\n", commentHeader, comment.Body)))
	}

	m.viewport.SetContent(strings.Join(content, "\n"))
	m.viewport.KeyMap = DefaultKeyMap()

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, []tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	cmds = append(cmds, cmd)

	return m, cmds
}

func (m Model) View() string {
	var s strings.Builder

	if m.ent == nil {
		s.WriteString("Nothing selected")
		return s.String()
	}

	entMap := m.ent.ToMap()

	var closed string

	closedAt := entMap["closed_at"].(time.Time)

	if closedAt.IsZero() {
		closed = lipgloss.NewStyle().Foreground(lipgloss.Color("#838383")).Render("[·] Open")
	} else {
		closed = lipgloss.NewStyle().Foreground(lipgloss.Color("#5db158")).Render(fmt.Sprintf("[✓] Closed %s", closedAt.Format(time.RFC822)))
	}

	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s", entMap["title"]))
	shortcode := lipgloss.NewStyle().Foreground(lipgloss.Color("#838383")).Render(fmt.Sprintf("#%s", entMap["shortcode"]))
	author := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(fmt.Sprintf("%s", entMap["author"]))
	createdAt := entMap["created_at"].(time.Time).Format(time.RFC822)
	descriptionReadPercentage := m.viewport.ScrollPercent() * 100

	descriptionReadPercentageIndicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#838383")).Render(fmt.Sprintf("%3.f%%", descriptionReadPercentage))

	s.WriteString(fmt.Sprintf("%s %s\n\n", title, shortcode))
	s.WriteString(fmt.Sprintf("%s - 3 comments\n", closed))
	s.WriteString(fmt.Sprintf("opened by %s on %s\n\n", author, createdAt))

	s.WriteString(m.viewport.View())

	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		s.WriteString(descriptionReadPercentageIndicator)
	}

	return s.String()
}
