// Package ostruct is a pure-Go (CGO=0), MRI-4.0.5-faithful core for Ruby's
// OpenStruct (require "ostruct").
//
// It provides the data structure underneath OpenStruct: an ordered attribute
// table keyed by Symbol, together with the accessors, conversions, comparison,
// and inspection that MRI exposes. The dynamic method_missing get/set glue and
// respond_to_missing? — the part that turns an arbitrary method name into a
// table read or write at call time — stays in the host runtime (rbgo) and is
// implemented in terms of this table (Get/Set/RespondToField).
//
// The table preserves insertion order: to_h, each_pair, and inspect all walk
// keys in the order they were first defined, matching MRI.
//
// Values are held as opaque any. So that Inspect and dig can match MRI without
// depending on a particular value model, the package renders values through the
// Inspector and Digger interfaces (the host's Ruby values implement these), and
// falls back to a built-in renderer covering the common Ruby scalar/collection
// shapes for deterministic, ruby-free testing.
package ostruct
