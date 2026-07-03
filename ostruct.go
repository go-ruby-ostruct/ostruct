package ostruct

import (
	"fmt"
	"strconv"
	"strings"
)

// Symbol is the key type of an OpenStruct table. MRI keys every OpenStruct
// attribute by Symbol; strings handed to [OpenStruct.Get], [OpenStruct.Set],
// and the bracket accessors are interned to Symbol via [ToSym].
type Symbol string

// ToSym converts a name to a [Symbol]. A [Symbol] is returned unchanged; a
// string is interned. This mirrors MRI's `name.to_sym` on the bracket and
// accessor paths.
func ToSym(name any) Symbol {
	switch n := name.(type) {
	case Symbol:
		return n
	case string:
		return Symbol(n)
	case fmt.Stringer:
		return Symbol(n.String())
	default:
		return Symbol(fmt.Sprint(name))
	}
}

// Inspector renders a value the way Ruby's Object#inspect would. The host
// runtime's Ruby values implement it so [OpenStruct.Inspect] reproduces MRI
// byte-for-byte; values that do not implement it are rendered by [InspectValue].
type Inspector interface {
	Inspect() string
}

// Digger is the dig protocol: a value that can itself be dug into (Array, Hash,
// Struct, or a nested OpenStruct in MRI). [OpenStruct.Dig] delegates the
// remaining keys to it, matching Ruby's `obj.dig(*keys)`.
type Digger interface {
	Dig(keys ...any) (any, error)
}

// NameError is raised by [OpenStruct.DeleteField] for an absent field. Its
// message matches MRI: `no field 'NAME' in #<OpenStruct ...>`.
type NameError struct{ Message string }

func (e *NameError) Error() string { return e.Message }

// TypeError is raised by [OpenStruct.Dig] when an intermediate value does not
// implement the dig protocol. Its message matches MRI: `CLASS does not have
// #dig method`.
type TypeError struct{ Message string }

func (e *TypeError) Error() string { return e.Message }

// ArgumentError is raised by [OpenStruct.Dig] when called with no keys, matching
// MRI's `wrong number of arguments (given 0, expected 1+)`.
type ArgumentError struct{ Message string }

func (e *ArgumentError) Error() string { return e.Message }

// OpenStruct is the ordered attribute table backing Ruby's OpenStruct. Keys are
// Symbols; insertion order is preserved across to_h, each_pair, and inspect.
//
// The table stores its entries as an insertion-ordered slice of [Pair]s (each
// Key a Symbol) alongside a Symbol→position index. Keeping the values *in* the
// ordered slice — rather than in a separate map keyed by the ordered keys — lets
// [OpenStruct.ToH] serialise with a single slice copy and no per-key hashing,
// while random access ([OpenStruct.Get]/[OpenStruct.Set]) stays a single map
// probe.
//
// The zero value is not ready for use; construct with [New].
type OpenStruct struct {
	pairs []Pair         // entries in insertion order; each Key is a Symbol
	index map[Symbol]int // Symbol → position of its entry in pairs
}

// New builds an OpenStruct seeded from hash, whose keys are interned to Symbols
// in iteration order. A nil hash yields an empty struct. This is MRI's
// `OpenStruct.new(hash)`.
//
// The seed is given as ordered key/value Pairs so that insertion order — which
// MRI preserves from the source hash — is deterministic. Pass no pairs for an
// empty struct.
func New(hash ...Pair) *OpenStruct {
	o := &OpenStruct{
		pairs: make([]Pair, 0, len(hash)),
		index: make(map[Symbol]int, len(hash)),
	}
	for _, p := range hash {
		o.Set(p.Key, p.Value)
	}
	return o
}

// Pair is one ordered key/value entry used to seed [New] and returned by
// [OpenStruct.ToH]/[OpenStruct.EachPair]. Key may be a Symbol or string.
type Pair struct {
	Key   any
	Value any
}

// Get returns the value of field name (Symbol or string), or nil if the field
// is not defined — MRI's reader for an undefined attribute returns nil.
func (o *OpenStruct) Get(name any) any {
	if i, ok := o.index[ToSym(name)]; ok {
		return o.pairs[i].Value
	}
	return nil
}

