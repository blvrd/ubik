package main

import (
	"fmt"

  "github.com/google/uuid"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	errMsg error
)

const (
	title = iota
	desc
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

type model struct {
	titleInput  textinput.Model
  descriptionInput   textarea.Model
	focused int
	err     error
}

func initialModel() model {
  ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 80
	ti.Width = 80
	ti.Prompt = ""

  ta := textarea.New()
	ta.Placeholder = "Describe your issue"
	ta.Prompt = ""

  ta.SetWidth(50)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	return model{
		titleInput:  ti,
    descriptionInput: ta,
		focused: 0,
		err:     nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, 2)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
      // last input
			if m.focused == 1 {
        // Add issue
        title := m.titleInput.Value()
        description := m.descriptionInput.Value()
        issue := Issue{
          Id:          uuid.New().String(),
          Author:      GetAuthorEmail(),
          Title:       title,
          Description: description,
          Closed:      "false",
          RefPath:     issuesPath,
        }

        Add(issue)
				return m, tea.Quit
			}
			m.nextInput()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN:
			m.nextInput()
		}

		m.titleInput.Blur()
		m.descriptionInput.Blur()
    if m.focused == 0 {
      m.titleInput.Focus()
    } else {
      m.descriptionInput.Focus()
    }

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

  m.titleInput, cmds[0] = m.titleInput.Update(msg)
  m.descriptionInput, cmds[1] = m.descriptionInput.Update(msg)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return fmt.Sprintf(
		` Add an issue:

 %s
 %s

 %s
 %s

 %s
`,
		inputStyle.Width(80).Render("Title"),
		m.titleInput.View(),
		inputStyle.Width(120).Render("Description"),
		m.descriptionInput.View(),
		continueStyle.Render("Continue ->"),
	) + "\n"
}

// nextInput focuses the next input field
func (m *model) nextInput() {
	m.focused = (m.focused + 1) % 2
}

// prevInput focuses the previous input field
func (m *model) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = 2 - 1
	}
}
