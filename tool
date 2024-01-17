#!/usr/bin/ruby

puts "hey"

# $ ./tool "this is the message"
#
# returns
#
# {
#   "message": "this is the message"
# }
#
# This will store the json message inside .git as a ref object
#
# Construct a JSON object with the message from STDIN
# Convert that to git's binary format
# Store that in .git refs or objects
# Once we do that, does "git push" automatically take those new objects with it to the remote?
# On the receiving end after we clone a repository or pull changes, this script should be able to read the custom object from .git
# and then print the JSON output
