# Ruby examples

Pure-Ruby examples for `OpenStruct`, whose data table is implemented by the
pure-Go core in this repo and bound into
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) (rbgo).

Run under rbgo:

```sh
rbgo examples/ostruct_usage.rb
```

| File | Shows |
| --- | --- |
| [`ostruct_usage.rb`](ostruct_usage.rb) | Build from a Hash, dynamic readers/writers, `[]`/`[]=`, `to_h`, `respond_to?`, `each_pair`, `delete_field`, `inspect`, and `==`. |