// Set defines (or overwrites) field name with value, returning value. A new
// field is appended to the insertion order; overwriting an existing field keeps
// its position. This is MRI's writer (`o.name = value` / `o[name] = value`).
func (o *OpenStruct) Set(name, value any) any {
	k := ToSym(name)
	if i, ok := o.index[k]; ok {
		o.pairs[i].Value = value
		return value
	}
	o.index[k] = len(o.pairs)
	o.pairs = append(o.pairs, Pair{Key: k, Value: value})
	return value
}

// Index is the `[]` accessor: it reads field name (Symbol or string).
func (o *OpenStruct) Index(name any) any { return o.Get(name) }

// SetIndex is the `[]=` accessor: it writes field name (Symbol or string) and
// returns value.
func (o *OpenStruct) SetIndex(name, value any) any { return o.Set(name, value) }

// RespondToField reports whether field name is defined. The host's
// respond_to_missing? combines this with its own `name end_with? "="` rule for
// writers; this core answers only "is this attribute present?".
func (o *OpenStruct) RespondToField(name any) bool {
	_, ok := o.index[ToSym(name)]
	return ok
}

// Len returns the number of defined fields.
func (o *OpenStruct) Len() int { return len(o.pairs) }

// Members returns the field names (Symbols) in insertion order — MRI's
// `members`/`to_h.keys`.
func (o *OpenStruct) Members() []Symbol {
	out := make([]Symbol, len(o.pairs))
	for i, p := range o.pairs {
		out[i] = p.Key.(Symbol)
	}
	return out
}

// ToH returns the table as ordered key/value [Pair]s with Symbol keys in
// insertion order — MRI's `to_h` (whose Hash preserves that order).
//
// Because the entries are already stored in insertion order, this is a single
// slice copy: no per-key hash probe is needed to rebuild the order. A fresh
// slice is returned (never the internal backing array), so callers may mutate
// the result exactly as Ruby's `to_h` hands back an independent Hash.
func (o *OpenStruct) ToH() []Pair {
	out := make([]Pair, len(o.pairs))
	copy(out, o.pairs)
	return out
}

// EachPair calls fn for every field in insertion order, stopping early if fn
// returns false. It mirrors MRI's `each_pair` (the host wraps it to yield to a
// Ruby block or return an Enumerator).
func (o *OpenStruct) EachPair(fn func(key Symbol, value any) bool) {
	for _, p := range o.pairs {
		if !fn(p.Key.(Symbol), p.Value) {
			return
		}
	}
}

// Dig walks keys: the first key reads this struct's field, and any remaining
// keys are delegated to that value's dig protocol ([Digger], or a nested
// *OpenStruct). With no keys it returns an *ArgumentError; a key whose value is
// missing yields nil; an intermediate value that cannot be dug into yields a
// *TypeError. This is MRI's `dig(*keys)`.
func (o *OpenStruct) Dig(keys ...any) (any, error) {
	if len(keys) == 0 {
		return nil, &ArgumentError{Message: "wrong number of arguments (given 0, expected 1+)"}
	}
	v := o.Get(keys[0])
	if len(keys) == 1 {
		return v, nil
	}
	if v == nil {
		return nil, nil
	}
	rest := keys[1:]
	switch d := v.(type) {
	case *OpenStruct:
		return d.Dig(rest...)
	case Digger:
		return d.Dig(rest...)
	default:
		return nil, &TypeError{Message: ClassName(v) + " does not have #dig method"}
	}
}

// DeleteField removes field name, returning its prior value. It raises a
// *NameError if the field is not defined — MRI's `delete_field`.
func (o *OpenStruct) DeleteField(name any) (any, error) {
	k := ToSym(name)
	i, ok := o.index[k]
	if !ok {
		return nil, &NameError{Message: "no field '" + string(k) + "' in " + o.Inspect()}
	}
	v := o.pairs[i].Value
	o.pairs = append(o.pairs[:i], o.pairs[i+1:]...)
	delete(o.index, k)
	// Entries after the removed slot shifted down one; re-index them.
	for j := i; j < len(o.pairs); j++ {
		o.index[o.pairs[j].Key.(Symbol)] = j
	}
	return v, nil
}

