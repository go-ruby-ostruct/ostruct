package ostruct

import (
	"errors"
	"testing"
)

func TestNewAndGet(t *testing.T) {
	o := New(Pair{"name", "John"}, Pair{"age", 70})
	if got := o.Get("name"); got != "John" {
		t.Fatalf("Get name = %v", got)
	}
	if got := o.Get(Symbol("age")); got != 70 {
		t.Fatalf("Get age = %v", got)
	}
	if got := o.Get("missing"); got != nil {
		t.Fatalf("Get missing = %v, want nil", got)
	}
	if o.Len() != 2 {
		t.Fatalf("Len = %d", o.Len())
	}
}

func TestNewEmptyAndNilSeed(t *testing.T) {
	if o := New(); o.Len() != 0 {
		t.Fatalf("empty Len = %d", o.Len())
	}
}

func TestSetDefinesAndOverwrites(t *testing.T) {
	o := New()
	if got := o.Set("x", 1); got != 1 {
		t.Fatalf("Set returns %v", got)
	}
	o.Set("y", 2)
	if got := o.Set("x", 9); got != 9 { // overwrite returns value
		t.Fatalf("overwrite returns %v", got)
	}
	// overwrite keeps position
	members := o.Members()
	if len(members) != 2 || members[0] != "x" || members[1] != "y" {
		t.Fatalf("members after overwrite = %v", members)
	}
	if o.Get("x") != 9 {
		t.Fatalf("x = %v", o.Get("x"))
	}
}

func TestIndexAccessors(t *testing.T) {
	o := New()
	if got := o.SetIndex("a", 5); got != 5 {
		t.Fatalf("SetIndex returns %v", got)
	}
	if got := o.Index("a"); got != 5 {
		t.Fatalf("Index = %v", got)
	}
	if got := o.Index(Symbol("a")); got != 5 {
		t.Fatalf("Index symbol = %v", got)
	}
}

func TestRespondToField(t *testing.T) {
	o := New(Pair{"a", 1})
	if !o.RespondToField("a") {
		t.Fatal("a should respond")
	}
	if !o.RespondToField(Symbol("a")) {
		t.Fatal("a (sym) should respond")
	}
	if o.RespondToField("b") {
		t.Fatal("b should not respond")
	}
}

func TestMembersIsCopy(t *testing.T) {
	o := New(Pair{"a", 1}, Pair{"b", 2})
	m := o.Members()
	m[0] = "zzz"
	if o.Members()[0] != "a" {
		t.Fatal("Members must return a copy")
	}
}

func TestToHOrderAndKeys(t *testing.T) {
	o := New(Pair{"name", "John"}, Pair{"age", 70})
	h := o.ToH()
	if len(h) != 2 {
		t.Fatalf("ToH len = %d", len(h))
	}
	if h[0].Key != Symbol("name") || h[0].Value != "John" {
		t.Fatalf("ToH[0] = %+v", h[0])
	}
	if h[1].Key != Symbol("age") || h[1].Value != 70 {
		t.Fatalf("ToH[1] = %+v", h[1])
	}
}

func TestEachPair(t *testing.T) {
	o := New(Pair{"a", 1}, Pair{"b", 2}, Pair{"c", 3})
	var keys []Symbol
	o.EachPair(func(k Symbol, v any) bool {
		keys = append(keys, k)
		return true
	})
	if len(keys) != 3 || keys[0] != "a" || keys[2] != "c" {
		t.Fatalf("EachPair order = %v", keys)
	}
	// early stop
	var seen int
	o.EachPair(func(k Symbol, v any) bool {
		seen++
		return k != "b"
	})
	if seen != 2 {
		t.Fatalf("early stop saw %d", seen)
	}
}

