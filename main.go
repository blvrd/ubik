package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	return i.description
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

  switch msg := msg.(type) {
  case tea.WindowSizeMsg:
    log.Println("window size msg")
    if !m.loaded {
      m.loaded = true
    }
    m.totalWidth = msg.Width
    m.totalHeight = msg.Height
    m.initIssueList(msg.Width, msg.Height-4)
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
				m.issueDetail.Init()
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
				m.focusState = issueFormFocused
				m.issueForm = issueFormModel{}
				selectedIssue := m.issueList.SelectedItem().(Issue)
				m.issueForm.SetTitle(selectedIssue.title)
				m.issueForm.SetDescription(selectedIssue.description)
				m.issueForm.focusState = titleFocused
				cmd = m.issueForm.titleInput.Focus()

				return m, cmd
			case "esc":
				m.focusState = issueListFocused
				return m, cmd
			}
		}

		m.issueList, cmd = m.issueList.Update(msg)
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

	issueListView := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Width(50).Render(m.issueList.View())
	var sidebarView string

	switch m.focusState {
	case issueDetailFocused:
		sidebarView = lipgloss.NewStyle().Width(100).Render(m.issueDetail.View())
	case issueFormFocused:
		sidebarView = lipgloss.NewStyle().Width(100).Render(m.issueForm.View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, issueListView, sidebarView)
}

func (m *Model) initIssueList(width, height int) {
	m.issueList = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	m.issueList.SetShowHelp(false)
	m.issueList.Title = "Issues"
	m.issueList.SetItems([]list.Item{
		Issue{
			id:     "12345",
			author: "garrett@blvrd.co",
			title:  "accepts_nested_attributes_for doesn't validate unchanged objects",
			description: `
Steps to reproduce

Updating records through nested attributes behaves differently than updating records directly if they have validations that are set to run on update only.

require 'bundler/inline'

gemfile(true) do
  source 'https://rubygems.org'

  gem 'rails', '7'
  gem 'sqlite3', '~> 1.7'
end

require 'active_record'
require 'minitest/autorun'
require 'logger'

# This connection will do for database-independent bug reports.
ActiveRecord::Base.establish_connection(adapter: 'sqlite3', database: ':memory:')
ActiveRecord::Base.logger = Logger.new(STDOUT)

ActiveRecord::Schema.define do
  create_table :lists, force: true

  create_table :items, force: true do |t|
    t.integer :list_id
    t.text :name
    t.text :description
  end
end

class List < ActiveRecord::Base
  has_many :items
  accepts_nested_attributes_for :items

  # Works as expected but with a generic
  # error message (e.g. "items is invalid"
  # instead of "items name must be present")
  #   validates_associated :items
end

class Item < ActiveRecord::Base
  belongs_to :list
  validates :name, presence: true, on: :update
end

class BugTest < Minitest::Test
  def test_updating_item_directly
    list = List.create!
    item = list.items.create!(name: "")

    # Works as expected.
    assert_raises ActiveRecord::RecordInvalid do
        item.update!(name: "")
    end
  end

  def test_updating_unchanged_item_through_parent
    list = List.create!
    item = list.items.create!(name: "")

    # Doesn't raise anything.
    assert_raises ActiveRecord::RecordInvalid do
        list.update!(
            items_attributes: [
                { "id" => item.id, name: "" }
            ]
        )
    end
  end

  def test_updating_changed_item_through_parent
    list = List.create!
    item = list.items.create!(name: "")

    # Works as expected if additional attributes
    # of the item are updated.
    assert_raises ActiveRecord::RecordInvalid do
        list.update!(
            items_attributes: [
                { "id" => item.id, name: "", description: "" }
            ]
        )
    end
  end
end
Expected behavior

Updating an Item with a blank name using nested attributes should raise a validation error just like it does when the Item is updated directly.

Actual behavior

No error is raised if the object is unchanged. Interestingly, if I include additional attributes to update it works as expected.

If I add validates_associated to List it does raise an error but the message is generic and doesn't include which attribute is invalid.

System configuration

Rails version: 7

Ruby version: 3
      `,
			status: 1,
			comments: []Comment{
				{
          author: "garrett@blvrd.co",
					content: `
Yeah, it won't save the item unless it's changed. Code is here: https://github.com/rails/rails/blob/main/activerecord/lib/active_record/autosave_association.rb#L273

If I add a return true your test passes.

I guess it makes sense not to save ... why save if it hasn't changed? Of course, like in your example, you can force it with item.update!.

Maybe it would be good to be able to do items_attributes!: [ (bang method on the key) ... which would force a save.
          `,
				},
				{
          author: "dev@example.com",
					content: `@justinko - thanks a lot for looking at this and pointing me to the code. I'm able to override changed_for_autosave? and it seems to be doing exactly what I'm looking for.

> why save if it hasn't changed?
I don't want it to hit the DB if the record has no changes but I do want validations to run. My use case is that a record is created for a user by another process (hence my validations only running on update). The user is presented with a form and should see validation error messaging if they attempt to update the record without populating the required values.

> Maybe it would be good to be able to do items_attributes!: [ (bang method on the key)
I like that! I'm happy to take a stab at a PR if that's something that would be of interest.`,
				},
			},
		},
		Issue{
			id:     "54321",
			author: "garrett@blvrd.co",
			title:  "Parallelized generator tests fail in race condition because destination is not worker aware",
			description: `
### Steps to reproduce

Write generator tests and turn on parallel testing.

Rails::Generators::TestCase expects a class level 'destination' https://github.com/rails/rails/blob/main/railties/lib/rails/generators/testing/behavior.rb#L46 but inherits from ActiveSupport::TestCase so if parallel testing is on https://guides.rubyonrails.org/testing.html#parallel-testing-with-processes the test cases can race creating/destroying the directory


### Expected behavior

Per parallel executor destinations


My hack to get around this for now in the test case:

def prepare_destination
  self.destination_root = File.expand_path(\"../tmp\", __dir__) + \"-#{Process.pid}\"
  super
end

Maybe destination should use the after fork hook like https://github.com/rails/rails/blob/main/activerecord/lib/active_record/test_databases.rb#L7 ?  Or maybe a cleaned up version of my workaround would suffice?

### System configuration

**Rails version**:

7.1.8.4

**Ruby version**:

3.1.6
      `,
			status: 1,
			comments: []Comment{
				{
          author: "dev@example.com",
					content: `The workaround and even the after_fork only works if the parallel processor is using processes and not threads. Unfortunately Rails::Generator::TestCase don't support parallel tests at the moment. We should probably disallow setting it at short-term and fix it to support parallel tests long term.`,
				},
				{
          author: "garrett@blvrd.co",
					content: `@rafaelfranca do you have a preferred approach?`,
				},
				{
          author: "garrett@blvrd.co",
					content: `If we can fix it, I'd prefer to try that first. Now, how we are going to make sure the destination dir is different for each test worker I don't know.`,
				},
			},
		},
	})
}

