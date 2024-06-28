package main

var seedIssues = []Issue{
	{
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
				author:  "dev@example.com",
				content: `The workaround and even the after_fork only works if the parallel processor is using processes and not threads. Unfortunately Rails::Generator::TestCase don't support parallel tests at the moment. We should probably disallow setting it at short-term and fix it to support parallel tests long term.`,
			},
			{
				author:  "garrett@blvrd.co",
				content: `@rafaelfranca do you have a preferred approach?`,
			},
			{
				author:  "garrett@blvrd.co",
				content: `If we can fix it, I'd prefer to try that first. Now, how we are going to make sure the destination dir is different for each test worker I don't know.`,
			},
		},
	},
}