// Equal reports OpenStruct equality the way MRI's `==`/`eql?` do: other must be
// an *OpenStruct (subclasses included, since MRI uses is_a?) and the two tables
// must be equal as ordered Symbol→value maps. Values compare by Go ==, with a
// nil-safe fallback for non-comparable values via fmt.
func (o *OpenStruct) Equal(other any) bool {
	ot, ok := other.(*OpenStruct)
	if !ok || ot == nil {
		return false
	}
	if len(o.pairs) != len(ot.pairs) {
		return false
	}
	for _, p := range o.pairs {
		j, present := ot.index[p.Key.(Symbol)]
		if !present || !valueEqual(p.Value, ot.pairs[j].Value) {
			return false
		}
	}
	return true
}

// Eql is an alias of [OpenStruct.Equal] for MRI's `eql?`, which OpenStruct
// defines identically to `==`.
func (o *OpenStruct) Eql(other any) bool { return o.Equal(other) }

// Inspect renders the struct as MRI does: `#<OpenStruct k=v, ...>` with fields
// in insertion order and each value rendered by its [Inspector] (or
// [InspectValue]); an empty struct renders as `#<OpenStruct>`.
func (o *OpenStruct) Inspect() string {
	if len(o.pairs) == 0 {
		return "#<OpenStruct>"
	}
	var b strings.Builder
	b.WriteString("#<OpenStruct ")
	for i, p := range o.pairs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(string(p.Key.(Symbol)))
		b.WriteByte('=')
		b.WriteString(InspectValue(p.Value))
	}
	b.WriteByte('>')
	return b.String()
}

// String is the `to_s` alias of [OpenStruct.Inspect].
func (o *OpenStruct) String() string { return o.Inspect() }

func valueEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if ao, ok := a.(*OpenStruct); ok {
		return ao.Equal(b)
	}
	if isComparable(a) && isComparable(b) {
		return a == b
	}
	return fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b)
}

func isComparable(v any) bool {
	switch v.(type) {
	case bool, string, Symbol,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return true
	default:
		return false
	}
}

// ClassName returns the Ruby class name MRI would use in a #dig TypeError. Hosts
// expose it via a Classer; built-in Go shapes map to their Ruby class.
func ClassName(v any) string {
	if c, ok := v.(interface{ RubyClassName() string }); ok {
		return c.RubyClassName()
	}
	switch v.(type) {
	case nil:
		return "NilClass"
	case bool:
		if v.(bool) {
			return "TrueClass"
		}
		return "FalseClass"
	case Symbol:
		return "Symbol"
	case string:
		return "String"
	case float32, float64:
		return "Float"
	default:
		return "Integer"
	}
}

// InspectValue renders v as Ruby's Object#inspect would. A value implementing
// [Inspector] supplies its own text; otherwise the built-in renderer covers the
// common Ruby scalar and collection shapes (nil, true/false, Integer, Float,
// String, Symbol, and []any/[]Pair as Array/Hash) for deterministic, ruby-free
// rendering.
func InspectValue(v any) string {
	if iv, ok := v.(Inspector); ok {
		return iv.Inspect()
	}
	switch t := v.(type) {
	case nil:
		return "nil"
	case bool:
		return strconv.FormatBool(t)
	case Symbol:
		return ":" + string(t)
	case string:
		return inspectString(t)
	case int:
		return strconv.Itoa(t)
	case int8:
		return strconv.FormatInt(int64(t), 10)
	case int16:
		return strconv.FormatInt(int64(t), 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint8:
		return strconv.FormatUint(uint64(t), 10)
	case uint16:
		return strconv.FormatUint(uint64(t), 10)
	case uint32:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float32:
		return inspectFloat(float64(t))
	case float64:
		return inspectFloat(t)
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = InspectValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []Pair:
		parts := make([]string, len(t))
		for i, p := range t {
			parts[i] = inspectHashKey(p.Key) + " => " + InspectValue(p.Value)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprint(v)
	}
}

func inspectHashKey(k any) string {
	if s, ok := k.(Symbol); ok {
		return ":" + string(s)
	}
	return InspectValue(k)
}

func inspectFloat(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}

func inspectString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