func TestDig(t *testing.T) {
	inner := New(Pair{"c", 1})
	o := New(Pair{"a", inner}, Pair{"flat", 5})

	// single key
	if v, err := o.Dig("flat"); err != nil || v != 5 {
		t.Fatalf("Dig flat = %v, %v", v, err)
	}
	// nested OpenStruct
	if v, err := o.Dig("a", "c"); err != nil || v != 1 {
		t.Fatalf("Dig a,c = %v, %v", v, err)
	}
	// missing first key -> nil (multi-key)
	if v, err := o.Dig("nope", "c"); err != nil || v != nil {
		t.Fatalf("Dig nope,c = %v, %v", v, err)
	}
	// single missing key -> nil
	if v, err := o.Dig("nope"); err != nil || v != nil {
		t.Fatalf("Dig nope = %v, %v", v, err)
	}
}

func TestDigZeroArgs(t *testing.T) {
	o := New(Pair{"a", 1})
	_, err := o.Dig()
	var ae *ArgumentError
	if !errors.As(err, &ae) {
		t.Fatalf("Dig() err = %v", err)
	}
	if ae.Message != "wrong number of arguments (given 0, expected 1+)" {
		t.Fatalf("ArgumentError msg = %q", ae.Message)
	}
}

func TestDigTypeError(t *testing.T) {
	o := New(Pair{"a", 1})
	_, err := o.Dig("a", "b")
	var te *TypeError
	if !errors.As(err, &te) {
		t.Fatalf("Dig into Integer err = %v", err)
	}
	if te.Message != "Integer does not have #dig method" {
		t.Fatalf("TypeError msg = %q", te.Message)
	}
}

// diggerStub implements Digger to exercise the Digger branch of Dig.
type diggerStub struct{ result any }

func (d diggerStub) Dig(keys ...any) (any, error) { return d.result, nil }

func TestDigDelegatesToDigger(t *testing.T) {
	o := New(Pair{"a", diggerStub{result: 42}})
	v, err := o.Dig("a", "b", "c")
	if err != nil || v != 42 {
		t.Fatalf("Dig via Digger = %v, %v", v, err)
	}
}

func TestDeleteField(t *testing.T) {
	o := New(Pair{"a", 1}, Pair{"b", 2}, Pair{"c", 3})
	v, err := o.DeleteField("b")
	if err != nil || v != 2 {
		t.Fatalf("DeleteField b = %v, %v", v, err)
	}
	if o.RespondToField("b") {
		t.Fatal("b should be gone")
	}
	m := o.Members()
	if len(m) != 2 || m[0] != "a" || m[1] != "c" {
		t.Fatalf("members after delete = %v", m)
	}
	// nil-valued field still deletable, returns nil
	o.Set("d", nil)
	dv, err := o.DeleteField("d")
	if err != nil || dv != nil {
		t.Fatalf("DeleteField d = %v, %v", dv, err)
	}
}

func TestDeleteFieldAbsent(t *testing.T) {
	o := New()
	_, err := o.DeleteField("x")
	var ne *NameError
	if !errors.As(err, &ne) {
		t.Fatalf("DeleteField absent err = %v", err)
	}
	if ne.Message != "no field 'x' in #<OpenStruct>" {
		t.Fatalf("NameError msg = %q", ne.Message)
	}
	// non-empty struct -> message includes inspect
	o2 := New(Pair{"a", 1})
	_, err = o2.DeleteField("x")
	ne = nil
	if !errors.As(err, &ne) || ne.Message != "no field 'x' in #<OpenStruct a=1>" {
		t.Fatalf("NameError msg = %v", err)
	}
}

