package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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

/* MAIN MODEL */

type Model struct {
	loaded      bool
	focusState  focusState
	issueList   list.Model
	issueDetail issueDetailModel
	issueForm   issueFormModel
	err         error
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

	if m.focusState == issueListFocused {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			if !m.loaded {
				m.loaded = true
			}
			m.initIssueList(msg.Width, msg.Height)
		case tea.KeyMsg:
			switch msg.String() {
			case "j":
				m.issueList, cmd = m.issueList.Update(msg)
				m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue), viewport: viewport.New(30, 40)}
				return m, cmd
			case "k":
				m.issueList, cmd = m.issueList.Update(msg)
				m.issueDetail = issueDetailModel{issue: m.issueList.SelectedItem().(Issue), viewport: viewport.New(30, 40)}
				return m, cmd
			case "enter":
				m.focusState = issueDetailFocused
				m.issueDetail.visible = true
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
				m.issueDetail.visible = false
				m.issueForm.SetTitle(selectedIssue.title)
				m.issueForm.SetDescription(selectedIssue.description)
				m.issueForm.titleInput.Focus()
				m.issueForm.visible = true

				return m, cmd
			case "esc":
				m.focusState = issueListFocused
				m.issueDetail.visible = false
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
				m.issueForm.visible = false
				m.issueDetail.visible = true
				return m, cmd
			}
		}

		m.issueForm, cmd = m.issueForm.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if !m.loaded {
		return "Loading..."
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, m.issueList.View(), m.issueDetail.View(), m.issueForm.View())
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

      nMaybe destination should use the after fork hook like https://github.com/rails/rails/blob/main/activerecord/lib/active_record/test_databases.rb#L7 ?  Or maybe a cleaned up version of my workaround would suffice?

      ### System configuration

      **Rails version**:

      7.1.8.4

      **Ruby version**:

      3.1.6
      `,
			status: 1,
		},
	})

	m.issueDetail = issueDetailModel{
		issue:    m.issueList.SelectedItem().(Issue),
		viewport: viewport.New(30, 40),
	}
}

type issueDetailModel struct {
	issue    Issue
	visible  bool
	viewport viewport.Model
}

func (id issueDetailModel) Init() tea.Cmd {
	return nil
}

func (m issueDetailModel) Update(msg tea.Msg) (issueDetailModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (id issueDetailModel) View() string {
	if !id.visible {
		return ""
	}
	id.viewport.SetContent(id.issue.description)
	return id.viewport.View()
}

type issueFormModel struct {
	titleInput       textinput.Model
	descriptionInput textarea.Model
	visible          bool
}

func (m issueFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m issueFormModel) Update(msg tea.Msg) (issueFormModel, tea.Cmd) {
	var cmd tea.Cmd

	m.titleInput, cmd = m.titleInput.Update(msg)
	m.descriptionInput, cmd = m.descriptionInput.Update(msg)

	return m, cmd
}

func (m issueFormModel) View() string {
	if !m.visible {
		return ""
	}

	var s strings.Builder

	s.WriteString(m.titleInput.View())
	s.WriteString("\n")
	s.WriteString(m.descriptionInput.View())

	return s.String()
}

func (m *issueFormModel) SetTitle(title string) {
	m.titleInput = textinput.New()
	m.titleInput.SetValue(title)
}

func (m *issueFormModel) SetDescription(description string) {
	m.descriptionInput = textarea.New()
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
