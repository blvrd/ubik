#!/usr/bin/ruby

require "json"
require "digest/sha1"
require "zlib"
require "pathname"

ob = {
  message: ARGV[0]
}

TEMP_CHARS = ("a".."z").to_a + ("A".."Z").to_a + ("0".."9").to_a

json = JSON.generate(ob)
id = Digest::SHA1.hexdigest(json)
def write_object(oid, content)
  root_path = Pathname.new(Dir.getwd)
  git_path  = root_path.join(".git")
  db_path   = git_path.join("refs").join("bugs")

  object_path = db_path.join(oid[0..1], oid[2..-1])
  return if File.exists?(object_path)

  dirname     = object_path.dirname
  temp_path   = dirname.join(generate_temp_name)

  begin
    flags = File::RDWR | File::CREAT | File::EXCL
    file = File.open(temp_path, flags)
  rescue Errno::ENOENT
    Dir.mkdir(dirname)
    file = File.open(temp_path, flags)
  end

  compressed = Zlib::Deflate.deflate(content, Zlib::BEST_SPEED)

  file.write(compressed)
  file.close

  File.rename(temp_path, object_path)
end

def generate_temp_name
  "tmp_obj_#{(1..6).map { TEMP_CHARS.sample }.join("")}"
end

write_object(id, json)

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
#
# require "digest/sha1"
# require "zlib"
#
# require_relative "./blob"
#
# class Database
#   TEMP_CHARS = ("a".."z").to_a + ("A".."Z").to_a + ("0".."9").to_a
#
#   def initialize(pathname)
#     @pathname = pathname
#   end
#
#   def store(object)
#     string = object.to_s.force_encoding(Encoding::ASCII_8BIT)
#     content = "#{object.type} #{string.bytesize}\0#{string}"
#
#     object.oid = Digest::SHA1.hexdigest(content)
#     write_object(object.oid, content)
#   end
#
# private
#
#   def write_object(oid, content)
#     object_path = @pathname.join(oid[0..1], oid[2..-1])
#     return if File.exists?(object_path)
#
#     dirname     = object_path.dirname
#     temp_path   = dirname.join(generate_temp_name)
#
#     begin
#       flags = File::RDWR | File::CREAT | File::EXCL
#       file = File.open(temp_path, flags)
#     rescue Errno::ENOENT
#       Dir.mkdir(dirname)
#       file = File.open(temp_path, flags)
#     end
#
#     compressed = Zlib::Deflate.deflate(content, Zlib::BEST_SPEED)
#
#     file.write(compressed)
#     file.close
#
#     File.rename(temp_path, object_path)
#   end
#
#   def generate_temp_name
#     "tmp_obj_#{(1..6).map { TEMP_CHARS.sample }.join("")}"
#   end
# end
#