func TestEqual(t *testing.T) {
	a := New(Pair{"name", "John"})
	b := New(Pair{"name", "John"})
	if !a.Equal(b) || !a.Eql(b) {
		t.Fatal("equal structs must be ==/eql?")
	}
	// order-independent equality
	c := New(Pair{"x", 1}, Pair{"y", 2})
	d := New(Pair{"y", 2}, Pair{"x", 1})
	if !c.Equal(d) {
		t.Fatal("equality is order-independent")
	}
	// different length
	if a.Equal(New(Pair{"name", "John"}, Pair{"k", 1})) {
		t.Fatal("different size must differ")
	}
	// same length, different key
	if c.Equal(New(Pair{"x", 1}, Pair{"z", 2})) {
		t.Fatal("different keys must differ")
	}
	// same keys, different value
	if c.Equal(New(Pair{"x", 1}, Pair{"y", 99})) {
		t.Fatal("different values must differ")
	}
	// not an OpenStruct
	if a.Equal("nope") {
		t.Fatal("non-OpenStruct must differ")
	}
	// typed nil
	var nilOS *OpenStruct
	if a.Equal(nilOS) {
		t.Fatal("nil *OpenStruct must differ")
	}
}

func TestEqualNestedAndNoncomparable(t *testing.T) {
	// nested OpenStruct values compared structurally
	a := New(Pair{"n", New(Pair{"x", 1})})
	b := New(Pair{"n", New(Pair{"x", 1})})
	if !a.Equal(b) {
		t.Fatal("nested OpenStructs must be equal")
	}
	// non-comparable values (slices) compared via fmt fallback
	c := New(Pair{"arr", []any{1, 2}})
	d := New(Pair{"arr", []any{1, 2}})
	if !c.Equal(d) {
		t.Fatal("slice values should compare equal via fallback")
	}
	e := New(Pair{"arr", []any{1, 3}})
	if c.Equal(e) {
		t.Fatal("differing slices must differ")
	}
	// nil vs non-nil value at same key
	f := New(Pair{"k", nil})
	g := New(Pair{"k", 1})
	if f.Equal(g) || g.Equal(f) {
		t.Fatal("nil vs non-nil must differ")
	}
	// nil == nil at same key
	h := New(Pair{"k", nil})
	if !f.Equal(h) {
		t.Fatal("nil == nil must be equal")
	}
}

func TestInspect(t *testing.T) {
	cases := []struct {
		o    *OpenStruct
		want string
	}{
		{New(), "#<OpenStruct>"},
		{New(Pair{"name", "John"}, Pair{"age", 70}), `#<OpenStruct name="John", age=70>`},
		{New(Pair{"n", nil}, Pair{"t", true}, Pair{"f", false}), "#<OpenStruct n=nil, t=true, f=false>"},
		{New(Pair{"s", `x"y`}), `#<OpenStruct s="x\"y">`},
		{New(Pair{"sym", Symbol("foo")}), "#<OpenStruct sym=:foo>"},
		{New(Pair{"arr", []any{1, 2}}), "#<OpenStruct arr=[1, 2]>"},
		{New(Pair{"flt", 1.5}), "#<OpenStruct flt=1.5>"},
	}
	for _, c := range cases {
		if got := c.o.Inspect(); got != c.want {
			t.Errorf("Inspect = %q, want %q", got, c.want)
		}
		if got := c.o.String(); got != c.want {
			t.Errorf("String (to_s) = %q, want %q", got, c.want)
		}
	}
}

func TestToSym(t *testing.T) {
	if ToSym("a") != Symbol("a") {
		t.Fatal("string -> sym")
	}
	if ToSym(Symbol("b")) != Symbol("b") {
		t.Fatal("sym -> sym")
	}
	if ToSym(stringerKey("c")) != Symbol("c") {
		t.Fatal("stringer -> sym")
	}
	if ToSym(42) != Symbol("42") {
		t.Fatal("other -> sym via Sprint")
	}
}

type stringerKey string

func (s stringerKey) String() string { return string(s) }