type issueDetailModel struct {
	issue    Issue
	viewport viewport.Model
}

func (m *issueDetailModel) Init() tea.Cmd {
	m.viewport = viewport.New(90, 40)
  content := m.issue.description

  commentStyle := lipgloss.NewStyle().
    Border(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("238")).
    Width(80).
    MarginTop(1)
  commentHeaderStyle := lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, true, false).
    BorderForeground(lipgloss.Color("238")).
    Width(80)

  for _, comment := range m.issue.comments {
    commentHeader := commentHeaderStyle.Render(fmt.Sprintf("%s commented at %s", comment.author, comment.createdAt))
    content += commentStyle.Render(fmt.Sprintf("%s\n%s\n", commentHeader, comment.content))
  }
	m.viewport.SetContent(content)
  m.viewport.GotoBottom()
	return nil
}

func (m issueDetailModel) Update(msg tea.Msg) (issueDetailModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (id issueDetailModel) View() string {
	return id.viewport.View()
}

type formFocusState int

const (
	titleFocused        formFocusState = 1
	descriptionFocused  formFocusState = 2
	confirmationFocused formFocusState = 3
)

type issueFormModel struct {
	titleInput       textinput.Model
	descriptionInput textarea.Model
	focusState       formFocusState
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
			case titleFocused:
				m.focusState = descriptionFocused
				m.titleInput.Blur()
				cmd = m.descriptionInput.Focus()
			case descriptionFocused:
				m.focusState = confirmationFocused
				m.descriptionInput.Blur()
			}

			return m, cmd
		case "enter":
			if m.focusState == confirmationFocused {
				return m, m.Submit
			}
		}
	}

	switch m.focusState {
	case titleFocused:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case descriptionFocused:
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
	if m.focusState == confirmationFocused {
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

func main() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gomnd
	if err != nil {
		fmt.Printf("error opening file for logging: %s", err)
		os.Exit(1)
	}
	log.SetOutput(f)

	if err != nil {
		log.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	if _, err := p.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
