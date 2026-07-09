# frozen_string_literal: true
#
# Basic usage of Ruby's OpenStruct, backed by the pure-Go core in this repo.
# Runs under go-embedded-ruby (rbgo): rbgo examples/ostruct_usage.rb

require "ostruct"

# Build a struct from a Hash; every key becomes a reader/writer.
person = OpenStruct.new(name: "Ada", born: 1815)
puts person.name            # => Ada
puts person.born            # => 1815

# Any assignment defines a new attribute on the fly.
person.field = "Computing"
puts person.field           # => Computing

# Attributes are also reachable by Symbol via [] / []=.
person[:city] = "London"
puts person[:city]          # => London

# Introspection: to_h preserves insertion order, respond_to? is dynamic.
puts person.to_h.inspect    # => {name: "Ada", born: 1815, field: "Computing", city: "London"}
puts person.respond_to?(:name)  # => true

# each_pair iterates attributes in insertion order.
person.each_pair { |key, value| puts "#{key} = #{value}" }

# delete_field removes an attribute and returns its value.
person.delete_field(:city)
puts person.inspect         # => #<OpenStruct name="Ada", born=1815, field="Computing">

# Structural equality compares the attribute tables.
puts(OpenStruct.new(x: 1) == OpenStruct.new(x: 1))  # => true