func TestInspectValueScalars(t *testing.T) {
	cases := []struct {
		v    any
		want string
	}{
		{nil, "nil"},
		{true, "true"},
		{false, "false"},
		{Symbol("foo"), ":foo"},
		{"hi\n\t\r\\", `"hi\n\t\r\\"`},
		{int(1), "1"},
		{int8(2), "2"},
		{int16(3), "3"},
		{int32(4), "4"},
		{int64(5), "5"},
		{uint(6), "6"},
		{uint8(7), "7"},
		{uint16(8), "8"},
		{uint32(9), "9"},
		{uint64(10), "10"},
		{float32(2.5), "2.5"},
		{float64(3.0), "3.0"},
		{float64(3.25), "3.25"},
		{[]any{1, "x", nil}, `[1, "x", nil]`},
		{[]Pair{{Symbol("a"), 1}, {"b", 2}}, `{:a => 1, "b" => 2}`},
	}
	for _, c := range cases {
		if got := InspectValue(c.v); got != c.want {
			t.Errorf("InspectValue(%#v) = %q, want %q", c.v, got, c.want)
		}
	}
}

// inspectorStub exercises the Inspector branch.
type inspectorStub struct{}

func (inspectorStub) Inspect() string { return "<<custom>>" }

func TestInspectValueUsesInspector(t *testing.T) {
	if got := InspectValue(inspectorStub{}); got != "<<custom>>" {
		t.Fatalf("Inspector branch = %q", got)
	}
	// and through OpenStruct.Inspect
	o := New(Pair{"k", inspectorStub{}})
	if got := o.Inspect(); got != "#<OpenStruct k=<<custom>>>" {
		t.Fatalf("OpenStruct uses Inspector = %q", got)
	}
}

// fallback type with no special handling -> fmt.Sprint
type weird struct{ n int }

func TestInspectValueFallback(t *testing.T) {
	got := InspectValue(weird{n: 7})
	if got != "{7}" {
		t.Fatalf("fallback inspect = %q", got)
	}
}

func TestClassName(t *testing.T) {
	cases := []struct {
		v    any
		want string
	}{
		{nil, "NilClass"},
		{true, "TrueClass"},
		{false, "FalseClass"},
		{Symbol("x"), "Symbol"},
		{"s", "String"},
		{1.0, "Float"},
		{float32(1), "Float"},
		{42, "Integer"},
		{[]any{}, "Integer"}, // default bucket
	}
	for _, c := range cases {
		if got := ClassName(c.v); got != c.want {
			t.Errorf("ClassName(%#v) = %q, want %q", c.v, got, c.want)
		}
	}
}

// classerStub implements the RubyClassName hook used by ClassName.
type classerStub struct{}

func (classerStub) RubyClassName() string { return "MyClass" }

func TestClassNameClasser(t *testing.T) {
	if got := ClassName(classerStub{}); got != "MyClass" {
		t.Fatalf("Classer branch = %q", got)
	}
	// and the Dig TypeError uses it
	o := New(Pair{"a", classerStub{}})
	_, err := o.Dig("a", "b")
	if err == nil || err.Error() != "MyClass does not have #dig method" {
		t.Fatalf("Dig TypeError via Classer = %v", err)
	}
}

func TestErrorTypes(t *testing.T) {
	if (&NameError{"n"}).Error() != "n" {
		t.Fatal("NameError")
	}
	if (&TypeError{"t"}).Error() != "t" {
		t.Fatal("TypeError")
	}
	if (&ArgumentError{"a"}).Error() != "a" {
		t.Fatal("ArgumentError")
	}
}

func TestInspectFloatScientific(t *testing.T) {
	// a value that FormatFloat renders with 'e' should not get a trailing .0
	if got := InspectValue(1e21); got != "1e+21" {
		t.Fatalf("scientific float = %q", got)
	}
}

func TestIsComparableViaEqual(t *testing.T) {
	// Symbol values are comparable and must compare by ==.
	a := New(Pair{"k", Symbol("x")})
	b := New(Pair{"k", Symbol("x")})
	if !a.Equal(b) {
		t.Fatal("Symbol values equal")
	}
	if a.Equal(New(Pair{"k", Symbol("y")})) {
		t.Fatal("differing Symbol values differ")
	}
}
